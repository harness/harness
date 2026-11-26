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
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha512"
	"encoding/pem"
	"errors"
	"testing"
	"time"

	"github.com/harness/gitness/app/services/publickey/keypgp"
	"github.com/harness/gitness/app/services/publickey/keyssh"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/git/sha"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

type fakePrincipalStore struct {
	store.PrincipalStore

	byEmail map[string]int64
	err     error
	calls   int
}

func (f *fakePrincipalStore) FindByEmail(_ context.Context, email string) (*types.Principal, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	id, ok := f.byEmail[email]
	if !ok {
		return nil, gitness_store.ErrResourceNotFound
	}
	return &types.Principal{ID: id, Email: email}, nil
}

type fakeSigStore struct {
	store.GitSignatureResultStore

	existing  map[sha.SHA]types.GitSignatureResult
	mapErr    error
	createErr error
	created   []*types.GitSignatureResult
}

func (f *fakeSigStore) Map(context.Context, int64, []sha.SHA) (map[sha.SHA]types.GitSignatureResult, error) {
	return f.existing, f.mapErr
}

func (f *fakeSigStore) TryCreateAll(_ context.Context, results []*types.GitSignatureResult) error {
	f.created = append(f.created, results...)
	return f.createErr
}

type fakeFetcher struct {
	keys  []types.PublicKey
	err   error
	calls int
}

func (f *fakeFetcher) FetchByFingerprint(
	context.Context, string, int64, []enum.PublicKeyUsage, []enum.PublicKeyScheme,
) ([]types.PublicKey, error) {
	f.calls++
	return f.keys, f.err
}

func (f *fakeFetcher) FetchBySubKeyID(
	context.Context, string, int64, []enum.PublicKeyUsage, []enum.PublicKeyScheme,
) ([]types.PublicKey, error) {
	f.calls++
	return f.keys, f.err
}

// sshMaterial is a signing key together with a signature over the given content.
type sshMaterial struct {
	publicKey types.PublicKey
	data      *types.SignedData
}

type sshSigBlob struct {
	MagicPreamble [6]byte
	Version       uint32
	PublicKey     []byte
	Namespace     string
	Reserved      string
	HashAlgorithm string
	Signature     []byte
}

type sshMessageWrapper struct {
	Namespace     string
	Reserved      string
	HashAlgorithm string
	Hash          []byte
}

func newSSHMaterial(t *testing.T, content []byte, version uint32) sshMaterial {
	t.Helper()

	_, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	signer, err := ssh.NewSignerFromKey(priv)
	require.NoError(t, err)

	sum := sha512.Sum512(content)
	message := append([]byte("SSHSIG"), ssh.Marshal(sshMessageWrapper{
		Namespace: "git", HashAlgorithm: "sha512", Hash: sum[:],
	})...)
	sig, err := signer.Sign(rand.Reader, message)
	require.NoError(t, err)

	blob := sshSigBlob{
		Version: version, PublicKey: signer.PublicKey().Marshal(), Namespace: "git",
		HashAlgorithm: "sha512", Signature: ssh.Marshal(sig),
	}
	copy(blob.MagicPreamble[:], "SSHSIG")

	return sshMaterial{
		publicKey: types.PublicKey{
			Identifier:  "key",
			Fingerprint: ssh.FingerprintSHA256(signer.PublicKey()),
			Content:     string(ssh.MarshalAuthorizedKey(signer.PublicKey())),
		},
		data: &types.SignedData{
			Type:          keyssh.SignatureType,
			Signature:     pem.EncodeToMemory(&pem.Block{Type: keyssh.SignatureType, Bytes: ssh.Marshal(blob)}),
			SignedContent: content,
		},
	}
}

const (
	signerEmail = "jane@example.com"
	repoID      = int64(5)
)

var commitTime = time.Unix(1_700_000_000, 0)

func signerSig() *types.Signature {
	return &types.Signature{Identity: types.Identity{Name: "Jane", Email: signerEmail}, When: commitTime}
}

