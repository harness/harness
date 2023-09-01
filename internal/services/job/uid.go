// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

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
