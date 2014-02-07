package model

import (
	"crypto/md5"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strings"
	"unicode"

	"code.google.com/p/go.crypto/ssh"
	"code.google.com/p/go.text/unicode/norm"
	"github.com/dchest/uniuri"
)

var (
	lat = []*unicode.RangeTable{unicode.Letter, unicode.Number}
	nop = []*unicode.RangeTable{unicode.Mark, unicode.Sk, unicode.Lm}
)

// helper function to create a Gravatar Hash
// for the given Email address.
func createGravatar(email string) string {
	email = strings.ToLower(strings.TrimSpace(email))
	hash := md5.New()
	hash.Write([]byte(email))
	return fmt.Sprintf("%x", hash.Sum(nil))
}

// helper function to create a Slug for the
// given string of text.
func createSlug(s string) string {
	buf := make([]rune, 0, len(s))
	dash := false
	for _, r := range norm.NFKD.String(s) {
		switch {
		// unicode 'letters' like mandarin characters pass through
		case unicode.IsOneOf(lat, r):
			buf = append(buf, unicode.ToLower(r))
			dash = true
		case unicode.IsOneOf(nop, r):
			// skip
		case dash:
			buf = append(buf, '-')
			dash = false
		}
	}
	if i := len(buf) - 1; i >= 0 && buf[i] == '-' {
		buf = buf[:i]
	}
	return string(buf)
}

// helper function to create a random 40-byte
// Token that is URL-friendly.
func createToken() string {
	return uniuri.NewLen(40)
}

// -----------------------------------------------------------------------------
// SSH Functions

const (
	RSA_BITS     = 2048 // Default number of bits in an RSA key
	RSA_BITS_MIN = 768  // Minimum number of bits in an RSA key
)

// helper function to generate an RSA Private Key.
func generatePrivateKey() (*rsa.PrivateKey, error) {
	return rsa.GenerateKey(rand.Reader, RSA_BITS)
}

// helper function that marshalls an RSA Public Key to an SSH
// .authorized_keys format
func marshalPublicKey(pubkey *rsa.PublicKey) string {
	pk, err := ssh.NewPublicKey(pubkey)
	if err != nil {
		return ""
	}

	return string(ssh.MarshalAuthorizedKey(pk))
}

// helper function that marshalls an RSA Private Key to
// a PEM encoded file.
func marshalPrivateKey(privkey *rsa.PrivateKey) string {
	privateKeyMarshaled := x509.MarshalPKCS1PrivateKey(privkey)
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Headers: nil, Bytes: privateKeyMarshaled})
	return string(privateKeyPEM)
}