type verifyFixture struct {
	principals *fakePrincipalStore
	sigs       *fakeSigStore
	fetcher    *fakeFetcher
	svc        SignatureVerifyService
}

func newVerifyFixture(keys ...types.PublicKey) *verifyFixture {
	f := &verifyFixture{
		principals: &fakePrincipalStore{byEmail: map[string]int64{signerEmail: 7}},
		sigs:       &fakeSigStore{},
		fetcher:    &fakeFetcher{keys: keys},
	}
	f.svc = NewSignatureVerifyService(f.principals, f.fetcher, f.sigs)
	return f
}

func commitWith(shaHex string, data *types.SignedData, signer *types.Signature) *types.Commit {
	c := &types.Commit{SHA: sha.Must(shaHex), SignedData: data}
	if signer != nil {
		c.Committer = *signer
	}
	return c
}

const (
	sha1Hex = "1111111111111111111111111111111111111111"
	sha2Hex = "2222222222222222222222222222222222222222"
)

func TestVerifyCommits_GoodSSHSignature(t *testing.T) {
	content := []byte("tree abc\n\nmessage")
	m := newSSHMaterial(t, content, 1)
	f := newVerifyFixture(m.publicKey)

	commit := commitWith(sha1Hex, m.data, signerSig())
	require.NoError(t, f.svc.VerifyCommits(t.Context(), repoID, []*types.Commit{commit}))

	require.NotNil(t, commit.Signature)
	require.Equal(t, enum.GitSignatureGood, commit.Signature.Result)
	require.Equal(t, int64(7), commit.Signature.PrincipalID)
	require.Equal(t, enum.PublicKeySchemeSSH, commit.Signature.KeyScheme)
	require.Equal(t, m.publicKey.Fingerprint, commit.Signature.KeyFingerprint)
	require.Equal(t, commitTime.UnixMilli(), commit.Signature.ObjectTime)
	require.Equal(t, repoID, commit.Signature.RepoID)

	// the result is persisted
	require.Len(t, f.sigs.created, 1)
	require.Equal(t, enum.GitSignatureGood, f.sigs.created[0].Result)
}

func TestVerifyCommits_BadSignature(t *testing.T) {
	m := newSSHMaterial(t, []byte("original"), 1)
	m.data.SignedContent = []byte("tampered")
	f := newVerifyFixture(m.publicKey)

	commit := commitWith(sha1Hex, m.data, signerSig())
	require.NoError(t, f.svc.VerifyCommits(t.Context(), repoID, []*types.Commit{commit}))
	require.Equal(t, enum.GitSignatureBad, commit.Signature.Result)
	require.Len(t, f.sigs.created, 1, "bad signatures are stored")
}

func TestVerifyCommits_KeyProperties(t *testing.T) {
	content := []byte("content")
	before := commitTime.UnixMilli() - 1000
	after := commitTime.UnixMilli() + 1000
	reason := enum.RevocationReasonCompromised

	tests := []struct {
		name   string
		mutate func(*types.PublicKey)
		want   enum.GitSignatureResult
	}{
		{"revoked", func(k *types.PublicKey) { k.RevocationReason = &reason }, enum.GitSignatureRevoked},
		{"not yet valid", func(k *types.PublicKey) { k.ValidFrom = &after }, enum.GitSignatureKeyExpired},
		{"expired", func(k *types.PublicKey) { k.ValidTo = &before }, enum.GitSignatureKeyExpired},
		{"within validity", func(k *types.PublicKey) { k.ValidFrom = &before; k.ValidTo = &after }, enum.GitSignatureGood},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newSSHMaterial(t, content, 1)
			tt.mutate(&m.publicKey)
			f := newVerifyFixture(m.publicKey)

			commit := commitWith(sha1Hex, m.data, signerSig())
			require.NoError(t, f.svc.VerifyCommits(t.Context(), repoID, []*types.Commit{commit}))
			require.Equal(t, tt.want, commit.Signature.Result)
		})
	}
}

