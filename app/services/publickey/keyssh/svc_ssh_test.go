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

package keyssh

import (
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/pem"
	"errors"
	"testing"

	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gossh "golang.org/x/crypto/ssh"
)

const svcObjectSHA = "0000000000000000000000000000000000000001"

type svcSSHKey struct {
	signer gossh.Signer
	pub    gossh.PublicKey
	line   string
}

func svcNewEd25519Key(t *testing.T, comment string) svcSSHKey {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	signer, err := gossh.NewSignerFromKey(priv)
	require.NoError(t, err)
	line := string(gossh.MarshalAuthorizedKey(signer.PublicKey()))
	if comment != "" {
		line = line[:len(line)-1] + " " + comment + "\n"
	}
	return svcSSHKey{signer: signer, pub: signer.PublicKey(), line: line}
}

func svcNewECDSAKey(t *testing.T) svcSSHKey {
	t.Helper()
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	signer, err := gossh.NewSignerFromKey(priv)
	require.NoError(t, err)
	return svcSSHKey{signer: signer, pub: signer.PublicKey(), line: string(gossh.MarshalAuthorizedKey(signer.PublicKey()))}
}

// svcSign produces an armored SSH signature in the sshsig format.
func svcSign(t *testing.T, key svcSSHKey, content []byte, namespace, hashAlg string, version uint32) []byte {
	t.Helper()

	var sum []byte
	switch hashAlg {
	case "sha256":
		s := sha256.Sum256(content)
		sum = s[:]
	default:
		s := sha512.Sum512(content)
		sum = s[:]
	}

	wrapper := gossh.Marshal(messageWrapper{Namespace: namespace, HashAlgorithm: hashAlg, Hash: sum})
	toSign := append([]byte(sshMagicPreamble), wrapper...)
	sig, err := key.signer.Sign(rand.Reader, toSign)
	require.NoError(t, err)

	blob := signatureBlob{
		Version:       version,
		PublicKey:     key.pub.Marshal(),
		Namespace:     namespace,
		HashAlgorithm: hashAlg,
		Signature:     gossh.Marshal(sig),
	}
	copy(blob.MagicPreamble[:], sshMagicPreamble)

	return pem.EncodeToMemory(&pem.Block{Type: SignatureType, Bytes: gossh.Marshal(blob)})
}

func TestSvcParse(t *testing.T) {
	ed := svcNewEd25519Key(t, "me@example.com")
	ec := svcNewECDSAKey(t)

	tests := []struct {
		name        string
		data        string
		wantType    string
		wantComment string
		wantErr     bool
	}{
		{"ed25519 with comment", ed.line, gossh.KeyAlgoED25519, "me@example.com", false},
		{"ecdsa", ec.line, gossh.KeyAlgoECDSA256, "", false},
		{"garbage", "this is not a key", "", "", true},
		{"empty", "", "", "", true},
		{"truncated key", "ssh-ed25519 AAAA", "", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := Parse([]byte(tt.data))
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantType, key.Type())
			assert.Equal(t, tt.wantComment, key.Comment())
		})
	}
}

func TestSvcKeyInfo(t *testing.T) {
	ed := svcNewEd25519Key(t, "")
	other := svcNewEd25519Key(t, "")

	key, err := Parse([]byte(ed.line))
	require.NoError(t, err)

	assert.True(t, key.Matches(ed.line))
	assert.False(t, key.Matches(other.line))
	assert.False(t, key.Matches("garbage"))

	fp := key.Fingerprint()
	assert.Equal(t, gossh.FingerprintSHA256(ed.pub), fp)
	assert.Equal(t, fp, FromSSH(ed.pub).Fingerprint())
	assert.NotEqual(t, fp, FromSSH(other.pub).Fingerprint())

	assert.Equal(t, enum.PublicKeySchemeSSH, key.Scheme())
	assert.Nil(t, key.ValidFrom())
	assert.Nil(t, key.ValidTo())
	assert.Nil(t, key.Identities())
	assert.Nil(t, key.RevocationReason())
	assert.JSONEq(t, "{}", string(key.Metadata()))
	assert.Nil(t, key.KeyIDs())
	assert.Nil(t, key.CompromisedIDs())
	assert.Empty(t, FromSSH(ed.pub).Comment())
}

