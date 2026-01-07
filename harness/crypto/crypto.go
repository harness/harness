// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// GenerateHMACSHA256 generates a new HMAC using SHA256 as hash function.
func GenerateHMACSHA256(data []byte, key []byte) (string, error) {
	h := hmac.New(sha256.New, key)

	// write all data into hash
	_, err := h.Write(data)
	if err != nil {
		return "", fmt.Errorf("failed to write data into hash: %w", err)
	}

	// sum hash to final value
	macBytes := h.Sum(nil)

	// encode MAC as hexadecimal
	return hex.EncodeToString(macBytes), nil
}

func IsShaEqual(key1, key2 string) bool {
	return hmac.Equal([]byte(key1), []byte(key2))
}
