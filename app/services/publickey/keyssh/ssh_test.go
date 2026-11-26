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
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/pem"
	"errors"
	"testing"

	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

func newSigner(t *testing.T, kind string) ssh.Signer {
	t.Helper()

	switch kind {
	case "rsa":
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)
		signer, err := ssh.NewSignerFromKey(key)
		require.NoError(t, err)
		return signer
	case "ecdsa":
		key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)
		signer, err := ssh.NewSignerFromKey(key)
		require.NoError(t, err)
		return signer
	default:
		_, key, err := ed25519.GenerateKey(rand.Reader)
		require.NoError(t, err)
		signer, err := ssh.NewSignerFromKey(key)
		require.NoError(t, err)
		return signer
	}
}

func authorizedKey(signer ssh.Signer, comment string) []byte {
	line := ssh.MarshalAuthorizedKey(signer.PublicKey())
	if comment == "" {
		return line
	}
	return append(line[:len(line)-1], []byte(" "+comment+"\n")...)
}

// sign builds an armored SSH signature for content the same way `ssh-keygen -Y sign -n git` does.
func sign(t *testing.T, signer ssh.Signer, content []byte, mutate func(*signatureBlob)) []byte {
	t.Helper()

	const hashAlgorithm = "sha512"
	sum := sha512.Sum512(content)
	message := append([]byte(sshMagicPreamble), ssh.Marshal(messageWrapper{
		Namespace:     sshNamespace,
		HashAlgorithm: hashAlgorithm,
		Hash:          sum[:],
	})...)

	sig, err := signer.Sign(rand.Reader, message)
	require.NoError(t, err)

	blob := signatureBlob{
		Version:       1,
		PublicKey:     signer.PublicKey().Marshal(),
		Namespace:     sshNamespace,
		HashAlgorithm: hashAlgorithm,
		Signature:     ssh.Marshal(sig),
	}
	copy(blob.MagicPreamble[:], sshMagicPreamble)
	if mutate != nil {
		mutate(&blob)
	}

	return pem.EncodeToMemory(&pem.Block{Type: SignatureType, Bytes: ssh.Marshal(blob)})
}

func TestParse(t *testing.T) {
	for _, kind := range []string{"ed25519", "rsa", "ecdsa"} {
		t.Run(kind, func(t *testing.T) {
			signer := newSigner(t, kind)
			info, err := Parse(authorizedKey(signer, "me@laptop"))
			require.NoError(t, err)

			require.Equal(t, "me@laptop", info.Comment())
			require.Equal(t, signer.PublicKey().Type(), info.Type())
			require.Equal(t, enum.PublicKeySchemeSSH, info.Scheme())
			require.Equal(t, ssh.FingerprintSHA256(signer.PublicKey()), info.Fingerprint())
		})
	}

	t.Run("invalid data", func(t *testing.T) {
		_, err := Parse([]byte("ssh-ed25519 not-base64"))
		require.Error(t, err)
		_, err = Parse([]byte(""))
		require.Error(t, err)
	})

	t.Run("certificates are not allowed", func(t *testing.T) {
		signer := newSigner(t, "ed25519")
		ca := newSigner(t, "ed25519")
		cert := &ssh.Certificate{Key: signer.PublicKey(), CertType: ssh.UserCert, ValidBefore: ssh.CertTimeInfinity}
		require.NoError(t, cert.SignCert(rand.Reader, ca))

		_, err := Parse(ssh.MarshalAuthorizedKey(cert))
		require.Error(t, err)
		require.Contains(t, err.Error(), "allowed key types")
	})
}

