//go:build !windows

package hidden

import "os/exec"

func Hide(*exec.Cmd) {}
