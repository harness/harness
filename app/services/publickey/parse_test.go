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
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"strings"
	"testing"

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

func TestParseString_SSH(t *testing.T) {
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	sshPub, err := ssh.NewPublicKey(pub)
	require.NoError(t, err)
	data := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(sshPub))) + " me@laptop"

	key, err := ParseString(data, nil)
	require.NoError(t, err)
	require.Equal(t, enum.PublicKeySchemeSSH, key.Scheme())
	require.Equal(t, "me@laptop", key.Comment())
	require.True(t, key.Matches(data))
}

func TestParseString_PGP(t *testing.T) {
	entity, err := openpgp.NewEntity("Jane", "", "jane@example.com", &packet.Config{
		Algorithm: packet.PubKeyAlgoRSA, RSABits: 2048,
	})
	require.NoError(t, err)

	var buf bytes.Buffer
	w, err := armor.Encode(&buf, openpgp.PublicKeyType, nil)
	require.NoError(t, err)
	require.NoError(t, entity.Serialize(w))
	require.NoError(t, w.Close())
	data := strings.TrimSpace(buf.String())

	key, err := ParseString(data, &types.Principal{Email: "jane@example.com"})
	require.NoError(t, err)
	require.Equal(t, enum.PublicKeySchemePGP, key.Scheme())
	require.Equal(t, "RSA", key.Type())

	_, err = ParseString(data, &types.Principal{Email: "someone@else.com"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "PGP")
}

func TestParseString_Errors(t *testing.T) {
	_, err := ParseString("", nil)
	require.Error(t, err)

	_, err = ParseString("definitely not a key", nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "SSH")

	// a PGP header without the matching footer is treated as an SSH key and rejected
	_, err = ParseString("-----BEGIN PGP PUBLIC KEY BLOCK-----\nabc", nil)
	require.Error(t, err)
}
