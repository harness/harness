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

package publickey

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha512"
	"encoding/pem"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/harness/gitness/app/store"
	gitnesserrors "github.com/harness/gitness/errors"
	"github.com/harness/gitness/git/sha"
	gitnessstore "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gossh "golang.org/x/crypto/ssh"
)

const (
	svcSHAOne = "0000000000000000000000000000000000000001"
	svcSHATwo = "0000000000000000000000000000000000000002"
)

type svcSSHKey struct {
	signer gossh.Signer
	line   string
}

func svcNewSSHKey(t *testing.T) svcSSHKey {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	signer, err := gossh.NewSignerFromKey(priv)
	require.NoError(t, err)
	return svcSSHKey{signer: signer, line: string(gossh.MarshalAuthorizedKey(signer.PublicKey()))}
}

type svcSigBlob struct {
	MagicPreamble [6]byte
	Version       uint32
	PublicKey     []byte
	Namespace     string
	Reserved      string
	HashAlgorithm string
	Signature     []byte
}

type svcSigWrapper struct {
	Namespace     string
	Reserved      string
	HashAlgorithm string
	Hash          []byte
}

func svcSSHSign(t *testing.T, key svcSSHKey, content []byte) []byte {
	t.Helper()
	sum := sha512.Sum512(content)
	toSign := append([]byte("SSHSIG"), gossh.Marshal(svcSigWrapper{
		Namespace: "git", HashAlgorithm: "sha512", Hash: sum[:],
	})...)
	sig, err := key.signer.Sign(rand.Reader, toSign)
	require.NoError(t, err)

	blob := svcSigBlob{
		Version: 1, PublicKey: key.signer.PublicKey().Marshal(), Namespace: "git",
		HashAlgorithm: "sha512", Signature: gossh.Marshal(sig),
	}
	copy(blob.MagicPreamble[:], "SSHSIG")
	return pem.EncodeToMemory(&pem.Block{Type: "SSH SIGNATURE", Bytes: gossh.Marshal(blob)})
}

func svcNewPGPEntity(t *testing.T, email string) *openpgp.Entity {
	t.Helper()
	e, err := openpgp.NewEntity("Tester", "", email, &packet.Config{Algorithm: packet.PubKeyAlgoEdDSA})
	require.NoError(t, err)
	return e
}

func svcPGPPublic(t *testing.T, e *openpgp.Entity) string {
	t.Helper()
	buf := &bytes.Buffer{}
	w, err := armor.Encode(buf, openpgp.PublicKeyType, nil)
	require.NoError(t, err)
	require.NoError(t, e.Serialize(w))
	require.NoError(t, w.Close())
	return buf.String()
}

func svcPGPSign(t *testing.T, e *openpgp.Entity, content string) []byte {
	t.Helper()
	buf := &bytes.Buffer{}
	require.NoError(t, openpgp.ArmoredDetachSign(buf, e, strings.NewReader(content),
		&packet.Config{Algorithm: packet.PubKeyAlgoEdDSA}))
	return buf.Bytes()
}

func TestSvcParseString(t *testing.T) {
	ssh := svcNewSSHKey(t)
	pgp := svcPGPPublic(t, svcNewPGPEntity(t, "tester@example.com"))

	t.Run("empty", func(t *testing.T) {
		_, err := ParseString("", nil)
		require.Error(t, err)
		assert.Equal(t, gitnesserrors.StatusInvalidArgument, gitnesserrors.AsStatus(err))
	})
	t.Run("ssh key", func(t *testing.T) {
		key, err := ParseString(ssh.line, nil)
		require.NoError(t, err)
		assert.Equal(t, enum.PublicKeySchemeSSH, key.Scheme())
	})
	t.Run("pgp key", func(t *testing.T) {
		key, err := ParseString(strings.TrimSpace(pgp), &types.Principal{Email: "tester@example.com"})
		require.NoError(t, err)
		assert.Equal(t, enum.PublicKeySchemePGP, key.Scheme())
	})
	t.Run("pgp key of another user", func(t *testing.T) {
		_, err := ParseString(strings.TrimSpace(pgp), &types.Principal{Email: "other@example.com"})
		require.Error(t, err)
	})
	t.Run("broken pgp block", func(t *testing.T) {
		_, err := ParseString("-----BEGIN PGP PUBLIC KEY BLOCK-----\nxx\n-----END PGP PUBLIC KEY BLOCK-----", nil)
		require.Error(t, err)
	})
	t.Run("broken ssh key", func(t *testing.T) {
		_, err := ParseString("ssh-ed25519 !!!", nil)
		require.Error(t, err)
	})
}

