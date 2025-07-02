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
	"errors"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/harness/gitness/app/services/publickey/keypgp"
	"github.com/harness/gitness/app/services/publickey/keyssh"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/git/sha"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

type SignatureVerifyService struct {
	principalStore          store.PrincipalStore
	publicKeyStore          store.PublicKeyStore
	publicKeySubKeyStore    store.PublicKeySubKeyStore
	gitSignatureResultStore store.GitSignatureResultStore
}

func NewSignatureVerifyService(
	principalStore store.PrincipalStore,
	publicKeyStore store.PublicKeyStore,
	publicKeySubKeyStore store.PublicKeySubKeyStore,
	gitSignatureResultStore store.GitSignatureResultStore,
) SignatureVerifyService {
	return SignatureVerifyService{
		principalStore:          principalStore,
		publicKeyStore:          publicKeyStore,
		publicKeySubKeyStore:    publicKeySubKeyStore,
		gitSignatureResultStore: gitSignatureResultStore,
	}
}

// NewVerifySession creates a new session for git object signature verification.
// The session holds a small cache for users and signing keys.
func (s SignatureVerifyService) NewVerifySession(repoID int64) *VerifySession {
	return &VerifySession{
		SignatureVerifyService: s,
		repoID:                 repoID,
		principalIDCache:       make(map[string]int64),
		keyCache:               make(map[personalKey]*types.PublicKey),
	}
}

func (s SignatureVerifyService) VerifyCommitTags(ctx context.Context, repoID int64, tags []*types.CommitTag) error {
	session := s.NewVerifySession(repoID)
	if err := verifyObjects(ctx, session, tags); err != nil {
		return err
	}

	session.StoreSignatures(ctx)

	return nil
}

func (s SignatureVerifyService) VerifyCommits(ctx context.Context, repoID int64, commits []*types.Commit) error {
	session := s.NewVerifySession(repoID)
	if err := verifyObjects(ctx, session, commits); err != nil {
		return err
	}

	session.StoreSignatures(ctx)

	return nil
}

func (s *VerifySession) VerifyCommitTags(ctx context.Context, tags []*types.CommitTag) error {
	return verifyObjects(ctx, s, tags)
}

func (s *VerifySession) VerifyCommits(ctx context.Context, commits []*types.Commit) error {
	return verifyObjects(ctx, s, commits)
}

// VerifySession holds short time caches for a single iteration of verifyObject function.
type VerifySession struct {
	SignatureVerifyService
	repoID int64

	// principalIDCache is cache of principal IDs. The key is email address.
	principalIDCache map[string]int64

	// keyCache is cache of personal keys. The map key holds principalID, key ID and fingerprint.
	keyCache map[personalKey]*types.PublicKey

	// sigResults are git signature verification results that should be stored to the database.
	sigResults []*types.GitSignatureResult
}

// personalKey is cache key for the cache of public keys.
type personalKey struct {
	principalID    int64
	keyID          string
	keyFingerprint string
}

func (s *VerifySession) principalByEmail(ctx context.Context, email string) (int64, error) {
	if principalID, ok := s.principalIDCache[email]; ok {
		return principalID, nil
	}

	principal, err := s.principalStore.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, gitness_store.ErrResourceNotFound) {
			s.principalIDCache[email] = 0
			return 0, nil
		}

		return 0, err
	}

	s.principalIDCache[email] = principal.ID

	return principal.ID, nil
}

func (s *VerifySession) fetchKey(
	ctx context.Context,
	v verifier,
	k personalKey,
	principalID int64,
) (*types.PublicKey, error) {
	key, ok := s.keyCache[k]
	if ok {
		return key, nil
	}

	key, err := v.Key(ctx, s.publicKeyStore, principalID)
	if err != nil {
		return nil, fmt.Errorf("failed to get public key from verifier: %w", err)
	}

	s.keyCache[k] = key // We also store nils here to avoid searching for a non-existing key multiple times.

	return key, nil
}

