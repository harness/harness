// Copyright 2026 Harness, Inc.
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

package keypgp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const svcObjectSHA = "0000000000000000000000000000000000000001"

func svcPGPConfig() *packet.Config {
	return &packet.Config{Algorithm: packet.PubKeyAlgoEdDSA}
}

func svcNewEntity(t *testing.T, name, email string) *openpgp.Entity {
	t.Helper()
	entity, err := openpgp.NewEntity(name, "work", email, svcPGPConfig())
	require.NoError(t, err)
	return entity
}

func svcArmorPublic(t *testing.T, entity *openpgp.Entity) string {
	t.Helper()
	buf := &bytes.Buffer{}
	w, err := armor.Encode(buf, openpgp.PublicKeyType, nil)
	require.NoError(t, err)
	require.NoError(t, entity.Serialize(w))
	require.NoError(t, w.Close())
	return buf.String()
}

func svcArmorPrivate(t *testing.T, entity *openpgp.Entity) string {
	t.Helper()
	buf := &bytes.Buffer{}
	w, err := armor.Encode(buf, openpgp.PrivateKeyType, nil)
	require.NoError(t, err)
	require.NoError(t, entity.SerializePrivate(w, svcPGPConfig()))
	require.NoError(t, w.Close())
	return buf.String()
}

func svcSign(t *testing.T, entity *openpgp.Entity, content string) []byte {
	t.Helper()
	buf := &bytes.Buffer{}
	require.NoError(t, openpgp.ArmoredDetachSign(buf, entity, strings.NewReader(content), svcPGPConfig()))
	return buf.Bytes()
}

func TestSvcParse(t *testing.T) {
	alice := svcNewEntity(t, "Alice", "alice@example.com")
	armored := svcArmorPublic(t, alice)

	t.Run("valid key", func(t *testing.T) {
		key, err := Parse(strings.NewReader(armored), &types.Principal{Email: "alice@example.com"})
		require.NoError(t, err)

		assert.Equal(t, enum.PublicKeySchemePGP, key.Scheme())
		assert.Equal(t, "work", key.Comment())
		assert.Equal(t, "EdDSA", key.Type())
		assert.NotEmpty(t, key.Fingerprint())
		assert.Nil(t, key.ValidTo())
		require.NotNil(t, key.ValidFrom())
		assert.Positive(t, *key.ValidFrom())
		assert.Nil(t, key.RevocationReason())
		assert.Equal(t, []types.Identity{{Name: "Alice", Email: "alice@example.com"}}, key.Identities())
		assert.Empty(t, key.CompromisedIDs())

		ids := key.KeyIDs()
		require.NotEmpty(t, ids)
		assert.Equal(t, alice.PrimaryKey.KeyIdString(), ids[0])

		var meta EntityMetadata
		require.NoError(t, json.Unmarshal(key.Metadata(), &meta))
		assert.Equal(t, key.Fingerprint(), meta.PrimaryKey.Fingerprint)
		require.NotNil(t, meta.PrimaryIdentity)
		assert.Equal(t, "alice@example.com", meta.PrimaryIdentity.Email)
	})
	t.Run("email comparison ignores case", func(t *testing.T) {
		_, err := Parse(strings.NewReader(armored), &types.Principal{Email: "ALICE@Example.com"})
		require.NoError(t, err)
	})
	t.Run("nil principal accepts any identity", func(t *testing.T) {
		_, err := Parse(strings.NewReader(armored), nil)
		require.NoError(t, err)
	})
	t.Run("principal email mismatch", func(t *testing.T) {
		_, err := Parse(strings.NewReader(armored), &types.Principal{Email: "bob@example.com"})
		require.Error(t, err)
	})
	t.Run("private key is refused", func(t *testing.T) {
		_, err := Parse(strings.NewReader(svcArmorPrivate(t, alice)), nil)
		require.Error(t, err)
	})
	t.Run("garbage is refused", func(t *testing.T) {
		_, err := Parse(strings.NewReader("not a key"), nil)
		require.Error(t, err)
	})
	t.Run("multiple primary keys are refused", func(t *testing.T) {
		bob := svcNewEntity(t, "Bob", "bob@example.com")
		buf := &bytes.Buffer{}
		w, err := armor.Encode(buf, openpgp.PublicKeyType, nil)
		require.NoError(t, err)
		require.NoError(t, alice.Serialize(w))
		require.NoError(t, bob.Serialize(w))
		require.NoError(t, w.Close())

		_, err = Parse(buf, nil)
		require.Error(t, err)
	})
	t.Run("signing subkeys are listed", func(t *testing.T) {
		e := svcNewEntity(t, "Carol", "carol@example.com")
		require.NoError(t, e.AddSigningSubkey(svcPGPConfig()))
		key, err := Parse(strings.NewReader(svcArmorPublic(t, e)), nil)
		require.NoError(t, err)

		var meta EntityMetadata
		require.NoError(t, json.Unmarshal(key.Metadata(), &meta))
		require.NotEmpty(t, meta.SubKeys)

		ids := key.KeyIDs()
		assert.Contains(t, ids, e.Subkeys[len(e.Subkeys)-1].PublicKey.KeyIdString())
		assert.Greater(t, len(ids), 1)
	})
}

