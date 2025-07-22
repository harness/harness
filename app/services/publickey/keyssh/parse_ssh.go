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

package keyssh

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
	return KeyInfo{
		PublicKey:  key,
		KeyComment: "",
	}
}

func Parse(keyData []byte) (KeyInfo, error) {
	publicKey, comment, _, _, err := gossh.ParseAuthorizedKey(keyData)
	if err != nil {
		return KeyInfo{}, errors.InvalidArgument("invalid SSH key data: %s" + err.Error())
	}

	keyType := publicKey.Type()

	// explicitly disallowed
	if slices.Contains(DisallowedTypes, keyType) {
		return KeyInfo{}, errors.InvalidArgument("keys of type %s are not allowed", keyType)
	}

	// only allowed
	if !slices.Contains(AllowedTypes, keyType) {
		return KeyInfo{}, errors.InvalidArgument("allowed key types are %v", AllowedTypes)
	}

	return KeyInfo{
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

type KeyInfo struct {
	PublicKey  gossh.PublicKey
	KeyComment string
}

func (key KeyInfo) Matches(s string) bool {
	otherKey, _, _, _, err := gossh.ParseAuthorizedKey([]byte(s))
	if err != nil {
		return false
	}

	return key.matchesKey(otherKey)
}

func (key KeyInfo) matchesKey(otherKey gossh.PublicKey) bool {
	return ssh.KeysEqual(key.PublicKey, otherKey)
}

func (key KeyInfo) Fingerprint() string {
	sum := sha256.New()
	sum.Write(key.PublicKey.Marshal())
	return "SHA256:" + base64.RawStdEncoding.EncodeToString(sum.Sum(nil))
}

func (key KeyInfo) Type() string {
	return key.PublicKey.Type()
}

func (key KeyInfo) Scheme() enum.PublicKeyScheme {
	return enum.PublicKeySchemeSSH
}

func (key KeyInfo) Comment() string {
	return key.KeyComment
}

func (key KeyInfo) ValidFrom() *int64 {
	return nil // SSH keys do not have validity period
}

func (key KeyInfo) ValidTo() *int64 {
	return nil // SSH keys do not have validity period
}

func (key KeyInfo) Identities() []types.Identity {
	return nil // SSH keys do not have identities
}

func (key KeyInfo) RevocationReason() *enum.RevocationReason {
	return nil // SSH keys do not have revocations
}

func (key KeyInfo) Metadata() json.RawMessage {
	return json.RawMessage("{}")
}

func (key KeyInfo) KeyIDs() []string {
	return nil // SSH keys do not have subkeys
}

func (key KeyInfo) CompromisedIDs() []string {
	return nil // SSH keys do not have a revocation reason
}
