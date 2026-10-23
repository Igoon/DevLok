package platform

import "time"

type ProcessInfo struct {
	PID     int
	Name    string
	ExePath string
}

type Platform interface {
	Name() string
	StateDir() (string, error)
	CacheDir() (string, error)
	HostsPath() string
	ReadHosts() ([]byte, error)
	WriteHosts(data []byte) error
	TrustCA(certPath string) error
	UntrustCA(certPath string) error
	KillTree(pid int) error
	KillProcessesUnder(root string) ([]int, error)
	ProcessStartTime(pid int) (time.Time, error)
	OpenBrowser(url string) error
	OpenFolder(path string) error
	OpenTerminal(dir string, env []string) error
	OpenEditor(path string) error
	OpenApp(path string) error
	FlushDNS() error
	ExeSuffix() string
	AppSuffix() string
}

func Current() Platform {
	return newPlatform()
}
