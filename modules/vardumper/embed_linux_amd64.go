//go:build linux && amd64

package vardumper

import _ "embed"

//go:embed bin/vardumper-parser-linux-amd64
var phpBinary []byte
