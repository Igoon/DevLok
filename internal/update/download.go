package update

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"devlok/internal/devlokerr"
)

const maxAppDownload = 512 << 20

type DownloadProgress func(downloaded, total int64)

func Download(ctx context.Context, rel *AppUpdate, dir string) (string, error) {
	return DownloadWithProgress(ctx, rel, dir, nil)
}

func DownloadWithProgress(ctx context.Context, rel *AppUpdate, dir string, progress DownloadProgress) (string, error) {
	if rel == nil {
		return "", devlokerr.New(devlokerr.CodeInvalidArgument, "Tidak ada rilis aplikasi untuk diunduh.", "")
	}
	u, err := url.Parse(rel.URL)
	if err != nil || (u.Scheme != "https" && !(u.Scheme == "http" && loopbackHost(u.Hostname()))) {
		return "", devlokerr.New(devlokerr.CodeInvalidArgument, "URL rilis aplikasi harus HTTPS.", "")
	}
	if len(rel.SHA256) != 64 {
		return "", devlokerr.New(devlokerr.CodeInvalidArgument, "SHA256 rilis aplikasi tidak valid.", "")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", devlokerr.Wrap(devlokerr.CodePermissionDenied, err, "Folder update tidak dapat dibuat.", "")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rel.URL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "DevLok-updater")
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if req.URL.Scheme == "https" || (req.URL.Scheme == "http" && loopbackHost(req.URL.Hostname())) {
				return nil
			}
			return fmt.Errorf("redirect update harus HTTPS")
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", devlokerr.Wrap(devlokerr.CodeTimeout, err, "Rilis aplikasi tidak dapat diunduh.", "")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", devlokerr.New(devlokerr.CodeStateInconsistent, "Unduhan update mengembalikan HTTP "+fmt.Sprint(resp.StatusCode), "")
	}
	name := safeName(path.Base(u.Path))
	if name == "" || name == "." {
		name = "DevLok-" + rel.Latest + ".exe"
	}
	out := filepath.Join(dir, "DevLok-"+rel.Latest+"-"+name)
	tmp := out + ".part"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o700)
	if err != nil {
		return "", devlokerr.Wrap(devlokerr.CodePermissionDenied, err, "File update tidak dapat ditulis.", "")
	}
	h := sha256.New()
	reader := io.Reader(io.LimitReader(resp.Body, maxAppDownload+1))
	if progress != nil {
		reader = &progressReader{r: reader, total: resp.ContentLength, cb: progress}
	}
	written, copyErr := io.Copy(io.MultiWriter(f, h), reader)
	closeErr := f.Close()
	if copyErr != nil || closeErr != nil {
		_ = os.Remove(tmp)
		if copyErr != nil {
			return "", devlokerr.Wrap(devlokerr.CodeStateInconsistent, copyErr, "Unduhan update gagal.", "")
		}
		return "", devlokerr.Wrap(devlokerr.CodePermissionDenied, closeErr, "File update tidak dapat diselesaikan.", "")
	}
	if written > maxAppDownload {
		_ = os.Remove(tmp)
		return "", devlokerr.New(devlokerr.CodeInvalidArgument, "Ukuran update melebihi batas 512 MB.", "")
	}
	got := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(got, rel.SHA256) {
		_ = os.Remove(tmp)
		return "", devlokerr.New(devlokerr.CodeChecksumMismatch, "Checksum update tidak cocok.", "Jangan jalankan installer; unduhan telah dihapus.")
	}
	if err := os.Rename(tmp, out); err != nil {
		_ = os.Remove(tmp)
		return "", devlokerr.Wrap(devlokerr.CodePermissionDenied, err, "File update tidak dapat diselesaikan.", "")
	}
	return out, nil
}

type progressReader struct {
	r     io.Reader
	total int64
	done  int64
	last  time.Time
	cb    DownloadProgress
}

func (p *progressReader) Read(b []byte) (int, error) {
	n, err := p.r.Read(b)
	p.done += int64(n)
	if p.cb != nil && (time.Since(p.last) > 100*time.Millisecond || err == io.EOF) {
		p.last = time.Now()
		total := p.total
		if total < 0 {
			total = 0
		}
		p.cb(p.done, total)
	}
	return n, err
}

func safeName(name string) string {
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '.', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	return b.String()
}
