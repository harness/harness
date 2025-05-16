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
	"encoding/base64"
	"slices"

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/types/enum"

	"github.com/gliderlabs/ssh"
	gossh "golang.org/x/crypto/ssh"
)

func FromSSH(key gossh.PublicKey) KeyInfo {
	return sshKeyInfo{
		Key: key,
	}
}

func parseSSH(keyData []byte) (sshKeyInfo, string, error) {
	publicKey, comment, _, _, err := gossh.ParseAuthorizedKey(keyData)
	if err != nil {
		return sshKeyInfo{}, "", errors.InvalidArgument("invalid SSH key data: %s" + err.Error())
	}

	keyType := publicKey.Type()

	// explicitly disallowed
	if slices.Contains(DisallowedTypes, keyType) {
		return sshKeyInfo{}, "", errors.InvalidArgument("keys of type %s are not allowed", keyType)
	}

	// only allowed
	if !slices.Contains(AllowedTypes, keyType) {
		return sshKeyInfo{}, "", errors.InvalidArgument("allowed key types are %v", AllowedTypes)
	}

	return sshKeyInfo{
		Key: publicKey,
	}, comment, nil
}

var AllowedTypes = []string{
	gossh.KeyAlgoRSA,
	gossh.KeyAlgoECDSA256,
	gossh.KeyAlgoECDSA384,
	gossh.KeyAlgoECDSA521,
	gossh.KeyAlgoED25519,
	gossh.KeyAlgoSKECDSA256,
	gossh.KeyAlgoSKED25519,
}

var DisallowedTypes = []string{
	gossh.KeyAlgoDSA,
}

type sshKeyInfo struct {
	Key gossh.PublicKey
}

func (key sshKeyInfo) Matches(s string) bool {
	otherKey, _, _, _, err := gossh.ParseAuthorizedKey([]byte(s))
	if err != nil {
		return false
	}

	return key.matchesKey(otherKey)
}

func (key sshKeyInfo) matchesKey(otherKey gossh.PublicKey) bool {
	return ssh.KeysEqual(key.Key, otherKey)
}

func (key sshKeyInfo) Fingerprint() string {
	sum := sha256.New()
	sum.Write(key.Key.Marshal())
	return "SHA256:" + base64.RawStdEncoding.EncodeToString(sum.Sum(nil))
}

func (key sshKeyInfo) Type() string {
	return key.Key.Type()
}

func (key sshKeyInfo) Scheme() enum.PublicKeyScheme {
	return enum.PublicKeySchemeSSH
}
