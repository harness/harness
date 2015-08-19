package secure

import (
	"crypto/sha256"
	"hash"

	"github.com/drone/drone/Godeps/_workspace/src/gopkg.in/yaml.v2"

	common "github.com/drone/drone/pkg/types"
	"github.com/drone/drone/pkg/utils/sshutil"
)

// Parse parses and returns the secure section of the
// yaml file as plaintext parameters.
func Parse(repo *common.Repo, raw string) (map[string]string, error) {
	params, err := parseSecure(raw)
	if err != nil {
		return nil, err
	}

	err = DecryptMap(repo, params)
	return params, err
}

// DecryptMap decrypts values of a map of named parameters
// from base64 to decrypted strings.
func DecryptMap(repo *common.Repo, params map[string]string) error {
	var err error
	hasher := toHash(repo.Hash)
	privKey := sshutil.UnMarshalPrivateKey([]byte(repo.Keys.Private))

	for name, encrypted := range params {
		params[name], err = sshutil.Decrypt(hasher, privKey, encrypted)
		if err != nil {
			return err
		}
	}
	return nil
}

// EncryptMap encrypts values of a map of named parameters
func EncryptMap(repo *common.Repo, params map[string]string) error {
	var err error
	hasher := toHash(repo.Hash)
	privKey := sshutil.UnMarshalPrivateKey([]byte(repo.Keys.Private))

	for name, value := range params {
		params[name], err = sshutil.Encrypt(hasher, &privKey.PublicKey, value)
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

// toHash is helper function to generate Hash of given string
func toHash(key string) hash.Hash {
	hasher := sha256.New()
	hasher.Write([]byte(key))
	hasher.Reset()
	return hasher
}