type svcPublicKeyStore struct {
	store.PublicKeyStore
	keys        []types.PublicKey
	listErr     error
	markErr     error
	markedID    int64
	listUsages  []enum.PublicKeyUsage
	listPrincID *int64
}

func (s *svcPublicKeyStore) ListByFingerprint(
	_ context.Context, _ string, principalID *int64, usages []enum.PublicKeyUsage, _ []enum.PublicKeyScheme,
) ([]types.PublicKey, error) {
	s.listUsages = usages
	s.listPrincID = principalID
	return s.keys, s.listErr
}

func (s *svcPublicKeyStore) MarkAsVerified(_ context.Context, id int64, _ int64) error {
	s.markedID = id
	return s.markErr
}

type svcPrincipalInfoCache struct {
	store.PrincipalInfoCache
	info *types.PrincipalInfo
	err  error
}

func (c svcPrincipalInfoCache) Get(context.Context, int64) (*types.PrincipalInfo, error) {
	return c.info, c.err
}

func TestSvcSSHAuthValidateKey(t *testing.T) {
	ctx := context.Background()
	key := svcNewSSHKey(t)
	other := svcNewSSHKey(t)
	now := time.Now().UnixMilli()
	past := now - 100_000
	future := now + 100_000
	compromised := enum.RevocationReasonCompromised
	retired := enum.RevocationReasonRetired

	tests := []struct {
		name       string
		stored     types.PublicKey
		listErr    error
		markErr    error
		cacheErr   error
		presented  svcSSHKey
		wantStatus gitnesserrors.Status
		wantErr    bool
	}{
		{name: "valid", stored: types.PublicKey{ID: 1, Content: key.line, PrincipalID: 3}, presented: key},
		{
			name: "valid within period", presented: key,
			stored: types.PublicKey{ID: 1, Content: key.line, ValidFrom: &past, ValidTo: &future},
		},
		{
			name: "different key with same fingerprint row", presented: other,
			stored:     types.PublicKey{ID: 1, Content: key.line},
			wantStatus: gitnesserrors.StatusNotFound, wantErr: true,
		},
		{
			name: "compromised", presented: key,
			stored:     types.PublicKey{ID: 1, Content: key.line, RevocationReason: &compromised},
			wantStatus: gitnesserrors.StatusForbidden, wantErr: true,
		},
		{
			name: "not yet valid", presented: key,
			stored:     types.PublicKey{ID: 1, Content: key.line, ValidFrom: &future},
			wantStatus: gitnesserrors.StatusForbidden, wantErr: true,
		},
		{
			name: "expired", presented: key,
			stored:     types.PublicKey{ID: 1, Content: key.line, ValidTo: &past},
			wantStatus: gitnesserrors.StatusForbidden, wantErr: true,
		},
		{
			name: "expired and revoked", presented: key,
			stored:     types.PublicKey{ID: 1, Content: key.line, ValidTo: &past, RevocationReason: &retired},
			wantStatus: gitnesserrors.StatusForbidden, wantErr: true,
		},
		{
			name: "list failure", presented: key, listErr: errors.New("db down"), wantErr: true,
			stored: types.PublicKey{ID: 1, Content: key.line},
		},
		{
			name: "principal cache failure", presented: key, cacheErr: errors.New("cache down"), wantErr: true,
			stored: types.PublicKey{ID: 1, Content: key.line},
		},
		{
			name: "mark verified failure", presented: key, markErr: errors.New("db down"), wantErr: true,
			stored: types.PublicKey{ID: 1, Content: key.line},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keys := &svcPublicKeyStore{keys: []types.PublicKey{tt.stored}, listErr: tt.listErr, markErr: tt.markErr}
			svc := NewSSHAuthService(keys, svcPrincipalInfoCache{
				info: &types.PrincipalInfo{ID: 3, UID: "alice"}, err: tt.cacheErr,
			})

			info, err := svc.ValidateKey(ctx, "git", tt.presented.signer.PublicKey())
			if tt.wantErr {
				require.Error(t, err)
				if tt.wantStatus != "" {
					assert.Equal(t, tt.wantStatus, gitnesserrors.AsStatus(err))
				}
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "alice", info.UID)
			assert.Equal(t, int64(1), keys.markedID)
			assert.Nil(t, keys.listPrincID, "keys are looked up across all principals")
			assert.ElementsMatch(t, []enum.PublicKeyUsage{
				enum.PublicKeyUsageAuth, enum.PublicKeyUsageAuthSign,
			}, keys.listUsages)
		})
	}
}

