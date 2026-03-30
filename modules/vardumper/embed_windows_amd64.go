//go:build windows && amd64

package vardumper

// phpBinary is empty on Windows — static-php does not provide micro.sfx for Windows.
// The VarDumper module will fail to start and should be disabled on this platform.
var phpBinary []byte
