//go:build !windows

package stack

func startSupervisor(string) error {
	return nil
}
