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
	"crypto/aes"
	"errors"
)

// indicates key size is too small.
var errKeySize = errors.New("encryption key must be 32 bytes")

// Encrypter provides database field encryption and decryption.
// Encrypted values are currently limited to strings, which is
// reflected in the interface design.
type Encrypter interface {
	Encrypt(plaintext string) ([]byte, error)
	Decrypt(ciphertext []byte) (string, error)
}

// New provides a new database field encrypter.
func New(key string) (Encrypter, error) {
	if key == "" {
		return &none{}, nil
	}
	if len(key) != 32 {
		return nil, errKeySize
	}
	b := []byte(key)
	block, err := aes.NewCipher(b)
	if err != nil {
		return nil, err
	}
	return &Aesgcm{block: block}, nil
}
