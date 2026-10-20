//go:build windows

package privileged

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"

	"devlok/internal/devlokerr"
	"devlok/internal/platform"
)

const (
	seeMaskNoCloseProcess = 0x00000040
	seeMaskNoAsync        = 0x00000100
	swHide                = 0
	infinite              = 0xFFFFFFFF
)

type shellExecuteInfo struct {
	cbSize       uint32
	fMask        uint32
	hwnd         windows.Handle
	lpVerb       *uint16
	lpFile       *uint16
	lpParameters *uint16
	lpDirectory  *uint16
	nShow        int32
	hInstApp     windows.Handle
	lpIDList     unsafe.Pointer
	lpClass      *uint16
	hkeyClass    windows.Handle
	dwHotKey     uint32
	hIcon        windows.Handle
	hProcess     windows.Handle
}

var shellExecuteEx = windows.NewLazySystemDLL("shell32.dll").NewProc("ShellExecuteExW")

func SyncHosts(domains []string) error {
	return run("hosts-sync", domains)
}

func RemoveHosts() error {
	return run("hosts-remove", nil)
}

func run(operation string, domains []string) error {
	helper, err := helperPath()
	if err != nil {
		return err
	}
	args := []string{operation}
	if operation == "hosts-sync" {
		state, err := platform.Current().StateDir()
		if err != nil {
			return err
		}
		payload := filepath.Join(state, "state", "hosts-domains.txt")
		if err := os.MkdirAll(filepath.Dir(payload), 0o700); err != nil {
			return err
		}
		if err := os.WriteFile(payload, []byte(strings.Join(domains, "\n")+"\n"), 0o600); err != nil {
			return devlokerr.Wrap(devlokerr.CodePermissionDenied, err, "Payload hosts tidak dapat ditulis.", "")
		}
		args = append(args, payload)
	}
	params := make([]string, 0, len(args))
	for _, a := range args {
		params = append(params, windowsQuote(a))
	}
	if err := shellRunAs(helper, strings.Join(params, " ")); err != nil {
		return err
	}
	return nil
}

func shellRunAs(file, params string) error {
	verb, err := windows.UTF16PtrFromString("runas")
	if err != nil {
		return err
	}
	filePtr, err := windows.UTF16PtrFromString(file)
	if err != nil {
		return err
	}
	paramsPtr, err := windows.UTF16PtrFromString(params)
	if err != nil {
		return err
	}
	sei := shellExecuteInfo{
		fMask:        seeMaskNoCloseProcess | seeMaskNoAsync,
		lpVerb:       verb,
		lpFile:       filePtr,
		lpParameters: paramsPtr,
		nShow:        swHide,
	}
	sei.cbSize = uint32(unsafe.Sizeof(sei))
	ret, _, callErr := shellExecuteEx.Call(uintptr(unsafe.Pointer(&sei)))
	if ret == 0 {
		if callErr == windows.ERROR_CANCELLED {
			return devlokerr.New(devlokerr.CodeCanceled,
				"Update hosts dibatalkan.",
				"Setujui prompt UAC agar domain lokal dapat ditambahkan ke hosts.")
		}
		msg := callErr.Error()
		if detail := helperLogTail(); detail != "" {
			msg += ": " + detail
		}
		return devlokerr.New(devlokerr.CodeElevationRequired,
			"Hosts membutuhkan izin Administrator.",
			"Setujui prompt UAC. "+msg)
	}
	if sei.hProcess == 0 {
		return devlokerr.New(devlokerr.CodeElevationRequired,
			"Hosts membutuhkan izin Administrator.",
			"Helper hosts tidak mengembalikan proses. Jalankan ulang aplikasi.")
	}
	defer windows.CloseHandle(sei.hProcess)
	if _, err := windows.WaitForSingleObject(sei.hProcess, infinite); err != nil {
		return devlokerr.Wrap(devlokerr.CodeTimeout, err, "Menunggu helper hosts timeout.", "")
	}
	var code uint32
	if err := windows.GetExitCodeProcess(sei.hProcess, &code); err != nil {
		return devlokerr.Wrap(devlokerr.CodeInternal, err, "Kode keluar helper hosts tidak dapat dibaca.", "")
	}
	if code != 0 {
		msg := fmt.Sprintf("Helper hosts gagal (exit %d).", code)
		if detail := helperLogTail(); detail != "" {
			msg += " " + detail
		}
		return devlokerr.New(devlokerr.CodeElevationRequired, "Hosts tidak dapat diperbarui.", msg)
	}
	return nil
}

func helperLogTail() string {
	state, err := platform.Current().StateDir()
	if err != nil {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(state, "logs", "hosts.log"))
	if err != nil {
		return ""
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) > 3 {
		lines = lines[len(lines)-3:]
	}
	return strings.Join(lines, " | ")
}

func helperPath() (string, error) {
	var candidates []string
	if self, err := os.Executable(); err == nil {
		dir := filepath.Dir(self)
		candidates = append(candidates,
			filepath.Join(dir, "dl-helper.exe"),
			filepath.Join(dir, "..", "..", "build", "bin", "dl-helper.exe"),
		)
	}
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates,
			filepath.Join(cwd, "build", "bin", "dl-helper.exe"),
			filepath.Join(cwd, "desktop", "build", "bin", "dl-helper.exe"),
		)
	}
	for _, p := range candidates {
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			return p, nil
		}
	}
	return "", devlokerr.New(devlokerr.CodeDependencyMissing, "dl-helper.exe tidak ditemukan.", "Build dengan go build -o build/bin/dl-helper.exe ./cmd/dl-helper.")
}

func windowsQuote(s string) string {
	if s == "" {
		return `""`
	}
	if !strings.ContainsAny(s, " \t\"") {
		return s
	}
	var b strings.Builder
	b.WriteByte('"')
	backslashes := 0
	for _, r := range s {
		switch r {
		case '\\':
			backslashes++
		case '"':
			b.WriteString(strings.Repeat("\\", backslashes*2+1))
			b.WriteRune(r)
			backslashes = 0
		default:
			b.WriteString(strings.Repeat("\\", backslashes))
			backslashes = 0
			b.WriteRune(r)
		}
	}
	b.WriteString(strings.Repeat("\\", backslashes*2))
	b.WriteByte('"')
	return b.String()
}
