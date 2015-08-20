package sshutil

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"hash"

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

// UnMarshalPrivateKey is a helper function that unmarshals a PEM
// bytes to an RSA Private Key
func UnMarshalPrivateKey(privateKeyPEM []byte) *rsa.PrivateKey {
	derBlock, _ := pem.Decode(privateKeyPEM)
	privateKey, err := x509.ParsePKCS1PrivateKey(derBlock.Bytes)

	if err != nil {
		return nil
	}
	return privateKey
}

// Encrypt is helper function to encrypt a plain-text string using
// an RSA public key.
func Encrypt(hash hash.Hash, pubkey *rsa.PublicKey, msg string) (string, error) {
	src, err := rsa.EncryptOAEP(hash, rand.Reader, pubkey, []byte(msg), nil)

	return base64.StdEncoding.EncodeToString(src), err
}

// Decrypt is helper function to encrypt a plain-text string using
// an RSA public key.
func Decrypt(hash hash.Hash, privkey *rsa.PrivateKey, secret string) (string, error) {
	decoded, err := base64.StdEncoding.DecodeString(secret)
	if err != nil {
		return "", err
	}

	out, err := rsa.DecryptOAEP(hash, rand.Reader, privkey, decoded, nil)

	return string(out), err
}
