package secure

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"

	"github.com/drone/drone/Godeps/_workspace/src/gopkg.in/yaml.v2"
)

// Parse parses and returns the secure section of the
// yaml file as plaintext parameters.
func Parse(key, raw string) (map[string]string, error) {
	params, err := parseSecure(raw)
	if err != nil {
		return nil, err
	}
	err = DecryptMap(key, params)
	return params, err
}

// Encrypt encrypts a string to base64 crypto using AES.
func Encrypt(key, text string) (_ string, err error) {
	plaintext := []byte(text)

	block, err := aes.NewCipher(trimKey(key))
	if err != nil {
		return
	}

	ciphertext := make([]byte, aes.BlockSize+len(plaintext))
	iv := ciphertext[:aes.BlockSize]
	if _, err = io.ReadFull(rand.Reader, iv); err != nil {
		return
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], plaintext)

	return base64.URLEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrtyps from base64 to decrypted string.
func Decrypt(key, text string) (_ string, err error) {
	ciphertext, err := base64.URLEncoding.DecodeString(text)
	if err != nil {
		return
	}

	block, err := aes.NewCipher(trimKey(key))
	if err != nil {
		return
	}

	if len(ciphertext) < aes.BlockSize {
		err = fmt.Errorf("ciphertext too short")
		return
	}
	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(ciphertext, ciphertext)

	return fmt.Sprintf("%s", ciphertext), nil
}

// DecryptMap decrypts a map of named parameters
// from base64 to decrypted string.
func DecryptMap(key string, params map[string]string) error {
	var err error
	for name, value := range params {
		params[name], err = Decrypt(key, value)
		if err != nil {
			return err
		}
	}
	return nil
}

// EncryptMap encrypts encrypts a map of string parameters
// to base64 crypto using AES.
func EncryptMap(key string, params map[string]string) error {
	var err error
	for name, value := range params {
		params[name], err = Encrypt(key, value)
		if err != nil {
			return err
		}
	}
	return nil
}

// helper function that trims a key to a maximum
// of 32 bytes to match the expected AES block size.
func trimKey(key string) []byte {
	b := []byte(key)
	if len(b) > 32 {
		b = b[:32]
	}
	return b
}

// helper function to parse the Secure data from
// the raw yaml file.
func parseSecure(raw string) (map[string]string, error) {
	data := struct {
		Secure map[string]string
	}{}
	err := yaml.Unmarshal([]byte(raw), &data)
	return data.Secure, err
}
