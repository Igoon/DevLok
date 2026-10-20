package ports

import (
	"fmt"
	"net"
)

type Status struct {
	Port        int    `json:"port"`
	Available   bool   `json:"available"`
	PID         int    `json:"pid,omitempty"`
	Process     string `json:"process,omitempty"`
	Path        string `json:"path,omitempty"`
	Description string `json:"description,omitempty"`
}

type Owner struct {
	PID     int
	Process string
	Path    string
}

func Check(port int) Status {
	st := Status{Port: port}
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	ln, err := net.Listen("tcp", addr)
	if err == nil {
		_ = ln.Close()
		if !listeningOn(port) {
			st.Available = true
			st.Description = "tersedia"
			return st
		}
	}
	st.Description = "dipakai"
	if owner, ok := ownerOf(port); ok {
		st.PID = owner.PID
		st.Process = owner.Process
		st.Path = owner.Path
	}
	return st
}

func CheckAll(list []int) []Status {
	out := make([]Status, 0, len(list))
	for _, p := range list {
		out = append(out, Check(p))
	}
	return out
}

func Pick(candidates ...int) (int, error) {
	for _, p := range candidates {
		if Check(p).Available {
			return p, nil
		}
	}
	return 0, fmt.Errorf("tidak ada port yang tersedia: %v", candidates)
}
