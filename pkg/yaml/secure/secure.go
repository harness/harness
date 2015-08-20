package secure

import (
	"crypto/rsa"
	"crypto/sha256"
	"hash"

	"github.com/drone/drone/Godeps/_workspace/src/gopkg.in/yaml.v2"

	"github.com/drone/drone/pkg/utils/sshutil"
)

// Parse parses and returns the secure section of the
// yaml file as plaintext parameters.
func Parse(privateKeyPEM, repoHash, raw string) (map[string]string, error) {
	params, err := parseSecure(raw)
	if err != nil {
		return nil, err
	}

	hasher := ToHash(repoHash)
	privKey := sshutil.UnMarshalPrivateKey([]byte(privateKeyPEM))

	err = DecryptMap(hasher, privKey, params)
	return params, err
}

// DecryptMap decrypts values of a map of named parameters
// from base64 to decrypted strings.
func DecryptMap(hasher hash.Hash, privKey *rsa.PrivateKey, params map[string]string) error {
	var err error

	for name, encrypted := range params {
		params[name], err = sshutil.Decrypt(hasher, privKey, encrypted)
		if err != nil {
			return err
		}
	}
	return nil
}

// EncryptMap encrypts values of a map of named parameters
func EncryptMap(hasher hash.Hash, pubKey *rsa.PublicKey, params map[string]string) error {
	var err error

	for name, value := range params {
		params[name], err = sshutil.Encrypt(hasher, pubKey, value)
		if err != nil {
			return err
		}
	}
	return nil
}

// parseSecure is helper function to parse the Secure data from
// the raw yaml file.
func parseSecure(raw string) (map[string]string, error) {
	data := struct {
		Secure map[string]string
	}{}
	err := yaml.Unmarshal([]byte(raw), &data)

	return data.Secure, err
}

// ToHash is helper function to generate Hash of given string
func ToHash(key string) hash.Hash {
	hasher := sha256.New()
	hasher.Write([]byte(key))
	hasher.Reset()
	return hasher
}
