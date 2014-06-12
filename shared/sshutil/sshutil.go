package sshutil

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"

	"code.google.com/p/go.crypto/ssh"
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
func MarshalPublicKey(pubkey *rsa.PublicKey) string {
	pk, err := ssh.NewPublicKey(pubkey)
	if err != nil {
		return ""
	}

	return string(ssh.MarshalAuthorizedKey(pk))
}

// helper function that marshalls an RSA Private Key to
// a PEM encoded file.
func MarshalPrivateKey(privkey *rsa.PrivateKey) string {
	privateKeyMarshaled := x509.MarshalPKCS1PrivateKey(privkey)
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Headers: nil, Bytes: privateKeyMarshaled})
	return string(privateKeyPEM)
}
