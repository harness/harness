// Copyright 2023 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package encrypt

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"
)

// Aesgcm provides an encrypter that uses the aesgcm encryption
// algorithm.
type Aesgcm struct {
	block  cipher.Block
	Compat bool
}

// Encrypt encrypts the plaintext using aesgcm.
func (e *Aesgcm) Encrypt(plaintext string) ([]byte, error) {
	gcm, err := cipher.NewGCM(e.block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	_, err = io.ReadFull(rand.Reader, nonce)
	if err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, []byte(plaintext), nil), nil
}

// Decrypt decrypts the ciphertext using aesgcm.
func (e *Aesgcm) Decrypt(ciphertext []byte) (string, error) {
	gcm, err := cipher.NewGCM(e.block)
	if err != nil {
		return "", err
	}

	if len(ciphertext) < gcm.NonceSize() {
		// if the decryption utility is running in compatibility
		// mode, it will return the ciphertext as plain text if
		// decryption fails. This should be used when running the
		// database in mixed-mode, where there is a mix of encrypted
		// and unencrypted content.
		if e.Compat {
			return string(ciphertext), nil
		}
		return "", errors.New("malformed ciphertext")
	}

	plaintext, err := gcm.Open(nil,
		ciphertext[:gcm.NonceSize()],
		ciphertext[gcm.NonceSize():],
		nil,
	)
	// if the decryption utility is running in compatibility
	// mode, it will return the ciphertext as plain text if
	// decryption fails. This should be used when running the
	// database in mixed-mode, where there is a mix of encrypted
	// and unencrypted content.
	if err != nil && e.Compat {
		return string(ciphertext), nil
	}
	return string(plaintext), err
}

// New provides a new aesgcm encrypter
func New(key string, compat bool) (Encrypter, error) {
	if len(key) != 32 {
		return nil, errKeySize
	}
	b := []byte(key)
	block, err := aes.NewCipher(b)
	if err != nil {
		return nil, err
	}
	return &Aesgcm{block: block, Compat: compat}, nil
}
