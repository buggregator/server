//go:build windows

package xdebugtrace

import "os"

// processAlive reports whether a process with pid exists: FindProcess
// opens a handle on Windows and fails for a pid that is gone.
func processAlive(pid int) bool {
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	p.Release()
	return true
}
