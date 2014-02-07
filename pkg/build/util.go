package build

import (
	"crypto/rand"
	"crypto/sha1"
	"fmt"
	"io"
)

// createUID is a helper function that will
// create a random, unique identifier.
func createUID() string {
	c := sha1.New()
	r := createRandom()
	io.WriteString(c, string(r))
	s := fmt.Sprintf("%x", c.Sum(nil))
	return "drone-" + s[0:10]
}

// createRandom creates a random block of bytes
// that we can use to generate unique identifiers.
func createRandom() []byte {
	k := make([]byte, sha1.BlockSize)
	if _, err := io.ReadFull(rand.Reader, k); err != nil {
		return nil
	}
	return k
}
