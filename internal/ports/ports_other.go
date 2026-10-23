//go:build !windows && !darwin

package ports

import (
	"net"
	"strconv"
)

func listeningOn(port int) bool {
	ln, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(port))
	if err != nil {
		return true
	}
	_ = ln.Close()
	return false
}

func ownerOf(int) (Owner, bool) {
	return Owner{}, false
}