func TestSvcHashFunc(t *testing.T) {
	assert.NotNil(t, hashFunc("sha256"))
	assert.NotNil(t, hashFunc("sha512"))
	assert.Nil(t, hashFunc("md5"))
	assert.Nil(t, hashFunc(""))
}

func TestSvcVerifyParse(t *testing.T) {
	ctx := context.Background()
	key := svcNewEd25519Key(t, "")
	content := []byte("tree abc\nauthor me\n\nmessage")
	objSHA := sha.Must(svcObjectSHA)

	t.Run("valid signature", func(t *testing.T) {
		v := &Verify{}
		sig := svcSign(t, key, content, sshNamespace, "sha512", 1)
		require.Empty(t, v.Parse(ctx, sig, objSHA))
		assert.Equal(t, gossh.FingerprintSHA256(key.pub), v.KeyFingerprint())
		assert.Equal(t, key.pub.Marshal(), v.SignaturePublicKey())
		assert.Equal(t, enum.PublicKeySchemeSSH, v.KeyScheme())
		assert.Empty(t, v.KeyID())
	})

	tests := []struct {
		name string
		sig  func() []byte
		want enum.GitSignatureResult
	}{
		{"not pem", func() []byte { return []byte("nope") }, enum.GitSignatureInvalid},
		{"wrong block type", func() []byte {
			return pem.EncodeToMemory(&pem.Block{Type: "PGP SIGNATURE", Bytes: []byte("x")})
		}, enum.GitSignatureInvalid},
		{"undecodable payload", func() []byte {
			return pem.EncodeToMemory(&pem.Block{Type: SignatureType, Bytes: []byte("x")})
		}, enum.GitSignatureInvalid},
		{"bad preamble", func() []byte {
			blob := signatureBlob{Version: 1, PublicKey: key.pub.Marshal(), Namespace: sshNamespace, HashAlgorithm: "sha512"}
			copy(blob.MagicPreamble[:], "BADSIG")
			return pem.EncodeToMemory(&pem.Block{Type: SignatureType, Bytes: gossh.Marshal(blob)})
		}, enum.GitSignatureInvalid},
		{"unsupported version", func() []byte {
			return svcSign(t, key, content, sshNamespace, "sha512", 2)
		}, enum.GitSignatureUnsupported},
		{"wrong namespace", func() []byte {
			return svcSign(t, key, content, "file", "sha512", 1)
		}, enum.GitSignatureInvalid},
		{"invalid embedded key", func() []byte {
			blob := signatureBlob{Version: 1, PublicKey: []byte("junk"), Namespace: sshNamespace, HashAlgorithm: "sha512"}
			copy(blob.MagicPreamble[:], sshMagicPreamble)
			return pem.EncodeToMemory(&pem.Block{Type: SignatureType, Bytes: gossh.Marshal(blob)})
		}, enum.GitSignatureInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &Verify{}
			assert.Equal(t, tt.want, v.Parse(ctx, tt.sig(), objSHA))
		})
	}
}

