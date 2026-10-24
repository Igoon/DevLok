package version

import "runtime"

var (
	Version = "0.2.7"
	Commit  = "none"
	Date    = "unknown"
)

type Info struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	Date      string `json:"date"`
	GoVersion string `json:"go_version"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
}

func Current() Info {
	return Info{
		Version:   Version,
		Commit:    Commit,
		Date:      Date,
		GoVersion: runtime.Version(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
	}
}

func String() string {
	i := Current()
	return "dl " + i.Version + " (" + i.Commit + ", " + i.Date + ") " + i.OS + "/" + i.Arch + " " + i.GoVersion
}
