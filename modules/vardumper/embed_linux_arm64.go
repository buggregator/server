//go:build linux && arm64

package vardumper

import _ "embed"

//go:embed bin/vardumper-parser-linux-arm64
var phpBinary []byte