func TestSvcParse_Revocations(t *testing.T) {
	t.Run("compromised primary key", func(t *testing.T) {
		e := svcNewEntity(t, "Dave", "dave@example.com")
		require.NoError(t, e.AddSigningSubkey(svcPGPConfig()))
		require.NoError(t, e.RevokeKey(packet.KeyCompromised, "leaked", svcPGPConfig()))

		key, err := Parse(strings.NewReader(svcArmorPublic(t, e)), nil)
		require.NoError(t, err)
		require.NotNil(t, key.RevocationReason())
		assert.Equal(t, enum.RevocationReasonCompromised, *key.RevocationReason())
		require.NotNil(t, key.ValidTo(), "a revoked key must have a validity end")

		compromised := key.CompromisedIDs()
		assert.Contains(t, compromised, e.PrimaryKey.KeyIdString())
		assert.Greater(t, len(compromised), 1, "signing subkeys of a compromised key are compromised too")
	})
	t.Run("retired primary key is not compromised", func(t *testing.T) {
		e := svcNewEntity(t, "Erin", "erin@example.com")
		require.NoError(t, e.RevokeKey(packet.KeyRetired, "old", svcPGPConfig()))

		key, err := Parse(strings.NewReader(svcArmorPublic(t, e)), nil)
		require.NoError(t, err)
		require.NotNil(t, key.RevocationReason())
		assert.Equal(t, enum.RevocationReasonRetired, *key.RevocationReason())
		assert.Empty(t, key.CompromisedIDs())
	})
	t.Run("only a compromised subkey", func(t *testing.T) {
		e := svcNewEntity(t, "Frank", "frank@example.com")
		require.NoError(t, e.AddSigningSubkey(svcPGPConfig()))
		sub := &e.Subkeys[len(e.Subkeys)-1]
		require.NoError(t, e.RevokeSubkey(sub, packet.KeyCompromised, "leaked", svcPGPConfig()))

		key, err := Parse(strings.NewReader(svcArmorPublic(t, e)), nil)
		require.NoError(t, err)
		assert.Nil(t, key.RevocationReason())
		assert.Equal(t, []string{sub.PublicKey.KeyIdString()}, key.CompromisedIDs())
	})
}

func TestSvcKeyInfoMatches(t *testing.T) {
	alice := svcNewEntity(t, "Alice", "alice@example.com")
	bob := svcNewEntity(t, "Bob", "bob@example.com")
	armored := svcArmorPublic(t, alice)

	key, err := Parse(strings.NewReader(armored), nil)
	require.NoError(t, err)

	assert.True(t, key.Matches(armored))
	assert.False(t, key.Matches(svcArmorPublic(t, bob)))
	assert.False(t, key.Matches("garbage"))
}

func TestSvcGetRevocationReason(t *testing.T) {
	reason := func(r packet.ReasonForRevocation) *packet.Signature {
		return &packet.Signature{RevocationReason: &r}
	}

	assert.Nil(t, getRevocationReason(nil))

	tests := []struct {
		name string
		revs []*packet.Signature
		want enum.RevocationReason
	}{
		{"compromised", []*packet.Signature{reason(packet.KeyCompromised)}, enum.RevocationReasonCompromised},
		{"retired", []*packet.Signature{reason(packet.KeyRetired)}, enum.RevocationReasonRetired},
		{"superseded", []*packet.Signature{reason(packet.KeySuperseded)}, enum.RevocationReasonSuperseded},
		{"unspecified", []*packet.Signature{reason(packet.NoReason)}, enum.RevocationReasonUnknown},
		{"nil entries", []*packet.Signature{nil, {}}, enum.RevocationReasonUnknown},
		{
			"compromised wins",
			[]*packet.Signature{reason(packet.KeyRetired), reason(packet.KeyCompromised)},
			enum.RevocationReasonCompromised,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getRevocationReason(tt.revs)
			require.NotNil(t, got)
			assert.Equal(t, tt.want, *got)
		})
	}
}

