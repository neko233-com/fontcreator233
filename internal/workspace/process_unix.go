//go:build !windows

package workspace

import (
	"os"
	"syscall"
)

// processAlive uses signal 0, which performs existence/permission checks
// without delivering a signal.
func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = p.Signal(syscall.Signal(0))
	return err == nil || os.IsPermission(err)
}