func TestVerifyCommits_UnverifiedAndUnsupported(t *testing.T) {
	m := newSSHMaterial(t, []byte("content"), 1)

	t.Run("unsigned objects are left alone", func(t *testing.T) {
		f := newVerifyFixture(m.publicKey)
		commit := commitWith(sha1Hex, nil, signerSig())
		require.NoError(t, f.svc.VerifyCommits(t.Context(), repoID, []*types.Commit{commit}))
		require.Nil(t, commit.Signature)
		require.Empty(t, f.sigs.created)
	})

	t.Run("unknown signature type", func(t *testing.T) {
		f := newVerifyFixture(m.publicKey)
		commit := commitWith(sha1Hex, &types.SignedData{Type: "X509 SIGNATURE"}, signerSig())
		require.NoError(t, f.svc.VerifyCommits(t.Context(), repoID, []*types.Commit{commit}))
		require.Equal(t, enum.GitSignatureUnsupported, commit.Signature.Result)
		require.Empty(t, f.sigs.created, "unsupported results are not stored")
	})

	t.Run("tag without a tagger", func(t *testing.T) {
		f := newVerifyFixture(m.publicKey)
		tag := &types.CommitTag{SHA: sha.Must(sha1Hex), SignedData: m.data}
		require.NoError(t, f.svc.VerifyCommitTags(t.Context(), repoID, []*types.CommitTag{tag}))
		require.Equal(t, enum.GitSignatureUnverified, tag.Signature.Result)
	})

	t.Run("unknown signer email", func(t *testing.T) {
		f := newVerifyFixture(m.publicKey)
		commit := commitWith(sha1Hex, m.data, &types.Signature{Identity: types.Identity{Email: "nobody@example.com"}})
		require.NoError(t, f.svc.VerifyCommits(t.Context(), repoID, []*types.Commit{commit}))
		require.Equal(t, enum.GitSignatureUnverified, commit.Signature.Result)
		require.Empty(t, f.sigs.created)
	})

	t.Run("key is not registered", func(t *testing.T) {
		f := newVerifyFixture()
		commit := commitWith(sha1Hex, m.data, signerSig())
		require.NoError(t, f.svc.VerifyCommits(t.Context(), repoID, []*types.Commit{commit}))
		require.Equal(t, enum.GitSignatureUnverified, commit.Signature.Result)
	})

	t.Run("garbage signature", func(t *testing.T) {
		f := newVerifyFixture(m.publicKey)
		data := &types.SignedData{Type: keyssh.SignatureType, Signature: []byte("garbage")}
		commit := commitWith(sha1Hex, data, signerSig())
		require.NoError(t, f.svc.VerifyCommits(t.Context(), repoID, []*types.Commit{commit}))
		require.Equal(t, enum.GitSignatureInvalid, commit.Signature.Result)
		require.Empty(t, f.sigs.created, "invalid results are not stored")
	})

	t.Run("unsupported signature version", func(t *testing.T) {
		f := newVerifyFixture(m.publicKey)
		future := newSSHMaterial(t, []byte("content"), 2)
		commit := commitWith(sha1Hex, future.data, signerSig())
		require.NoError(t, f.svc.VerifyCommits(t.Context(), repoID, []*types.Commit{commit}))
		require.Equal(t, enum.GitSignatureUnsupported, commit.Signature.Result)
	})

	t.Run("pgp signature that is not armored", func(t *testing.T) {
		f := newVerifyFixture(m.publicKey)
		data := &types.SignedData{Type: keypgp.SignatureType, Signature: []byte("garbage")}
		commit := commitWith(sha1Hex, data, signerSig())
		require.NoError(t, f.svc.VerifyCommits(t.Context(), repoID, []*types.Commit{commit}))
		require.Equal(t, enum.GitSignatureInvalid, commit.Signature.Result)
	})
}

