//go:build windows

package selfupdate

import "errors"

// execSelf: Windows has no exec; the user restarts the server.
func execSelf(string) error {
	return errors.New("restart the server to run the new version")
}
