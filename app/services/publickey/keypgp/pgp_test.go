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
	"github.com/stretchr/testify/require"
)

const testEmail = "jane@example.com"

func newEntity(t *testing.T, email string) *openpgp.Entity {
	t.Helper()

	e, err := openpgp.NewEntity("Jane Doe", "work key", email, &packet.Config{
		Algorithm: packet.PubKeyAlgoRSA,
		RSABits:   2048,
	})
	require.NoError(t, err)
	return e
}

func armoredPublic(t *testing.T, e *openpgp.Entity) string {
	t.Helper()

	var buf bytes.Buffer
	w, err := armor.Encode(&buf, openpgp.PublicKeyType, nil)
	require.NoError(t, err)
	require.NoError(t, e.Serialize(w))
	require.NoError(t, w.Close())
	return buf.String()
}

func armoredPrivate(t *testing.T, e *openpgp.Entity) string {
	t.Helper()

	var buf bytes.Buffer
	w, err := armor.Encode(&buf, openpgp.PrivateKeyType, nil)
	require.NoError(t, err)
	require.NoError(t, e.SerializePrivate(w, nil))
	require.NoError(t, w.Close())
	return buf.String()
}

func detachSign(t *testing.T, e *openpgp.Entity, content string) []byte {
	t.Helper()

	var buf bytes.Buffer
	require.NoError(t, openpgp.ArmoredDetachSign(&buf, e, strings.NewReader(content), nil))
	return buf.Bytes()
}

func TestParse(t *testing.T) {
	entity := newEntity(t, testEmail)
	require.NoError(t, entity.AddSigningSubkey(nil))
	public := armoredPublic(t, entity)

	t.Run("valid key without principal", func(t *testing.T) {
		info, err := Parse(strings.NewReader(public), nil)
		require.NoError(t, err)

		require.Equal(t, enum.PublicKeySchemePGP, info.Scheme())
		require.Equal(t, "RSA", info.Type())
		require.Equal(t, "work key", info.Comment())
		require.NotEmpty(t, info.Fingerprint())
		require.Equal(t, strings.ToUpper(info.Fingerprint()), info.Fingerprint())
		require.NotNil(t, info.ValidFrom())
		require.Positive(t, *info.ValidFrom())
		require.Nil(t, info.ValidTo())
		require.Nil(t, info.RevocationReason())
		require.Equal(t, []types.Identity{{Name: "Jane Doe", Email: testEmail}}, info.Identities())

		var meta EntityMetadata
		require.NoError(t, json.Unmarshal(info.Metadata(), &meta))
		require.Equal(t, info.Fingerprint(), meta.PrimaryKey.Fingerprint)
		require.Equal(t, "RSA", meta.PrimaryKey.Algorithm)
		require.Equal(t, uint16(2048), meta.PrimaryKey.BitLength)
		require.Equal(t, &types.Identity{Name: "Jane Doe", Email: testEmail}, meta.PrimaryIdentity)
		// RSA subkeys are always reported as capable of signing
		require.Len(t, meta.SubKeys, len(entity.Subkeys))

		require.Len(t, info.KeyIDs(), 1+len(entity.Subkeys))
		require.Empty(t, info.CompromisedIDs())
	})

	t.Run("matching principal is case insensitive", func(t *testing.T) {
		_, err := Parse(strings.NewReader(public), &types.Principal{Email: strings.ToUpper(testEmail)})
		require.NoError(t, err)
	})

	t.Run("principal mismatch", func(t *testing.T) {
		_, err := Parse(strings.NewReader(public), &types.Principal{Email: "other@example.com"})
		require.Error(t, err)
		require.Contains(t, err.Error(), "email")
	})

	t.Run("private key is refused", func(t *testing.T) {
		_, err := Parse(strings.NewReader(armoredPrivate(t, entity)), nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "private key")
	})

	t.Run("garbage input", func(t *testing.T) {
		_, err := Parse(strings.NewReader("not a key"), nil)
		require.Error(t, err)
	})

	t.Run("multiple primary keys", func(t *testing.T) {
		other := newEntity(t, "other@example.com")
		var buf bytes.Buffer
		w, err := armor.Encode(&buf, openpgp.PublicKeyType, nil)
		require.NoError(t, err)
		require.NoError(t, entity.Serialize(w))
		require.NoError(t, other.Serialize(w))
		require.NoError(t, w.Close())

		_, err = Parse(&buf, nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "multiple")
	})
}

func TestKeyInfo_Matches(t *testing.T) {
	entity := newEntity(t, testEmail)
	public := armoredPublic(t, entity)
	info, err := Parse(strings.NewReader(public), nil)
	require.NoError(t, err)

	require.True(t, info.Matches(public))
	require.False(t, info.Matches(armoredPublic(t, newEntity(t, testEmail))))
	require.False(t, info.Matches("garbage"))
}

