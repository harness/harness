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

package types

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
)

const maxTruncatedLen = 8

// MaskSecret is a wrapper to store decrypted secrets in memory. This is help to prevent them
// from getting prints in logs and fmt.
type MaskSecret struct {
	value       string
	hashedValue string
}

func NewMaskSecret(val string) *MaskSecret {
	hash := sha256.New()
	hash.Write([]byte(val))

	hashedValueStr := fmt.Sprintf("%x", hash.Sum(nil))

	return &MaskSecret{
		value:       val,
		hashedValue: hashedValueStr[:maxTruncatedLen],
	}
}

// Value returns the unmasked value of the MaskSecret.
// Use cautiously to avoid exposing sensitive data.
func (s *MaskSecret) Value() string {
	if s == nil {
		return ""
	}
	return s.value
}

func (s *MaskSecret) String() string {
	if s == nil {
		return ""
	}

	return s.hashedValue
}

func (s *MaskSecret) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.value)
}

func (s *MaskSecret) UnmarshalJSON(data []byte) error {
	var input string
	if err := json.Unmarshal(data, &input); err != nil {
		return err
	}

	s.value = input
	return nil
}
