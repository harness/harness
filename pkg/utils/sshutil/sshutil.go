package sshutil

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"

	"github.com/drone/drone/Godeps/_workspace/src/code.google.com/p/go.crypto/ssh"
)

const (
	RSA_BITS     = 2048 // Default number of bits in an RSA key
	RSA_BITS_MIN = 768  // Minimum number of bits in an RSA key
)

// helper function to generate an RSA Private Key.
func GeneratePrivateKey() (*rsa.PrivateKey, error) {
	return rsa.GenerateKey(rand.Reader, RSA_BITS)
}

// helper function that marshalls an RSA Public Key to an SSH
// .authorized_keys format
func MarshalPublicKey(pubkey *rsa.PublicKey) []byte {
	pk, err := ssh.NewPublicKey(pubkey)
	if err != nil {
		return []byte{}
	}

	return ssh.MarshalAuthorizedKey(pk)
}

// helper function that marshalls an RSA Private Key to
// a PEM encoded file.
func MarshalPrivateKey(privkey *rsa.PrivateKey) []byte {
	privateKeyMarshaled := x509.MarshalPKCS1PrivateKey(privkey)
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Headers: nil, Bytes: privateKeyMarshaled})
	return privateKeyPEM
}

// helper function to encrypt a plain-text string using
// an RSA public key.
func Encrypt(pubkey *rsa.PublicKey, msg string) ([]byte, error) {
	return rsa.EncryptPKCS1v15(rand.Reader, pubkey, []byte(msg))
}

// helper function to encrypt a plain-text string using
// an RSA public key.
func Decrypt(privkey *rsa.PrivateKey, secret string) (string, error) {
	msg, err := rsa.DecryptPKCS1v15(rand.Reader, privkey, []byte(secret))
	return string(msg), err
}