func TestKeyInfo_Revoked(t *testing.T) {
	for name, tc := range map[string]struct {
		reason packet.ReasonForRevocation
		want   enum.RevocationReason
	}{
		"compromised": {packet.KeyCompromised, enum.RevocationReasonCompromised},
		"retired":     {packet.KeyRetired, enum.RevocationReasonRetired},
		"superseded":  {packet.KeySuperseded, enum.RevocationReasonSuperseded},
		"unspecified": {packet.NoReason, enum.RevocationReasonUnknown},
	} {
		t.Run(name, func(t *testing.T) {
			entity := newEntity(t, testEmail)
			require.NoError(t, entity.AddSigningSubkey(nil))
			require.NoError(t, entity.RevokeKey(tc.reason, "revoked in test", nil))

			info, err := Parse(strings.NewReader(armoredPublic(t, entity)), nil)
			require.NoError(t, err)

			require.NotNil(t, info.RevocationReason())
			require.Equal(t, tc.want, *info.RevocationReason())

			if tc.want == enum.RevocationReasonCompromised {
				// a compromised primary key compromises all signing subkeys
				require.Len(t, info.CompromisedIDs(), 1+len(entity.Subkeys))
			} else {
				require.Empty(t, info.CompromisedIDs())
			}
		})
	}

	t.Run("compromised subkey only", func(t *testing.T) {
		entity := newEntity(t, testEmail)
		require.NoError(t, entity.AddSigningSubkey(nil))
		var signing *openpgp.Subkey
		for i := range entity.Subkeys {
			if entity.Subkeys[i].PublicKey.CanSign() {
				signing = &entity.Subkeys[i]
			}
		}
		require.NotNil(t, signing)
		require.NoError(t, entity.RevokeSubkey(signing, packet.KeyCompromised, "leaked", nil))

		info, err := Parse(strings.NewReader(armoredPublic(t, entity)), nil)
		require.NoError(t, err)
		require.Equal(t, []string{signing.PublicKey.KeyIdString()}, info.CompromisedIDs())
		require.Nil(t, info.RevocationReason())
	})
}

func TestPGPAlgo(t *testing.T) {
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
		packet.PublicKeyAlgorithm(250):  "",
	}
	for algo, want := range tests {
		require.Equal(t, want, pgpAlgo(algo))
	}
}

func TestGetRevocationReason(t *testing.T) {
	require.Nil(t, getRevocationReason(nil))

	reason := func(r packet.ReasonForRevocation) *packet.Signature {
		return &packet.Signature{RevocationReason: &r}
	}

	got := getRevocationReason([]*packet.Signature{nil, {}})
	require.NotNil(t, got)
	require.Equal(t, enum.RevocationReasonUnknown, *got)

	got = getRevocationReason([]*packet.Signature{reason(packet.KeyRetired), reason(packet.KeySuperseded)})
	require.Equal(t, enum.RevocationReasonSuperseded, *got)

	got = getRevocationReason([]*packet.Signature{reason(packet.KeyRetired), reason(packet.KeyCompromised)})
	require.Equal(t, enum.RevocationReasonCompromised, *got)
}

func TestVerify_Parse(t *testing.T) {
	ctx := context.Background()
	entity := newEntity(t, testEmail)
	objectSHA := sha.Must("1111111111111111111111111111111111111111")

	t.Run("valid signature", func(t *testing.T) {
		v := &Verify{}
		require.Empty(t, v.Parse(ctx, detachSign(t, entity, "content"), objectSHA))
		require.Equal(t, enum.PublicKeySchemePGP, v.KeyScheme())
		require.Len(t, v.KeyID(), 16)
		require.Equal(t, strings.ToUpper(v.KeyID()), v.KeyID())
		require.Equal(t, entity.PrimaryKey.KeyIdString(), v.KeyID())
		require.NotEmpty(t, v.KeyFingerprint())
	})

	t.Run("invalid inputs", func(t *testing.T) {
		for name, data := range map[string][]byte{
			"garbage":         []byte("garbage"),
			"wrong armor":     []byte(armoredPublic(t, entity)),
			"empty signature": []byte("-----BEGIN PGP SIGNATURE-----\n\n-----END PGP SIGNATURE-----\n"),
		} {
			t.Run(name, func(t *testing.T) {
				v := &Verify{}
				require.Equal(t, enum.GitSignatureInvalid, v.Parse(ctx, data, objectSHA))
			})
		}
	})

	t.Run("not a signature packet", func(t *testing.T) {
		var buf bytes.Buffer
		w, err := armor.Encode(&buf, openpgp.SignatureType, nil)
		require.NoError(t, err)
		require.NoError(t, entity.PrimaryKey.Serialize(w))
		require.NoError(t, w.Close())

		v := &Verify{}
		require.Equal(t, enum.GitSignatureInvalid, v.Parse(ctx, buf.Bytes(), objectSHA))
	})
}