func TestVerifyCommits_Caching(t *testing.T) {
	m := newSSHMaterial(t, []byte("content"), 1)
	f := newVerifyFixture(m.publicKey)

	commits := []*types.Commit{
		commitWith(sha1Hex, m.data, signerSig()),
		commitWith(sha2Hex, m.data, signerSig()),
	}
	session := f.svc.NewVerifySession(repoID)
	require.NoError(t, session.VerifyCommits(t.Context(), commits))
	require.NoError(t, session.VerifyCommitTags(t.Context(), nil))

	for _, c := range commits {
		require.Equal(t, enum.GitSignatureGood, c.Signature.Result)
	}
	require.Equal(t, 1, f.principals.calls, "principal lookups are cached per email")
	require.Equal(t, 1, f.fetcher.calls, "key lookups are cached per key")

	session.StoreSignatures(t.Context())
	require.Len(t, f.sigs.created, 2)
}

func TestVerifyCommits_UnknownPrincipalIsCached(t *testing.T) {
	m := newSSHMaterial(t, []byte("content"), 1)
	f := newVerifyFixture(m.publicKey)

	other := &types.Signature{Identity: types.Identity{Email: "nobody@example.com"}}
	commits := []*types.Commit{commitWith(sha1Hex, m.data, other), commitWith(sha2Hex, m.data, other)}
	require.NoError(t, f.svc.VerifyCommits(t.Context(), repoID, commits))
	require.Equal(t, 1, f.principals.calls)
}

func TestVerifyCommits_UsesStoredResults(t *testing.T) {
	m := newSSHMaterial(t, []byte("content"), 1)
	f := newVerifyFixture(m.publicKey)
	stored := types.GitSignatureResult{Result: enum.GitSignatureGood, PrincipalID: 99}
	f.sigs.existing = map[sha.SHA]types.GitSignatureResult{sha.Must(sha1Hex): stored}

	commits := []*types.Commit{
		commitWith(sha1Hex, m.data, signerSig()),
		commitWith(sha2Hex, m.data, signerSig()),
	}
	require.NoError(t, f.svc.VerifyCommits(t.Context(), repoID, commits))

	require.Equal(t, int64(99), commits[0].Signature.PrincipalID, "taken from the database")
	require.Equal(t, int64(7), commits[1].Signature.PrincipalID, "verified from scratch")
	require.Len(t, f.sigs.created, 1)
}

func TestVerifyCommits_Errors(t *testing.T) {
	m := newSSHMaterial(t, []byte("content"), 1)
	boom := errors.New("boom")

	t.Run("signature store read", func(t *testing.T) {
		f := newVerifyFixture(m.publicKey)
		f.sigs.mapErr = boom
		err := f.svc.VerifyCommits(t.Context(), repoID, []*types.Commit{commitWith(sha1Hex, m.data, signerSig())})
		require.ErrorIs(t, err, boom)
	})

	t.Run("principal lookup", func(t *testing.T) {
		f := newVerifyFixture(m.publicKey)
		f.principals.err = boom
		err := f.svc.VerifyCommits(t.Context(), repoID, []*types.Commit{commitWith(sha1Hex, m.data, signerSig())})
		require.ErrorIs(t, err, boom)
	})

	t.Run("key fetch", func(t *testing.T) {
		f := newVerifyFixture(m.publicKey)
		f.fetcher.err = boom
		err := f.svc.VerifyCommits(t.Context(), repoID, []*types.Commit{commitWith(sha1Hex, m.data, signerSig())})
		require.ErrorIs(t, err, boom)
	})

	t.Run("failing to persist results is not fatal", func(t *testing.T) {
		f := newVerifyFixture(m.publicKey)
		f.sigs.createErr = boom
		commit := commitWith(sha1Hex, m.data, signerSig())
		require.NoError(t, f.svc.VerifyCommits(t.Context(), repoID, []*types.Commit{commit}))
		require.Equal(t, enum.GitSignatureGood, commit.Signature.Result)
	})
}