type svcFetcher struct {
	keys  []types.PublicKey
	err   error
	calls int
}

func (f *svcFetcher) FetchByFingerprint(
	context.Context, string, int64, []enum.PublicKeyUsage, []enum.PublicKeyScheme,
) ([]types.PublicKey, error) {
	f.calls++
	return f.keys, f.err
}

func (f *svcFetcher) FetchBySubKeyID(
	context.Context, string, int64, []enum.PublicKeyUsage, []enum.PublicKeyScheme,
) ([]types.PublicKey, error) {
	f.calls++
	return f.keys, f.err
}

type svcPrincipalStore struct {
	store.PrincipalStore
	byEmail map[string]*types.Principal
	err     error
	calls   int
}

func (s *svcPrincipalStore) FindByEmail(_ context.Context, email string) (*types.Principal, error) {
	s.calls++
	if s.err != nil {
		return nil, s.err
	}
	p, ok := s.byEmail[email]
	if !ok {
		return nil, gitnessstore.ErrResourceNotFound
	}
	return p, nil
}

type svcSigStore struct {
	store.GitSignatureResultStore
	existing  map[sha.SHA]types.GitSignatureResult
	mapErr    error
	created   []*types.GitSignatureResult
	createErr error
}

func (s *svcSigStore) Map(context.Context, int64, []sha.SHA) (map[sha.SHA]types.GitSignatureResult, error) {
	return s.existing, s.mapErr
}

func (s *svcSigStore) TryCreateAll(_ context.Context, res []*types.GitSignatureResult) error {
	s.created = append(s.created, res...)
	return s.createErr
}

type svcVerifyEnv struct {
	svc        SignatureVerifyService
	principals *svcPrincipalStore
	fetcher    *svcFetcher
	sigs       *svcSigStore
}

func svcNewVerifyEnv() *svcVerifyEnv {
	env := &svcVerifyEnv{
		principals: &svcPrincipalStore{byEmail: map[string]*types.Principal{
			"alice@example.com": {ID: 3, Email: "alice@example.com"},
		}},
		fetcher: &svcFetcher{},
		sigs:    &svcSigStore{existing: map[sha.SHA]types.GitSignatureResult{}},
	}
	env.svc = NewSignatureVerifyService(env.principals, env.fetcher, env.sigs)
	return env
}

func svcSigner(email string) types.Signature {
	return types.Signature{
		Identity: types.Identity{Name: "Alice", Email: email},
		When:     time.Now(),
	}
}

func svcSSHCommit(t *testing.T, key svcSSHKey, shaStr, email string) *types.Commit {
	t.Helper()
	content := []byte("commit content " + shaStr)
	return &types.Commit{
		SHA:       sha.Must(shaStr),
		Committer: svcSigner(email),
		SignedData: &types.SignedData{
			Type: "SSH SIGNATURE", Signature: svcSSHSign(t, key, content), SignedContent: content,
		},
	}
}

