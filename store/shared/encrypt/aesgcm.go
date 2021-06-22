// Copyright 2019 Drone IO, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package encrypt

import (
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