func TestKeyInfo(t *testing.T) {
	signer := newSigner(t, "ed25519")
	info, err := Parse(authorizedKey(signer, ""))
	require.NoError(t, err)

	require.True(t, info.Matches(string(authorizedKey(signer, "other comment"))))
	require.False(t, info.Matches(string(authorizedKey(newSigner(t, "ed25519"), ""))))
	require.False(t, info.Matches("garbage"))

	require.Nil(t, info.ValidFrom())
	require.Nil(t, info.ValidTo())
	require.Nil(t, info.Identities())
	require.Nil(t, info.RevocationReason())
	require.JSONEq(t, "{}", string(info.Metadata()))
	require.Nil(t, info.KeyIDs())
	require.Nil(t, info.CompromisedIDs())

	fromSSH := FromSSH(signer.PublicKey())
	require.Empty(t, fromSSH.Comment())
	require.Equal(t, info.Fingerprint(), fromSSH.Fingerprint())
}

func TestHashFunc(t *testing.T) {
	require.NotNil(t, hashFunc("sha256"))
	require.Equal(t, sha256.Size, hashFunc("sha256").Size())
	require.NotNil(t, hashFunc("sha512"))
	require.Equal(t, sha512.Size, hashFunc("sha512").Size())
	require.Nil(t, hashFunc("md5"))
	require.Nil(t, hashFunc(""))
}

func TestVerify_Parse(t *testing.T) {
	ctx := context.Background()
	signer := newSigner(t, "ed25519")
	objectSHA := sha.Must("1111111111111111111111111111111111111111")
	content := []byte("tree abc\n\nmessage")

	t.Run("valid", func(t *testing.T) {
		v := &Verify{}
		require.Empty(t, v.Parse(ctx, sign(t, signer, content, nil), objectSHA))
		require.Equal(t, ssh.FingerprintSHA256(signer.PublicKey()), v.KeyFingerprint())
		require.Equal(t, signer.PublicKey().Marshal(), v.SignaturePublicKey())
		require.Equal(t, enum.PublicKeySchemeSSH, v.KeyScheme())
		require.Empty(t, v.KeyID())
	})

	t.Run("not pem", func(t *testing.T) {
		require.Equal(t, enum.GitSignatureInvalid, (&Verify{}).Parse(ctx, []byte("garbage"), objectSHA))
	})

	t.Run("wrong block type", func(t *testing.T) {
		data := pem.EncodeToMemory(&pem.Block{Type: "PGP SIGNATURE", Bytes: []byte("x")})
		require.Equal(t, enum.GitSignatureInvalid, (&Verify{}).Parse(ctx, data, objectSHA))
	})

	t.Run("not a signature blob", func(t *testing.T) {
		data := pem.EncodeToMemory(&pem.Block{Type: SignatureType, Bytes: []byte("short")})
		require.Equal(t, enum.GitSignatureInvalid, (&Verify{}).Parse(ctx, data, objectSHA))
	})

	for name, tc := range map[string]struct {
		mutate func(*signatureBlob)
		want   enum.GitSignatureResult
	}{
		"bad preamble": {
			func(b *signatureBlob) { b.MagicPreamble = [6]byte{'N', 'O', 'T', 'S', 'S', 'H'} },
			enum.GitSignatureInvalid,
		},
		"newer version":   {func(b *signatureBlob) { b.Version = 2 }, enum.GitSignatureUnsupported},
		"wrong namespace": {func(b *signatureBlob) { b.Namespace = "file" }, enum.GitSignatureInvalid},
		"bad public key":  {func(b *signatureBlob) { b.PublicKey = []byte("nope") }, enum.GitSignatureInvalid},
	} {
		t.Run(name, func(t *testing.T) {
			v := &Verify{}
			require.Equal(t, tc.want, v.Parse(ctx, sign(t, signer, content, tc.mutate), objectSHA))
		})
	}
}

