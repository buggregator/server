//go:build darwin && arm64

package vardumper

import _ "embed"

//go:embed bin/vardumper-parser-darwin-arm64
var phpBinary []byte