func TestSvcPgpAlgo(t *testing.T) {
	tests := map[packet.PublicKeyAlgorithm]string{
		packet.PubKeyAlgoRSA:            "RSA",
		packet.PubKeyAlgoRSASignOnly:    "RSA",
		packet.PubKeyAlgoRSAEncryptOnly: "RSA",
		packet.PubKeyAlgoElGamal:        "ElGamal",
		packet.PubKeyAlgoDSA:            "DSA",
		packet.PubKeyAlgoECDH:           "ECDH",
		packet.PubKeyAlgoECDSA:          "ECDSA",
		packet.PubKeyAlgoEdDSA:          "EdDSA",
		packet.PubKeyAlgoX25519:         "X25519",
		packet.PubKeyAlgoX448:           "X448",
		packet.PubKeyAlgoEd25519:        "Ed25519",
		packet.PubKeyAlgoEd448:          "Ed448",
		packet.PublicKeyAlgorithm(200):  "",
	}
	for algo, want := range tests {
		assert.Equal(t, want, pgpAlgo(algo))
	}
}

func TestSvcVerifyParse(t *testing.T) {
	ctx := context.Background()
	alice := svcNewEntity(t, "Alice", "alice@example.com")
	objSHA := sha.Must(svcObjectSHA)

	t.Run("valid signature", func(t *testing.T) {
		v := &Verify{}
		require.Empty(t, v.Parse(ctx, svcSign(t, alice, "content"), objSHA))
		assert.Equal(t, alice.PrimaryKey.KeyIdString(), v.KeyID())
		assert.Equal(t, enum.PublicKeySchemePGP, v.KeyScheme())
		assert.NotEmpty(t, v.KeyFingerprint())
	})
	t.Run("garbage", func(t *testing.T) {
		assert.Equal(t, enum.GitSignatureInvalid, (&Verify{}).Parse(ctx, []byte("garbage"), objSHA))
	})
	t.Run("wrong armor type", func(t *testing.T) {
		assert.Equal(t, enum.GitSignatureInvalid, (&Verify{}).Parse(ctx, []byte(svcArmorPublic(t, alice)), objSHA))
	})
	t.Run("empty body", func(t *testing.T) {
		buf := &bytes.Buffer{}
		w, err := armor.Encode(buf, openpgp.SignatureType, nil)
		require.NoError(t, err)
		require.NoError(t, w.Close())
		assert.Equal(t, enum.GitSignatureInvalid, (&Verify{}).Parse(ctx, buf.Bytes(), objSHA))
	})
	t.Run("not a signature packet", func(t *testing.T) {
		buf := &bytes.Buffer{}
		w, err := armor.Encode(buf, openpgp.SignatureType, nil)
		require.NoError(t, err)
		require.NoError(t, alice.PrimaryKey.Serialize(w))
		require.NoError(t, w.Close())
		assert.Equal(t, enum.GitSignatureInvalid, (&Verify{}).Parse(ctx, buf.Bytes(), objSHA))
	})
}

type svcFetcher struct {
	keys  []types.PublicKey
	err   error
	lastK string
}

func (f *svcFetcher) FetchByFingerprint(
	context.Context, string, int64, []enum.PublicKeyUsage, []enum.PublicKeyScheme,
) ([]types.PublicKey, error) {
	return nil, errors.New("unexpected call")
}

func (f *svcFetcher) FetchBySubKeyID(
	_ context.Context, id string, _ int64, _ []enum.PublicKeyUsage, _ []enum.PublicKeyScheme,
) ([]types.PublicKey, error) {
	f.lastK = id
	return f.keys, f.err
}