func TestVerify_Verify(t *testing.T) {
	ctx := context.Background()
	objectSHA := sha.Must("1111111111111111111111111111111111111111")
	content := []byte("tree abc\n\nmessage")
	committer := types.Signature{}

	for _, kind := range []string{"ed25519", "rsa", "ecdsa"} {
		t.Run(kind, func(t *testing.T) {
			signer := newSigner(t, kind)
			public := authorizedKey(signer, "")

			v := &Verify{}
			require.Empty(t, v.Parse(ctx, sign(t, signer, content, nil), objectSHA))

			require.Equal(t, enum.GitSignatureGood, v.Verify(ctx, public, content, objectSHA, committer))
			require.Equal(t, enum.GitSignatureBad, v.Verify(ctx, public, []byte("tampered"), objectSHA, committer))

			otherPublic := authorizedKey(newSigner(t, kind), "")
			require.Equal(t, enum.GitSignatureBad, v.Verify(ctx, otherPublic, content, objectSHA, committer))
		})
	}

	signer := newSigner(t, "ed25519")
	public := authorizedKey(signer, "")

	t.Run("unreadable public key", func(t *testing.T) {
		v := &Verify{}
		require.Empty(t, v.Parse(ctx, sign(t, signer, content, nil), objectSHA))
		require.Equal(t, enum.GitSignatureUnverified, v.Verify(ctx, []byte("junk"), content, objectSHA, committer))
	})

	t.Run("unknown hash algorithm", func(t *testing.T) {
		v := &Verify{}
		md5Sig := sign(t, signer, content, func(b *signatureBlob) { b.HashAlgorithm = "md5" })
		require.Empty(t, v.Parse(ctx, md5Sig, objectSHA))
		require.Equal(t, enum.GitSignatureInvalid, v.Verify(ctx, public, content, objectSHA, committer))
	})

	t.Run("corrupt signature bytes", func(t *testing.T) {
		v := &Verify{}
		corrupt := sign(t, signer, content, func(b *signatureBlob) { b.Signature = []byte("x") })
		require.Empty(t, v.Parse(ctx, corrupt, objectSHA))
		require.Equal(t, enum.GitSignatureInvalid, v.Verify(ctx, public, content, objectSHA, committer))
	})

	t.Run("sha256 hashed signature", func(t *testing.T) {
		sum := sha256.Sum256(content)
		message := append([]byte(sshMagicPreamble), ssh.Marshal(messageWrapper{
			Namespace: sshNamespace, HashAlgorithm: "sha256", Hash: sum[:],
		})...)
		sig, err := signer.Sign(rand.Reader, message)
		require.NoError(t, err)

		v := &Verify{hashAlgorithm: "sha256", signatureBytes: ssh.Marshal(sig)}
		require.Equal(t, enum.GitSignatureGood, v.Verify(ctx, public, content, objectSHA, committer))
	})
}

type fakeKeyFetcher struct {
	keys []types.PublicKey
	err  error

	gotFingerprint string
	gotUsages      []enum.PublicKeyUsage
	gotSchemes     []enum.PublicKeyScheme
}

func (f *fakeKeyFetcher) FetchByFingerprint(
	_ context.Context, fingerprint string, _ int64, usages []enum.PublicKeyUsage, schemes []enum.PublicKeyScheme,
) ([]types.PublicKey, error) {
	f.gotFingerprint, f.gotUsages, f.gotSchemes = fingerprint, usages, schemes
	return f.keys, f.err
}

func (f *fakeKeyFetcher) FetchBySubKeyID(
	context.Context, string, int64, []enum.PublicKeyUsage, []enum.PublicKeyScheme,
) ([]types.PublicKey, error) {
	return nil, errors.New("unexpected call")
}

func TestVerify_Key(t *testing.T) {
	ctx := context.Background()
	v := &Verify{keyFingerprint: "SHA256:abc"}

	t.Run("found", func(t *testing.T) {
		f := &fakeKeyFetcher{keys: []types.PublicKey{{Identifier: "first"}}}
		key, err := v.Key(ctx, f, 7)
		require.NoError(t, err)
		require.Equal(t, "first", key.Identifier)
		require.Equal(t, "SHA256:abc", f.gotFingerprint)
		require.Equal(t, []enum.PublicKeyUsage{enum.PublicKeyUsageSign, enum.PublicKeyUsageAuthSign}, f.gotUsages)
		require.Equal(t, []enum.PublicKeyScheme{enum.PublicKeySchemeSSH}, f.gotSchemes)
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