func TestSvcVerifyCommits_SSH(t *testing.T) {
	ctx := context.Background()
	key := svcNewSSHKey(t)

	t.Run("good signature is stored", func(t *testing.T) {
		env := svcNewVerifyEnv()
		env.fetcher.keys = []types.PublicKey{{ID: 1, Content: key.line}}
		commit := svcSSHCommit(t, key, svcSHAOne, "alice@example.com")

		require.NoError(t, env.svc.VerifyCommits(ctx, 9, []*types.Commit{commit}))

		require.NotNil(t, commit.Signature)
		assert.Equal(t, enum.GitSignatureGood, commit.Signature.Result)
		assert.Equal(t, int64(3), commit.Signature.PrincipalID)
		assert.Equal(t, enum.PublicKeySchemeSSH, commit.Signature.KeyScheme)
		require.Len(t, env.sigs.created, 1)
		assert.Equal(t, int64(9), env.sigs.created[0].RepoID)
	})
	t.Run("signature from the database is reused", func(t *testing.T) {
		env := svcNewVerifyEnv()
		env.sigs.existing[sha.Must(svcSHAOne)] = types.GitSignatureResult{Result: enum.GitSignatureGood}
		commit := svcSSHCommit(t, key, svcSHAOne, "alice@example.com")

		require.NoError(t, env.svc.VerifyCommits(ctx, 9, []*types.Commit{commit}))

		require.NotNil(t, commit.Signature)
		assert.Equal(t, enum.GitSignatureGood, commit.Signature.Result)
		assert.Equal(t, 0, env.principals.calls, "no verification is done for persisted results")
		assert.Empty(t, env.sigs.created)
	})
	t.Run("commits without signature data are skipped", func(t *testing.T) {
		env := svcNewVerifyEnv()
		commit := &types.Commit{SHA: sha.Must(svcSHAOne)}
		require.NoError(t, env.svc.VerifyCommits(ctx, 9, []*types.Commit{commit}))
		assert.Nil(t, commit.Signature)
	})
	t.Run("unknown signer email is unverified and not stored", func(t *testing.T) {
		env := svcNewVerifyEnv()
		commit := svcSSHCommit(t, key, svcSHAOne, "stranger@example.com")
		require.NoError(t, env.svc.VerifyCommits(ctx, 9, []*types.Commit{commit}))
		assert.Equal(t, enum.GitSignatureUnverified, commit.Signature.Result)
		assert.Empty(t, env.sigs.created)
	})
	t.Run("key is not registered", func(t *testing.T) {
		env := svcNewVerifyEnv()
		commit := svcSSHCommit(t, key, svcSHAOne, "alice@example.com")
		require.NoError(t, env.svc.VerifyCommits(ctx, 9, []*types.Commit{commit}))
		assert.Equal(t, enum.GitSignatureUnverified, commit.Signature.Result)
	})
	t.Run("revoked key", func(t *testing.T) {
		env := svcNewVerifyEnv()
		reason := enum.RevocationReasonCompromised
		env.fetcher.keys = []types.PublicKey{{ID: 1, Content: key.line, RevocationReason: &reason}}
		commit := svcSSHCommit(t, key, svcSHAOne, "alice@example.com")
		require.NoError(t, env.svc.VerifyCommits(ctx, 9, []*types.Commit{commit}))
		assert.Equal(t, enum.GitSignatureRevoked, commit.Signature.Result)
		assert.Len(t, env.sigs.created, 1)
	})
	t.Run("key not yet valid or expired at commit time", func(t *testing.T) {
		now := time.Now().UnixMilli()
		future := now + 1_000_000
		past := now - 1_000_000

		env := svcNewVerifyEnv()
		env.fetcher.keys = []types.PublicKey{{ID: 1, Content: key.line, ValidFrom: &future}}
		commit := svcSSHCommit(t, key, svcSHAOne, "alice@example.com")
		require.NoError(t, env.svc.VerifyCommits(ctx, 9, []*types.Commit{commit}))
		assert.Equal(t, enum.GitSignatureKeyExpired, commit.Signature.Result)

		env = svcNewVerifyEnv()
		env.fetcher.keys = []types.PublicKey{{ID: 1, Content: key.line, ValidTo: &past}}
		commit = svcSSHCommit(t, key, svcSHAOne, "alice@example.com")
		require.NoError(t, env.svc.VerifyCommits(ctx, 9, []*types.Commit{commit}))
		assert.Equal(t, enum.GitSignatureKeyExpired, commit.Signature.Result)
	})
	t.Run("tampered content is bad", func(t *testing.T) {
		env := svcNewVerifyEnv()
		env.fetcher.keys = []types.PublicKey{{ID: 1, Content: key.line}}
		commit := svcSSHCommit(t, key, svcSHAOne, "alice@example.com")
		commit.SignedData.SignedContent = []byte("something else")
		require.NoError(t, env.svc.VerifyCommits(ctx, 9, []*types.Commit{commit}))
		assert.Equal(t, enum.GitSignatureBad, commit.Signature.Result)
	})
	t.Run("keys are cached per session", func(t *testing.T) {
		env := svcNewVerifyEnv()
		env.fetcher.keys = []types.PublicKey{{ID: 1, Content: key.line}}
		commits := []*types.Commit{
			svcSSHCommit(t, key, svcSHAOne, "alice@example.com"),
			svcSSHCommit(t, key, svcSHATwo, "alice@example.com"),
		}
		require.NoError(t, env.svc.VerifyCommits(ctx, 9, commits))
		assert.Equal(t, 1, env.fetcher.calls)
		assert.Equal(t, 1, env.principals.calls)
	})
	t.Run("invalid signature is not stored", func(t *testing.T) {
		env := svcNewVerifyEnv()
		commit := &types.Commit{
			SHA: sha.Must(svcSHAOne), Committer: svcSigner("alice@example.com"),
			SignedData: &types.SignedData{Type: "SSH SIGNATURE", Signature: []byte("junk")},
		}
		require.NoError(t, env.svc.VerifyCommits(ctx, 9, []*types.Commit{commit}))
		assert.Equal(t, enum.GitSignatureInvalid, commit.Signature.Result)
		assert.Empty(t, env.sigs.created)
	})
	t.Run("unsupported signature type", func(t *testing.T) {
		env := svcNewVerifyEnv()
		commit := &types.Commit{
			SHA: sha.Must(svcSHAOne), Committer: svcSigner("alice@example.com"),
			SignedData: &types.SignedData{Type: "X509", Signature: []byte("x")},
		}
		require.NoError(t, env.svc.VerifyCommits(ctx, 9, []*types.Commit{commit}))
		assert.Equal(t, enum.GitSignatureUnsupported, commit.Signature.Result)
	})
	t.Run("storing failures do not fail verification", func(t *testing.T) {
		env := svcNewVerifyEnv()
		env.sigs.createErr = errors.New("db down")
		env.fetcher.keys = []types.PublicKey{{ID: 1, Content: key.line}}
		commit := svcSSHCommit(t, key, svcSHAOne, "alice@example.com")
		require.NoError(t, env.svc.VerifyCommits(ctx, 9, []*types.Commit{commit}))
	})
}

