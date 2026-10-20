//go:build windows

package doctor

import (
	"os"
	"path/filepath"
	"strconv"

	"devlok/internal/platform"
	"golang.org/x/sys/windows/registry"
)

const webView2GUID = `{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}`

func platformChecks(plat platform.Platform) []Check {
	return []Check{
		checkWindowsVersion(),
		checkWebView2(),
		checkVCRuntime(),
	}
}

func checkWindowsVersion() Check {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion`, registry.QUERY_VALUE)
	if err != nil {
		return Check{Name: "windows", Status: StatusWarn, Message: "Versi Windows tidak dapat dibaca."}
	}
	defer k.Close()
	build, _, _ := k.GetStringValue("CurrentBuildNumber")
	display, _, _ := k.GetStringValue("DisplayVersion")
	n, err := strconv.Atoi(build)
	if err != nil {
		return Check{Name: "windows", Status: StatusWarn, Message: "Build Windows tidak dikenali: " + build}
	}
	msg := "Windows build " + build
	if display != "" {
		msg += " (" + display + ")"
	}
	if n < 19045 {
		return Check{Name: "windows", Status: StatusWarn, Message: msg, Hint: "DevLok menargetkan Windows 10 22H2 (build 19045) atau Windows 11."}
	}
	return Check{Name: "windows", Status: StatusOK, Message: msg}
}

func checkWebView2() Check {
	candidates := []struct {
		root registry.Key
		path string
	}{
		{registry.LOCAL_MACHINE, `SOFTWARE\WOW6432Node\Microsoft\EdgeUpdate\Clients\` + webView2GUID},
		{registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\EdgeUpdate\Clients\` + webView2GUID},
		{registry.CURRENT_USER, `Software\Microsoft\EdgeUpdate\Clients\` + webView2GUID},
	}
	for _, c := range candidates {
		k, err := registry.OpenKey(c.root, c.path, registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		pv, _, err := k.GetStringValue("pv")
		_ = k.Close()
		if err == nil && pv != "" {
			return Check{Name: "webview2", Status: StatusOK, Message: "WebView2 " + pv}
		}
	}
	return Check{Name: "webview2", Status: StatusWarn, Message: "WebView2 Runtime tidak terdeteksi.", Hint: "Installer DevLok akan membuka halaman unduh WebView2, atau gunakan winget install Microsoft.EdgeWebView2Runtime."}
}

func checkVCRuntime() Check {
	root := os.Getenv("SystemRoot")
	if root == "" {
		root = `C:\Windows`
	}
	system32 := filepath.Join(root, "System32")
	required := []string{"msvcp140.dll", "vcruntime140.dll"}
	for _, name := range required {
		if _, err := os.Stat(filepath.Join(system32, name)); err != nil {
			return Check{Name: "vcruntime", Status: StatusWarn, Message: "Visual C++ Runtime tidak lengkap (" + name + ").", Hint: "Instal Microsoft Visual C++ Redistributable 2015-2022 x64."}
		}
	}
	return Check{Name: "vcruntime", Status: StatusOK, Message: "Visual C++ Runtime x64 terdeteksi."}
}
