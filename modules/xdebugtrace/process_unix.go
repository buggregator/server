//go:build !windows

package xdebugtrace

import (
	"errors"
	"syscall"
)

// processAlive reports whether a process with pid exists (EPERM: it does,
// owned by another user).
func processAlive(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}
