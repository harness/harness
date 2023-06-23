// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package hash

// SerializeReference serializes a reference to prepare it for hashing.
func SerializeReference(ref string, sha string) []byte {
	return []byte(ref + ":" + sha)
}

// SerializeHead serializes the head to prepare it for hashing.
func SerializeHead(value string) []byte {
	return []byte("HEAD:" + value)
}