func TestSvcVerifyCommits_Errors(t *testing.T) {
	ctx := context.Background()
	key := svcNewSSHKey(t)

	t.Run("reading stored signatures fails", func(t *testing.T) {
		env := svcNewVerifyEnv()
		env.sigs.mapErr = errors.New("db down")
		commit := svcSSHCommit(t, key, svcSHAOne, "alice@example.com")
		require.Error(t, env.svc.VerifyCommits(ctx, 9, []*types.Commit{commit}))
	})
	t.Run("principal lookup fails", func(t *testing.T) {
		env := svcNewVerifyEnv()
		env.principals.err = errors.New("db down")
		commit := svcSSHCommit(t, key, svcSHAOne, "alice@example.com")
		require.Error(t, env.svc.VerifyCommits(ctx, 9, []*types.Commit{commit}))
	})
	t.Run("key fetch fails", func(t *testing.T) {
		env := svcNewVerifyEnv()
		env.fetcher.err = errors.New("db down")
		commit := svcSSHCommit(t, key, svcSHAOne, "alice@example.com")
		require.Error(t, env.svc.VerifyCommits(ctx, 9, []*types.Commit{commit}))
	})
}

func TestSvcVerifyCommitTags(t *testing.T) {
	ctx := context.Background()
	key := svcNewSSHKey(t)

	t.Run("annotated tag with good signature", func(t *testing.T) {
		env := svcNewVerifyEnv()
		env.fetcher.keys = []types.PublicKey{{ID: 1, Content: key.line}}
		commit := svcSSHCommit(t, key, svcSHAOne, "alice@example.com")
		tagger := svcSigner("alice@example.com")
		tag := &types.CommitTag{
			SHA: commit.SHA, Tagger: &tagger, SignedData: commit.SignedData,
		}
		require.NoError(t, env.svc.VerifyCommitTags(ctx, 9, []*types.CommitTag{tag}))
		require.NotNil(t, tag.Signature)
		assert.Equal(t, enum.GitSignatureGood, tag.Signature.Result)
	})
	t.Run("tag without tagger is unverified", func(t *testing.T) {
		env := svcNewVerifyEnv()
		commit := svcSSHCommit(t, key, svcSHAOne, "alice@example.com")
		tag := &types.CommitTag{SHA: commit.SHA, SignedData: commit.SignedData}
		require.NoError(t, env.svc.VerifyCommitTags(ctx, 9, []*types.CommitTag{tag}))
		assert.Equal(t, enum.GitSignatureUnverified, tag.Signature.Result)
	})
	t.Run("session verifies commits and tags", func(t *testing.T) {
		env := svcNewVerifyEnv()
		env.fetcher.keys = []types.PublicKey{{ID: 1, Content: key.line}}
		session := env.svc.NewVerifySession(9)
		commit := svcSSHCommit(t, key, svcSHAOne, "alice@example.com")
		require.NoError(t, session.VerifyCommits(ctx, []*types.Commit{commit}))
		tagger := svcSigner("alice@example.com")
		tag := &types.CommitTag{SHA: commit.SHA, Tagger: &tagger, SignedData: commit.SignedData}
		require.NoError(t, session.VerifyCommitTags(ctx, []*types.CommitTag{tag}))
		session.StoreSignatures(ctx)
		assert.Len(t, env.sigs.created, 2)
	})
}