func (s *VerifySession) StoreSignatures(ctx context.Context) {
	err := s.gitSignatureResultStore.TryCreateAll(ctx, s.sigResults)
	if err != nil {
		log.Ctx(ctx).Warn().Err(err).
			Msg("failed to create git signature results")
	}
}

func verifyObjects[T signedObject](ctx context.Context, session *VerifySession, objects []T) error {
	// Fill objects' signature data from the DB,
	// and get a map of objects without signature data in the DB.
	objectMap, err := fillSignatureFromDB(ctx, &session.SignatureVerifyService, session.repoID, objects)
	if err != nil {
		return fmt.Errorf("failed to backfill object signature info from the DB: %w", err)
	}

	for _, object := range objectMap {
		sigResult, err := verifyGitObjectSignature(ctx, session, object)
		if err != nil {
			return fmt.Errorf("failed to verify object signature: %w", err)
		}

		object.SetSignature(sigResult)

		// These we don't store to the database: Invalid, Unsupported and Unverified.
		// An invalid signature can mean not just that the signature contains garbage data, but also that
		// we failed to verify it because of a bug. So, we deliberately don't store them to the DB.
		if result := sigResult.Result; result == enum.GitSignatureInvalid ||
			result == enum.GitSignatureUnsupported ||
			result == enum.GitSignatureUnverified {
			continue
		}

		session.sigResults = append(session.sigResults, sigResult)
	}

	return nil
}

// fillSignatureFromDB reads git object signatures from the DB,
// updates the elements of the provided slice,
// and return a map of objects that do not yet have a signature in the DB.
func fillSignatureFromDB[T signedObject](
	ctx context.Context,
	s *SignatureVerifyService,
	repoID int64,
	objects []T,
) (map[sha.SHA]T, error) {
	objectMap := make(map[sha.SHA]T)
	for i := range objects {
		if objects[i].GetSignedData() != nil {
			objectMap[objects[i].GetSHA()] = objects[i]
		}
	}

	if len(objectMap) == 0 {
		return objectMap, nil
	}

	// Get slice of SHAs from the map.
	objectSHAs := slices.AppendSeq[[]sha.SHA](make([]sha.SHA, 0, len(objectMap)), maps.Keys(objectMap))

	// Read signature data from the tags from the DB.
	objectSignatureMap, err := s.gitSignatureResultStore.Map(ctx, repoID, objectSHAs)
	if err != nil {
		return nil, fmt.Errorf("failed to read commit signatures: %w", err)
	}

	// Update the objects found in the database and remove them from the map.
	for objectSHA, objectSignature := range objectSignatureMap {
		object := objectMap[objectSHA]
		object.SetSignature(&objectSignature)

		delete(objectMap, objectSHA)
	}

	return objectMap, nil
}

