//go:build !windows

package selfupdate

import (
	"os"
	"syscall"
)

// execSelf replaces the process with exe, same arguments and environment.
func execSelf(exe string) error {
	return syscall.Exec(exe, os.Args, os.Environ())
}