func TestSvcVerifyCommits_PGP(t *testing.T) {
	ctx := context.Background()
	entity := svcNewPGPEntity(t, "alice@example.com")
	const content = "pgp signed commit"

	newCommit := func() *types.Commit {
		return &types.Commit{
			SHA:       sha.Must(svcSHAOne),
			Committer: svcSigner("alice@example.com"),
			SignedData: &types.SignedData{
				Type: "PGP SIGNATURE", Signature: svcPGPSign(t, entity, content), SignedContent: []byte(content),
			},
		}
	}

	t.Run("good", func(t *testing.T) {
		env := svcNewVerifyEnv()
		env.fetcher.keys = []types.PublicKey{{ID: 1, Content: svcPGPPublic(t, entity)}}
		commit := newCommit()
		require.NoError(t, env.svc.VerifyCommits(ctx, 9, []*types.Commit{commit}))
		assert.Equal(t, enum.GitSignatureGood, commit.Signature.Result)
		assert.Equal(t, enum.PublicKeySchemePGP, commit.Signature.KeyScheme)
		assert.Equal(t, entity.PrimaryKey.KeyIdString(), commit.Signature.KeyID)
	})
	t.Run("key not registered", func(t *testing.T) {
		env := svcNewVerifyEnv()
		commit := newCommit()
		require.NoError(t, env.svc.VerifyCommits(ctx, 9, []*types.Commit{commit}))
		assert.Equal(t, enum.GitSignatureUnverified, commit.Signature.Result)
	})
}
