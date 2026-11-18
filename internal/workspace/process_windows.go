//go:build windows

package workspace

import "os"

// processAlive reports whether pid belongs to a live process. OpenProcess
// fails for a missing PID, so FindProcess is enough on Windows.
func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	p, err := os.FindProcess(pid)
	if err != nil || p == nil {
		return false
	}
	_ = p.Release()
	return true
}
