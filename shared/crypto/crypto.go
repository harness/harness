package crypto

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io"

	"code.google.com/p/go.crypto/ssh"
	"github.com/square/go-jose"
)

const (
	RSA_BITS     = 2048 // Default number of bits in an RSA key
	RSA_BITS_MIN = 768  // Minimum number of bits in an RSA key
)

// standard characters allowed in token string.
var chars = []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789")

// default token length
var length = 32

// Rand generates a 32-bit random string.
func Rand() string {
	b := make([]byte, length)
	r := make([]byte, length+(length/4)) // storage for random bytes.
	clen := byte(len(chars))
	maxrb := byte(256 - (256 % len(chars)))
	i := 0
	for {
		io.ReadFull(rand.Reader, r)
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
}

// helper function to generate an RSA Private Key.
func GeneratePrivateKey() (*rsa.PrivateKey, error) {
	return rsa.GenerateKey(rand.Reader, RSA_BITS)
}

// helper function that marshalls an RSA Public Key to an SSH
// .authorized_keys format
func MarshalPublicKey(public *rsa.PublicKey) []byte {
	private, err := ssh.NewPublicKey(public)
	if err != nil {
		return []byte{}
	}

	return ssh.MarshalAuthorizedKey(private)
}

// helper function that marshalls an RSA Private Key to
// a PEM encoded file.
func MarshalPrivateKey(private *rsa.PrivateKey) []byte {
	marshaled := x509.MarshalPKCS1PrivateKey(private)
	encoded := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Headers: nil, Bytes: marshaled})
	return encoded
}

// UnmarshalPrivateKey is a helper function that unmarshals a PEM
// bytes to an RSA Private Key
func UnmarshalPrivateKey(private []byte) *rsa.PrivateKey {
	decoded, _ := pem.Decode(private)
	parsed, err := x509.ParsePKCS1PrivateKey(decoded.Bytes)
	if err != nil {
		return nil
	}
	return parsed
}

// Encrypt encrypts a secret string.
func Encrypt(in, privKey string) (string, error) {
	rsaPrivKey, err := decodePrivateKey(privKey)
	if err != nil {
		return "", err
	}

	return encrypt(in, &rsaPrivKey.PublicKey)
}

// decodePrivateKey is a helper function that unmarshals a PEM
// bytes to an RSA Private Key
func decodePrivateKey(privateKey string) (*rsa.PrivateKey, error) {
	derBlock, _ := pem.Decode([]byte(privateKey))
	return x509.ParsePKCS1PrivateKey(derBlock.Bytes)
}

// encrypt encrypts a plaintext variable using JOSE with
// RSA_OAEP and A128GCM algorithms.
func encrypt(text string, pubKey *rsa.PublicKey) (string, error) {
	var encrypted string
	var plaintext = []byte(text)

	// Creates a new encrypter using defaults
	encrypter, err := jose.NewEncrypter(jose.RSA_OAEP, jose.A128GCM, pubKey)
	if err != nil {
		return encrypted, err
	}
	// Encrypts the plaintext value and serializes
	// as a JOSE string.
	object, err := encrypter.Encrypt(plaintext)
	if err != nil {
		return encrypted, err
	}
	return object.CompactSerialize()
}