func verifyGitObjectSignature[T signedObject](
	ctx context.Context,
	s *VerifySession,
	object T,
) (*types.GitSignatureResult, error) {
	signedData := object.GetSignedData()
	if signedData == nil {
		return &sigVerUnverified, nil
	}

	var v verifier

	switch signedData.Type {
	case keyssh.SignatureType:
		v = &keyssh.Verify{}
	case keypgp.SignatureType:
		v = &keypgp.Verify{}
	default:
		return &sigVerUnsupported, nil // We mark unsupported signature types as unsupported.
	}

	// Get the object's signer email address - the committer for commits, the tagger for annotated tags.

	signer := object.GetSigner()
	if signer == nil {
		return &sigVerUnverified, nil
	}

	objectTime := signer.When.UnixMilli()
	email := signer.Identity.Email

	// Find the principal by the signer's email address.
	// If principal is not found the signature is unverified.

	principalID, err := s.principalByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to find principal by email: %w", err)
	}

	if principalID == 0 {
		return &sigVerUnverified, nil
	}

	// Find the key info from the signature.

	if result := v.Parse(ctx, object.GetSignedData().Signature, object.GetSHA()); result != "" {
		//nolint:exhaustive
		switch result {
		case enum.GitSignatureInvalid:
			return &sigVerInvalid, nil
		case enum.GitSignatureUnsupported:
			return &sigVerUnsupported, nil
		default:
			// Should not happen.
			return nil, fmt.Errorf("unexpected signature verification result=%q after signature parsing", result)
		}
	}

	// Fetch the key from the DB. If it's not there, the signature is unverified.

	key, err := s.fetchKey(ctx, v, personalKey{
		principalID:    principalID,
		keyID:          v.KeyID(),
		keyFingerprint: v.KeyFingerprint(),
	}, principalID)
	if err != nil {
		return nil, fmt.Errorf("failed to get public key: %w", err)
	}
	if key == nil {
		return &sigVerUnverified, nil
	}

	now := time.Now().UnixMilli()
	sigResult := &types.GitSignatureResult{
		RepoID:         s.repoID,
		ObjectSHA:      object.GetSHA(),
		ObjectTime:     objectTime,
		Created:        now,
		Updated:        now,
		Result:         "", // the result will be set later
		PrincipalID:    principalID,
		KeyScheme:      v.KeyScheme(),
		KeyID:          v.KeyID(),
		KeyFingerprint: v.KeyFingerprint(),
	}

	// Using the key's properties, if possible override the verification result.

	switch {
	case key.RevocationReason != nil:
		sigResult.Result = enum.GitSignatureRevoked
		return sigResult, nil
	case key.ValidFrom != nil && objectTime < *key.ValidFrom:
		sigResult.Result = enum.GitSignatureKeyExpired
		return sigResult, nil
	case key.ValidTo != nil && objectTime > *key.ValidTo:
		sigResult.Result = enum.GitSignatureKeyExpired
		return sigResult, nil
	}

	// Verify the git object signature using the key from the database.

	sigResult.Result = v.Verify(
		ctx,
		[]byte(key.Content),
		object.GetSignedData().SignedContent,
		object.GetSHA(),
		*signer)

	return sigResult, nil
}

// verifier is interface to verify a git object signature.
// It's implemented by keypgp.Verify and keyssh.Verify.
type verifier interface {
	// Parse parses the provided signature and extracts info about the signing key (ID/Fingerprint).
	Parse(
		ctx context.Context,
		signature []byte,
		objectSHA sha.SHA,
	) enum.GitSignatureResult

	// Key fetches the key from the DB.
	Key(
		ctx context.Context,
		publicKeyStore store.PublicKeyStore,
		principalID int64,
	) (*types.PublicKey, error)

	// Verify checks if the signed content matches signature.
	Verify(
		ctx context.Context,
		key []byte,
		signedContent []byte,
		objectSHA sha.SHA,
		committer types.Signature,
	) enum.GitSignatureResult

	// KeyScheme returns the signing key's scheme.
	KeyScheme() enum.PublicKeyScheme

	// KeyID returns the signing key ID. Use after a call to the Parse method
	KeyID() string

	// KeyFingerprint returns the signing key fingerprint. Use after a call to the Parse method
	KeyFingerprint() string
}

// signedObject is interface used to verify signature.
// It's implemented by types.Commit and types.CommitTag.
type signedObject interface {
	GetSHA() sha.SHA
	SetSignature(sig *types.GitSignatureResult)
	GetSigner() *types.Signature
	GetSignedData() *types.SignedData
}

var sigVerUnverified = types.GitSignatureResult{
	Result: enum.GitSignatureUnverified,
}

var sigVerUnsupported = types.GitSignatureResult{
	Result: enum.GitSignatureUnsupported,
}

var sigVerInvalid = types.GitSignatureResult{
	Result: enum.GitSignatureInvalid,
}
