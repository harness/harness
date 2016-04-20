// Package uniuri generates random strings good for use in URIs to identify
// unique objects.
//
// Example usage:
//
//	s := uniuri.New() // s is now "apHCJBl7L1OmC57n"
//
// A standard string created by New() is 16 bytes in length and consists of
// Latin upper and lowercase letters, and numbers (from the set of 62 allowed
// characters), which means that it has ~95 bits of entropy. To get more
// entropy, you can use NewLen(UUIDLen), which returns 20-byte string, giving
// ~119 bits of entropy, or any other desired length.
//
// Functions read from crypto/rand random source, and panic if they fail to
// read from it.
package uniuri

import (
	"crypto/rand"
	"io"
)

const (
	// Standard length of uniuri string to achive ~95 bits of entropy.
	StdLen = 16
	// Length of uniurl string to achive ~119 bits of entropy, closest
	// to what can be losslessly converted to UUIDv4 (122 bits).
	UUIDLen = 20
)

// Standard characters allowed in uniuri string.
var StdChars = []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789")

// New returns a new random string of the standard length, consisting of
// standard characters.
func New() string {
	return NewLenChars(StdLen, StdChars)
}

// NewLen returns a new random string of the provided length, consisting of
// standard characters.
func NewLen(length int) string {
	return NewLenChars(length, StdChars)
}

// NewLenChars returns a new random string of the provided length, consisting
// of the provided byte slice of allowed characters (maximum 256).
func NewLenChars(length int, chars []byte) string {
	b := make([]byte, length)
	r := make([]byte, length+(length/4)) // storage for random bytes.
	clen := byte(len(chars))
	maxrb := byte(256 - (256 % len(chars)))
	i := 0
	for {
		if _, err := io.ReadFull(rand.Reader, r); err != nil {
			panic("error reading from random source: " + err.Error())
		}
		for _, c := range r {
			if c >= maxrb {
				// Skip this number to avoid modulo bias.
				continue
			}
			b[i] = chars[c%clen]
			i++
			if i == length {
				return string(b)
			}
		}
	}
	panic("unreachable")
}