func TestSvcVerifyVerify(t *testing.T) {
	ctx := context.Background()
	key := svcNewEd25519Key(t, "")
	ecKey := svcNewECDSAKey(t)
	other := svcNewEd25519Key(t, "")
	content := []byte("signed content")
	objSHA := sha.Must(svcObjectSHA)
	signer := types.Signature{}

	prepared := func(t *testing.T, k svcSSHKey, hashAlg string) *Verify {
		t.Helper()
		v := &Verify{}
		require.Empty(t, v.Parse(ctx, svcSign(t, k, content, sshNamespace, hashAlg, 1), objSHA))
		return v
	}

	tests := []struct {
		name    string
		verify  func(t *testing.T) enum.GitSignatureResult
		want    enum.GitSignatureResult
		message string
	}{
		{"good sha512", func(t *testing.T) enum.GitSignatureResult {
			return prepared(t, key, "sha512").Verify(ctx, []byte(key.line), content, objSHA, signer)
		}, enum.GitSignatureGood, ""},
		{"good sha256", func(t *testing.T) enum.GitSignatureResult {
			return prepared(t, key, "sha256").Verify(ctx, []byte(key.line), content, objSHA, signer)
		}, enum.GitSignatureGood, ""},
		{"good ecdsa", func(t *testing.T) enum.GitSignatureResult {
			return prepared(t, ecKey, "sha512").Verify(ctx, []byte(ecKey.line), content, objSHA, signer)
		}, enum.GitSignatureGood, ""},
		{"tampered content", func(t *testing.T) enum.GitSignatureResult {
			return prepared(t, key, "sha512").Verify(ctx, []byte(key.line), []byte("other"), objSHA, signer)
		}, enum.GitSignatureBad, ""},
		{"different key", func(t *testing.T) enum.GitSignatureResult {
			return prepared(t, key, "sha512").Verify(ctx, []byte(other.line), content, objSHA, signer)
		}, enum.GitSignatureBad, ""},
		{"unparsable stored key", func(t *testing.T) enum.GitSignatureResult {
			return prepared(t, key, "sha512").Verify(ctx, []byte("garbage"), content, objSHA, signer)
		}, enum.GitSignatureUnverified, ""},
		{"unknown hash algorithm", func(t *testing.T) enum.GitSignatureResult {
			v := prepared(t, key, "sha512")
			v.hashAlgorithm = "md5"
			return v.Verify(ctx, []byte(key.line), content, objSHA, signer)
		}, enum.GitSignatureInvalid, ""},
		{"undecodable signature bytes", func(t *testing.T) enum.GitSignatureResult {
			v := prepared(t, key, "sha512")
			v.signatureBytes = []byte("x")
			return v.Verify(ctx, []byte(key.line), content, objSHA, signer)
		}, enum.GitSignatureInvalid, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.verify(t))
		})
	}
}

type svcFetcher struct {
	keys    []types.PublicKey
	err     error
	lastFP  string
	lastPID int64
}

func (f *svcFetcher) FetchByFingerprint(
	_ context.Context, fp string, principalID int64, _ []enum.PublicKeyUsage, _ []enum.PublicKeyScheme,
) ([]types.PublicKey, error) {
	f.lastFP = fp
	f.lastPID = principalID
	return f.keys, f.err
}

func (f *svcFetcher) FetchBySubKeyID(
	context.Context, string, int64, []enum.PublicKeyUsage, []enum.PublicKeyScheme,
) ([]types.PublicKey, error) {
	return nil, errors.New("unexpected call")
}

func TestSvcVerifyKey(t *testing.T) {
	ctx := context.Background()
	key := svcNewEd25519Key(t, "")
	v := &Verify{}
	require.Empty(t, v.Parse(ctx, svcSign(t, key, []byte("c"), sshNamespace, "sha512", 1), sha.Must(svcObjectSHA)))

	t.Run("found", func(t *testing.T) {
		f := &svcFetcher{keys: []types.PublicKey{{ID: 1}, {ID: 2}}}
		got, err := v.Key(ctx, f, 42)
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, int64(1), got.ID)
		assert.Equal(t, gossh.FingerprintSHA256(key.pub), f.lastFP)
		assert.Equal(t, int64(42), f.lastPID)
	})
	t.Run("not found", func(t *testing.T) {
		got, err := v.Key(ctx, &svcFetcher{}, 42)
		require.NoError(t, err)
		assert.Nil(t, got)
	})
	t.Run("fetch error", func(t *testing.T) {
		_, err := v.Key(ctx, &svcFetcher{err: errors.New("db down")}, 42)
		require.Error(t, err)
	})
}
