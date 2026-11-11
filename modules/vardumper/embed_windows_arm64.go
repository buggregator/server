//go:build windows && arm64

package vardumper

import _ "embed"

//go:embed bin/vardumper-parser-windows-arm64.exe
var phpBinary []byte