func TestSvcVerifyKey(t *testing.T) {
	ctx := context.Background()
	alice := svcNewEntity(t, "Alice", "alice@example.com")
	v := &Verify{}
	require.Empty(t, v.Parse(ctx, svcSign(t, alice, "c"), sha.Must(svcObjectSHA)))

	f := &svcFetcher{keys: []types.PublicKey{{ID: 9}}}
	got, err := v.Key(ctx, f, 1)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, int64(9), got.ID)
	assert.Equal(t, v.KeyID(), f.lastK)

	got, err = v.Key(ctx, &svcFetcher{}, 1)
	require.NoError(t, err)
	assert.Nil(t, got)

	_, err = v.Key(ctx, &svcFetcher{err: errors.New("db down")}, 1)
	require.Error(t, err)
}

func TestSvcVerifyVerify(t *testing.T) {
	ctx := context.Background()
	alice := svcNewEntity(t, "Alice", "alice@example.com")
	bob := svcNewEntity(t, "Bob", "bob@example.com")
	const content = "signed content"
	objSHA := sha.Must(svcObjectSHA)
	aliceKey := []byte(svcArmorPublic(t, alice))

	committer := func(email string) types.Signature {
		return types.Signature{Identity: types.Identity{Name: "x", Email: email}, When: time.Now()}
	}
	parsed := func(t *testing.T, e *openpgp.Entity, signed string) *Verify {
		t.Helper()
		v := &Verify{}
		require.Empty(t, v.Parse(ctx, svcSign(t, e, signed), objSHA))
		return v
	}

	t.Run("good", func(t *testing.T) {
		res := parsed(t, alice, content).Verify(ctx, aliceKey, []byte(content), objSHA, committer("alice@example.com"))
		assert.Equal(t, enum.GitSignatureGood, res)
	})
	t.Run("committer email is case insensitive", func(t *testing.T) {
		res := parsed(t, alice, content).Verify(ctx, aliceKey, []byte(content), objSHA, committer("ALICE@example.com"))
		assert.Equal(t, enum.GitSignatureGood, res)
	})
	t.Run("committer email not in key", func(t *testing.T) {
		res := parsed(t, alice, content).Verify(ctx, aliceKey, []byte(content), objSHA, committer("eve@example.com"))
		assert.Equal(t, enum.GitSignatureBad, res)
	})
	t.Run("tampered content", func(t *testing.T) {
		res := parsed(t, alice, content).Verify(ctx, aliceKey, []byte("tampered"), objSHA, committer("alice@example.com"))
		assert.Equal(t, enum.GitSignatureInvalid, res)
	})
	t.Run("signed by another key", func(t *testing.T) {
		res := parsed(t, bob, content).Verify(ctx, aliceKey, []byte(content), objSHA, committer("alice@example.com"))
		assert.Equal(t, enum.GitSignatureBad, res)
	})
	t.Run("unreadable key ring", func(t *testing.T) {
		res := parsed(t, alice, content).Verify(ctx, []byte("junk"), []byte(content), objSHA, committer("a@b.c"))
		assert.Equal(t, enum.GitSignatureUnverified, res)
	})
	t.Run("signature not parsed beforehand", func(t *testing.T) {
		res := (&Verify{}).Verify(ctx, aliceKey, []byte(content), objSHA, committer("alice@example.com"))
		assert.Equal(t, enum.GitSignatureInvalid, res)
	})
	t.Run("revoked key", func(t *testing.T) {
		revoked := svcNewEntity(t, "Rev", "rev@example.com")
		v := parsed(t, revoked, content)
		require.NoError(t, revoked.RevokeKey(packet.KeyCompromised, "leaked", svcPGPConfig()))
		res := v.Verify(ctx, []byte(svcArmorPublic(t, revoked)), []byte(content), objSHA, committer("rev@example.com"))
		assert.Equal(t, enum.GitSignatureRevoked, res)
	})
}

func TestSvcHasSigningKey(t *testing.T) {
	e := svcNewEntity(t, "Alice", "alice@example.com")
	require.NoError(t, e.AddSigningSubkey(svcPGPConfig()))
	ring := openpgp.EntityList{e}

	assert.True(t, hasSigningKey(ring, e.PrimaryKey.KeyId))
	assert.True(t, hasSigningKey(ring, e.Subkeys[len(e.Subkeys)-1].PublicKey.KeyId))
	assert.False(t, hasSigningKey(ring, 12345))
	assert.False(t, hasSigningKey(nil, e.PrimaryKey.KeyId))
}
