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

package job

import (
	"crypto/rand"
	"encoding/base32"
)

// UID returns unique random string with length equal to 16.
func UID() (string, error) {
	const uidSizeBytes = 10 // must be divisible by 5, the resulting string length will be uidSizeBytes/5*8

	var buf [uidSizeBytes]byte
	_, err := rand.Read(buf[:])
	if err != nil {
		return "", err
	}

	uid := base32.StdEncoding.EncodeToString(buf[:])

	return uid, nil
}
