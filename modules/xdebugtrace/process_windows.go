//go:build windows

package xdebugtrace

import (
	"errors"
	"os"
	"syscall"
)

// processAlive reports whether a process with pid exists: FindProcess
// opens a handle on Windows and fails for a pid that is gone. Access
// denied (an elevated process seen from a plain one) means it exists.
func processAlive(pid int) bool {
	p, err := os.FindProcess(pid)
	if err != nil {
		return errors.Is(err, syscall.ERROR_ACCESS_DENIED)
	}
	p.Release()
	return true
}
