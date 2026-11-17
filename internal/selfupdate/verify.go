package selfupdate

import (
	"bufio"
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
)

// releaseKey is the base64 ed25519 public key whose private half signs
// checksums.txt in release.yml (secret RELEASE_SIGNING_KEY). A var only so a
// test build can sign its own fake release (-ldflags -X); releases keep it.
var releaseKey = "bFthyZgjx8k7TUu4ZekM3sa0zSxSRzOTaX2BfxaUH2I="

var (
	errNotSigned    = errors.New("the release is not signed (no checksums.txt.sig)")
	errBadSignature = errors.New("the signature of checksums.txt does not match the built-in release key")
)

func publicKey() ed25519.PublicKey {
	b, err := base64.StdEncoding.DecodeString(releaseKey)
	if err != nil || len(b) != ed25519.PublicKeySize {
		panic("selfupdate: bad built-in release key")
	}
	return ed25519.PublicKey(b)
}

// verifySignature checks the raw ed25519 signature of checksums.txt
// (openssl pkeyutl -sign -rawin).
func verifySignature(key ed25519.PublicKey, sums, sig []byte) error {
	if len(sig) == 0 {
		return errNotSigned
	}
	if len(sig) != ed25519.SignatureSize || !ed25519.Verify(key, sums, sig) {
		return errBadSignature
	}
	return nil
}

// checksumFor finds the sha256 of name in a sha256sum listing.
func checksumFor(sums []byte, name string) ([]byte, error) {
	sc := bufio.NewScanner(bytes.NewReader(sums))
	for sc.Scan() {
		f := strings.Fields(sc.Text())
		if len(f) != 2 || strings.TrimPrefix(f[1], "*") != name {
			continue
		}
		sum, err := hex.DecodeString(f[0])
		if err != nil || len(sum) != 32 {
			return nil, errors.New("checksums.txt: bad sha256 for " + name)
		}
		return sum, nil
	}
	return nil, errors.New("checksums.txt has no line for " + name)
}
