//go:build windows && amd64

package vardumper

import _ "embed"

//go:embed bin/vardumper-parser-windows-amd64.exe
var phpBinary []byte
