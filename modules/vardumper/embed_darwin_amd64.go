//go:build darwin && amd64

package vardumper

import _ "embed"

//go:embed bin/vardumper-parser-darwin-amd64
var phpBinary []byte
