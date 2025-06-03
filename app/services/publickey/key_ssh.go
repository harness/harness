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
	"encoding/json"
	"slices"

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/gliderlabs/ssh"
	gossh "golang.org/x/crypto/ssh"
)

func FromSSH(key gossh.PublicKey) KeyInfo {
	return SSHKeyInfo{
		PublicKey:  key,
		KeyComment: "",
	}
}

func parseSSH(keyData []byte) (SSHKeyInfo, error) {
	publicKey, comment, _, _, err := gossh.ParseAuthorizedKey(keyData)
	if err != nil {
		return SSHKeyInfo{}, errors.InvalidArgument("invalid SSH key data: %s" + err.Error())
	}

	keyType := publicKey.Type()

	// explicitly disallowed
	if slices.Contains(DisallowedTypes, keyType) {
		return SSHKeyInfo{}, errors.InvalidArgument("keys of type %s are not allowed", keyType)
	}

	// only allowed
	if !slices.Contains(AllowedTypes, keyType) {
		return SSHKeyInfo{}, errors.InvalidArgument("allowed key types are %v", AllowedTypes)
	}

	return SSHKeyInfo{
		PublicKey:  publicKey,
		KeyComment: comment,
	}, nil
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

type SSHKeyInfo struct {
	PublicKey  gossh.PublicKey
	KeyComment string
}

func (key SSHKeyInfo) Matches(s string) bool {
	otherKey, _, _, _, err := gossh.ParseAuthorizedKey([]byte(s))
	if err != nil {
		return false
	}

	return key.matchesKey(otherKey)
}

func (key SSHKeyInfo) matchesKey(otherKey gossh.PublicKey) bool {
	return ssh.KeysEqual(key.PublicKey, otherKey)
}

func (key SSHKeyInfo) Fingerprint() string {
	sum := sha256.New()
	sum.Write(key.PublicKey.Marshal())
	return "SHA256:" + base64.RawStdEncoding.EncodeToString(sum.Sum(nil))
}

func (key SSHKeyInfo) Type() string {
	return key.PublicKey.Type()
}

func (key SSHKeyInfo) Scheme() enum.PublicKeyScheme {
	return enum.PublicKeySchemeSSH
}

func (key SSHKeyInfo) Comment() string {
	return key.KeyComment
}

func (key SSHKeyInfo) ValidFrom() *int64 {
	return nil // SSH keys do not have validity period
}

func (key SSHKeyInfo) ValidTo() *int64 {
	return nil // SSH keys do not have validity period
}

func (key SSHKeyInfo) Identities() []types.Identity {
	return nil // SSH keys do not have identities
}

func (key SSHKeyInfo) RevocationReason() *enum.RevocationReason {
	return nil
}

func (key SSHKeyInfo) Metadata() json.RawMessage {
	return json.RawMessage("{}")
}

func (key SSHKeyInfo) SubKeyIDs() []string {
	return nil
}