func TestVerify_Verify(t *testing.T) {
	ctx := context.Background()
	entity := newEntity(t, testEmail)
	public := []byte(armoredPublic(t, entity))
	objectSHA := sha.Must("1111111111111111111111111111111111111111")
	committer := types.Signature{Identity: types.Identity{Name: "Jane", Email: testEmail}, When: time.Now()}
	const content = "tree abc\nauthor Jane\n\nmessage"

	parsed := func(t *testing.T, e *openpgp.Entity, signed string) *Verify {
		t.Helper()
		v := &Verify{}
		require.Empty(t, v.Parse(ctx, detachSign(t, e, signed), objectSHA))
		return v
	}

	t.Run("good", func(t *testing.T) {
		v := parsed(t, entity, content)
		require.Equal(t, enum.GitSignatureGood, v.Verify(ctx, public, []byte(content), objectSHA, committer))
	})

	t.Run("committer email is matched case insensitively", func(t *testing.T) {
		v := parsed(t, entity, content)
		c := committer
		c.Identity.Email = strings.ToUpper(testEmail)
		require.Equal(t, enum.GitSignatureGood, v.Verify(ctx, public, []byte(content), objectSHA, c))
	})

	t.Run("committer without matching identity", func(t *testing.T) {
		v := parsed(t, entity, content)
		c := committer
		c.Identity.Email = "other@example.com"
		require.Equal(t, enum.GitSignatureBad, v.Verify(ctx, public, []byte(content), objectSHA, c))
	})

	t.Run("modified content", func(t *testing.T) {
		v := parsed(t, entity, content)
		require.Equal(t, enum.GitSignatureInvalid,
			v.Verify(ctx, public, []byte(content+" tampered"), objectSHA, committer))
	})

	t.Run("signed by a different key", func(t *testing.T) {
		other := newEntity(t, testEmail)
		v := parsed(t, other, content)
		require.Equal(t, enum.GitSignatureBad, v.Verify(ctx, public, []byte(content), objectSHA, committer))
	})

	t.Run("unreadable public key", func(t *testing.T) {
		v := parsed(t, entity, content)
		require.Equal(t, enum.GitSignatureUnverified,
			v.Verify(ctx, []byte("not a key"), []byte(content), objectSHA, committer))
	})

	t.Run("signature was never parsed", func(t *testing.T) {
		v := &Verify{}
		require.Equal(t, enum.GitSignatureInvalid, v.Verify(ctx, public, []byte(content), objectSHA, committer))
	})

	t.Run("revoked key", func(t *testing.T) {
		revoked := newEntity(t, testEmail)
		v := parsed(t, revoked, content)
		require.NoError(t, revoked.RevokeKey(packet.KeyCompromised, "leaked", nil))
		require.Equal(t, enum.GitSignatureRevoked,
			v.Verify(ctx, []byte(armoredPublic(t, revoked)), []byte(content), objectSHA, committer))
	})
}

func TestHasSigningKey(t *testing.T) {
	entity := newEntity(t, testEmail)
	require.NoError(t, entity.AddSigningSubkey(nil))
	ring := openpgp.EntityList{entity}

	require.True(t, hasSigningKey(ring, entity.PrimaryKey.KeyId))
	for i := range entity.Subkeys {
		flagged := entity.Subkeys[i].Sig != nil && entity.Subkeys[i].Sig.FlagSign
		require.Equal(t, flagged, hasSigningKey(ring, entity.Subkeys[i].PublicKey.KeyId))
	}
	require.False(t, hasSigningKey(ring, 12345))
}

type fakeKeyFetcher struct {
	keys []types.PublicKey
	err  error

	gotSubKeyID string
	gotUsages   []enum.PublicKeyUsage
	gotSchemes  []enum.PublicKeyScheme
}

func (f *fakeKeyFetcher) FetchByFingerprint(
	context.Context, string, int64, []enum.PublicKeyUsage, []enum.PublicKeyScheme,
) ([]types.PublicKey, error) {
	return nil, errors.New("unexpected call")
}

func (f *fakeKeyFetcher) FetchBySubKeyID(
	_ context.Context, subKeyID string, _ int64, usages []enum.PublicKeyUsage, schemes []enum.PublicKeyScheme,
) ([]types.PublicKey, error) {
	f.gotSubKeyID, f.gotUsages, f.gotSchemes = subKeyID, usages, schemes
	return f.keys, f.err
}

func TestVerify_Key(t *testing.T) {
	ctx := context.Background()
	v := &Verify{keyID: "0000000000000001"}

	t.Run("found", func(t *testing.T) {
		f := &fakeKeyFetcher{keys: []types.PublicKey{{Identifier: "first"}, {Identifier: "second"}}}
		key, err := v.Key(ctx, f, 7)
		require.NoError(t, err)
		require.Equal(t, "first", key.Identifier)
		require.Equal(t, "0000000000000001", f.gotSubKeyID)
		require.Equal(t, []enum.PublicKeyUsage{enum.PublicKeyUsageSign}, f.gotUsages)
		require.Equal(t, []enum.PublicKeyScheme{enum.PublicKeySchemePGP}, f.gotSchemes)
	})

	t.Run("not found", func(t *testing.T) {
		key, err := v.Key(ctx, &fakeKeyFetcher{}, 7)
		require.NoError(t, err)
		require.Nil(t, key)
	})

	t.Run("fetch error", func(t *testing.T) {
		_, err := v.Key(ctx, &fakeKeyFetcher{err: errors.New("db down")}, 7)
		require.Error(t, err)
	})
}
