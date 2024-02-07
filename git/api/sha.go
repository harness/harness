package api

import (
	"encoding/hex"
	"strings"
)

// NilSHA defines empty git SHA
const NilSHA = "0000000000000000000000000000000000000000"

// EmptyTreeSHA is the SHA of an empty tree
const EmptyTreeSHA = "4b825dc642cb6eb9a060e54bf8d69288fbee4904"

// SHA a git commit name
type SHA []byte

// String returns a string representation of the SHA
func (s SHA) String() string {
	return hex.EncodeToString(s)
}

// IsZero returns whether this SHA1 is all zeroes
func (s SHA) IsZero() bool {
	return len(s) == 0
}

// NewSHA creates a new SHA from a s string.
func NewSHA(s string) (SHA, error) {
	s = strings.TrimSpace(s)
	b, err := hex.DecodeString(s)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func MustNewSHA(s string) SHA {
	sha, _ := NewSHA(s)
	return sha
}
