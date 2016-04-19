package shasum

import (
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"io"
	"strings"
)

// Check is a calculates and verifies a file checksum.
// This supports the sha1, sha256 and sha512 values.
func Check(in, checksum string) bool {
	hash, size, _ := split(checksum)

	// if a byte size is provided for the
	// Yaml file it must match.
	if size > 0 && int64(len(in)) != size {
		return false
	}

	switch len(hash) {
	case 64:
		return sha256sum(in) == hash
	case 128:
		return sha512sum(in) == hash
	case 40:
		return sha1sum(in) == hash
	case 0:
		return true // if no checksum assume valid
	}

	return false
}

func sha1sum(in string) string {
	h := sha1.New()
	io.WriteString(h, in)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func sha256sum(in string) string {
	h := sha256.New()
	io.WriteString(h, in)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func sha512sum(in string) string {
	h := sha512.New()
	io.WriteString(h, in)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func split(in string) (string, int64, string) {
	var hash string
	var name string
	var size int64

	// the checksum might be split into multiple
	// sections including the file size and name.
	switch strings.Count(in, " ") {
	case 1:
		fmt.Sscanf(in, "%s %s", &hash, &name)
	case 2:
		fmt.Sscanf(in, "%s %d %s", &hash, &size, &name)
	default:
		hash = in
	}

	return hash, size, name
}
