//go:build windows

package ports

import (
	"encoding/binary"
	"fmt"
	"path/filepath"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const tcpTableOwnerPidAll = 5

type mibTcpRowOwnerPid struct {
	State      uint32
	LocalAddr  uint32
	LocalPort  uint32
	RemoteAddr uint32
	RemotePort uint32
	OwningPID  uint32
}

var procGetExtendedTcpTable = syscall.NewLazyDLL("iphlpapi.dll").NewProc("GetExtendedTcpTable")

func readTCPTable() ([]mibTcpRowOwnerPid, error) {
	var size uint32
	for attempt := 0; attempt < 3; attempt++ {
		ret, _, _ := procGetExtendedTcpTable.Call(
			0,
			uintptr(unsafe.Pointer(&size)),
			0,
			uintptr(syscall.AF_INET),
			tcpTableOwnerPidAll,
			0,
		)
		if ret != uintptr(syscall.ERROR_INSUFFICIENT_BUFFER) && ret != 0 {
			return nil, fmt.Errorf("GetExtendedTcpTable size: %d", ret)
		}
		if size == 0 {
			return nil, nil
		}
		buf := make([]byte, size)
		ret, _, _ = procGetExtendedTcpTable.Call(
			uintptr(unsafe.Pointer(&buf[0])),
			uintptr(unsafe.Pointer(&size)),
			0,
			uintptr(syscall.AF_INET),
			tcpTableOwnerPidAll,
			0,
		)
		if ret == uintptr(syscall.ERROR_INSUFFICIENT_BUFFER) {
			continue
		}
		if ret != 0 {
			return nil, fmt.Errorf("GetExtendedTcpTable: %d", ret)
		}
		count := *(*uint32)(unsafe.Pointer(&buf[0]))
		if count == 0 {
			return nil, nil
		}
		src := unsafe.Slice((*mibTcpRowOwnerPid)(unsafe.Pointer(&buf[4])), count)
		out := make([]mibTcpRowOwnerPid, count)
		copy(out, src)
		return out, nil
	}
	return nil, fmt.Errorf("GetExtendedTcpTable: tabel berubah terus")
}

func rowPort(row mibTcpRowOwnerPid) int {
	portBytes := (*[4]byte)(unsafe.Pointer(&row.LocalPort))
	return int(binary.BigEndian.Uint16(portBytes[:2]))
}

func listeningOn(port int) bool {
	rows, err := readTCPTable()
	if err != nil {
		return false
	}
	for _, row := range rows {
		if row.State == 2 && rowPort(row) == port {
			return true
		}
	}
	return false
}

func ownerOf(port int) (Owner, bool) {
	rows, err := readTCPTable()
	if err != nil {
		return Owner{}, false
	}
	for _, row := range rows {
		if row.State != 2 || rowPort(row) != port {
			continue
		}
		pid := int(row.OwningPID)
		name, path := processInfo(pid)
		return Owner{PID: pid, Process: name, Path: path}, true
	}
	return Owner{}, false
}

func processInfo(pid int) (string, string) {
	if pid <= 0 {
		return "", ""
	}
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return "", ""
	}
	defer windows.CloseHandle(h)
	buf := make([]uint16, windows.MAX_LONG_PATH)
	size := uint32(len(buf))
	if err := windows.QueryFullProcessImageName(h, 0, &buf[0], &size); err != nil {
		return "", ""
	}
	full := windows.UTF16ToString(buf[:size])
	if full == "" {
		return "", ""
	}
	return filepath.Base(full), full
}
