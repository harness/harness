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

package publickey

import (
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"hash"
)

// SSHSignatureBlob describes a lightweight SSH signature format.
// https://github.com/openssh/openssh-portable/blob/V_9_9_P2/PROTOCOL.sshsig#L34
type SSHSignatureBlob struct {
	MagicPreamble [6]byte
	Version       uint32
	PublicKey     []byte
	Namespace     string
	Reserved      string
	HashAlgorithm string
	Signature     []byte
}

// SSHMessageWrapper represents SSH signed data.
// https://github.com/openssh/openssh-portable/blob/V_9_9_P2/PROTOCOL.sshsig#L81
type SSHMessageWrapper struct {
	Namespace     string
	Reserved      string
	HashAlgorithm string
	Hash          []byte
}

func SSHHash(hashAlgorithm string) (hash.Hash, error) {
	switch hashAlgorithm {
	case "sha256":
		return sha256.New(), nil
	case "sha512":
		return sha512.New(), nil
	default:
		return nil, fmt.Errorf("unsupported hash algorithm: %s", hashAlgorithm)
	}
}
