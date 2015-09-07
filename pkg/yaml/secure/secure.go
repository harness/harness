package secure

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"

	"github.com/square/go-jose"
)

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
