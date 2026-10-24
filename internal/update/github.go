package update

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"runtime"
	"strings"
	"time"

	"devlok/internal/devlokerr"
	"devlok/internal/version"
)

const githubReleasesAPI = "https://api.github.com/repos/Igoon/DevLok/releases?per_page=20"

type githubRelease struct {
	TagName    string        `json:"tag_name"`
	Name       string        `json:"name"`
	Body       string        `json:"body"`
	Draft      bool          `json:"draft"`
	Prerelease bool          `json:"prerelease"`
	Assets     []githubAsset `json:"assets"`
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

func GitHubCheck(ctx context.Context) (*AppUpdate, error) {
	return githubCheck(ctx, githubReleasesAPI)
}

func githubCheck(ctx context.Context, apiURL string) (*AppUpdate, error) {
	u, err := url.Parse(apiURL)
	if err != nil || (u.Scheme != "https" && !(u.Scheme == "http" && loopbackHost(u.Hostname()))) {
		return nil, devlokerr.New(devlokerr.CodeInvalidArgument, "URL GitHub API tidak valid.", "")
	}
	client := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if req.URL.Scheme == "https" || (req.URL.Scheme == "http" && loopbackHost(req.URL.Hostname())) {
				return nil
			}
			return fmt.Errorf("redirect update harus HTTPS")
		},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "DevLok-updater")
	resp, err := client.Do(req)
	if err != nil {
		return nil, devlokerr.Wrap(devlokerr.CodeTimeout, err, "Rilis GitHub tidak dapat diperiksa.", "")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, devlokerr.New(devlokerr.CodeStateInconsistent, "GitHub API mengembalikan HTTP "+fmt.Sprint(resp.StatusCode), "")
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	var releases []githubRelease
	if err := json.Unmarshal(body, &releases); err != nil {
		return nil, devlokerr.Wrap(devlokerr.CodeInvalidArgument, err, "Respons rilis GitHub tidak valid.", "")
	}
	current := version.Current().Version
	var stable, pre *AppUpdate
	var stableSHA, preSHA string
	for _, rel := range releases {
		if rel.Draft {
			continue
		}
		v := strings.TrimPrefix(strings.TrimSpace(rel.TagName), "v")
		if v == "" || Compare(v, current) <= 0 {
			continue
		}
		installer, shaURL := releaseInstaller(rel)
		if installer == "" || shaURL == "" {
			continue
		}
		up := &AppUpdate{
			Current: current,
			Latest:  v,
			URL:     installer,
			Notes:   trimNotes(rel.Body),
		}
		if rel.Prerelease {
			if pre == nil || Compare(v, pre.Latest) > 0 {
				pre = up
				preSHA = shaURL
			}
		} else if stable == nil || Compare(v, stable.Latest) > 0 {
			stable = up
			stableSHA = shaURL
		}
	}
	chosen, shaURL := stable, stableSHA
	if chosen == nil && strings.Contains(current, "-") {
		chosen, shaURL = pre, preSHA
	}
	if chosen == nil {
		return nil, nil
	}
	sum, err := fetchSHA(ctx, client, shaURL)
	if err != nil {
		return nil, err
	}
	chosen.SHA256 = sum
	return chosen, nil
}

func releaseInstaller(rel githubRelease) (string, string) {
	assets := map[string]string{}
	for _, a := range rel.Assets {
		assets[a.Name] = a.BrowserDownloadURL
	}
	for _, name := range appAssetCandidates(rel.TagName) {
		if link := assets[name]; link != "" {
			return link, assets[name+".sha256"]
		}
	}
	return "", ""
}

func appAssetCandidates(tag string) []string {
	ver := strings.TrimPrefix(strings.TrimSpace(tag), "v")
	if runtime.GOOS == "darwin" {
		arch := "arm64"
		if runtime.GOARCH == "amd64" {
			arch = "intel"
		}
		return []string{"DevLok-macos-" + arch + ".dmg", "DevLok-macos-" + arch + ".zip"}
	}
	return []string{"DevLok-Setup-" + ver + ".exe"}
}

func fetchSHA(ctx context.Context, client *http.Client, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "DevLok-updater")
	resp, err := client.Do(req)
	if err != nil {
		return "", devlokerr.Wrap(devlokerr.CodeTimeout, err, "Checksum rilis tidak dapat diunduh.", "")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", devlokerr.New(devlokerr.CodeStateInconsistent, "Checksum rilis mengembalikan HTTP "+fmt.Sprint(resp.StatusCode), "")
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return "", err
	}
	fields := strings.Fields(string(data))
	if len(fields) == 0 || len(fields[0]) != 64 {
		return "", devlokerr.New(devlokerr.CodeInvalidArgument, "Checksum rilis tidak valid.", "")
	}
	return strings.ToLower(fields[0]), nil
}

func trimNotes(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 2000 {
		return s[:2000] + "..."
	}
	return s
}
