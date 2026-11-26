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

package database_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/harness/gitness/app/store/database"
	gitness_errors "github.com/harness/gitness/errors"
	"github.com/harness/gitness/git/sha"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

func prqNewSignatureResult(repoID int64, objectSHA string, principalID int64) types.GitSignatureResult {
	return types.GitSignatureResult{
		RepoID:         repoID,
		ObjectSHA:      sha.Must(objectSHA),
		ObjectTime:     prqBaseTime,
		Created:        prqBaseTime,
		Updated:        prqBaseTime,
		Result:         enum.GitSignatureGood,
		PrincipalID:    principalID,
		KeyScheme:      enum.PublicKeySchemePGP,
		KeyID:          "key-" + objectSHA[:4],
		KeyFingerprint: "fingerprint-" + objectSHA[:4],
	}
}

func TestPrqGitSignatureResultStore_CreateAndMap(t *testing.T) {
	f := prqNewFixture(t)
	store := database.NewGitSignatureResultStore(f.db)

	first := prqNewSignatureResult(prqRepoMain, prqSHA1, prqUserA)
	second := prqNewSignatureResult(prqRepoMain, prqSHA3, prqUserB)
	otherRepo := prqNewSignatureResult(prqRepoOther, prqSHA1, prqUserA)
	for _, res := range []types.GitSignatureResult{first, second, otherRepo} {
		require.NoError(t, store.Create(f.ctx, res))
	}

	err := store.Create(f.ctx, first)
	require.ErrorIs(t, err, gitness_store.ErrDuplicate)

	got, err := store.Map(f.ctx, prqRepoMain, []sha.SHA{first.ObjectSHA, second.ObjectSHA, sha.Must(prqSHA4)})
	require.NoError(t, err)
	require.Len(t, got, 2)
	require.Equal(t, first, got[first.ObjectSHA])
	require.Equal(t, second, got[second.ObjectSHA])

	got, err = store.Map(f.ctx, prqRepoOther, []sha.SHA{first.ObjectSHA, second.ObjectSHA})
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, otherRepo, got[first.ObjectSHA])

	got, err = store.Map(f.ctx, prqRepoMain, []sha.SHA{sha.Must(prqSHA4)})
	require.NoError(t, err)
	require.Empty(t, got)
}

func TestPrqGitSignatureResultStore_TryCreateAll(t *testing.T) {
	f := prqNewFixture(t)
	store := database.NewGitSignatureResultStore(f.db)

	require.NoError(t, store.TryCreateAll(f.ctx, nil))

	existing := prqNewSignatureResult(prqRepoMain, prqSHA1, prqUserA)
	require.NoError(t, store.Create(f.ctx, existing))

	clashing := prqNewSignatureResult(prqRepoMain, prqSHA1, prqUserB)
	clashing.Result = enum.GitSignatureBad
	fresh := prqNewSignatureResult(prqRepoMain, prqSHA3, prqUserB)

	// conflicting rows are skipped, the others are inserted.
	require.NoError(t, store.TryCreateAll(f.ctx, []*types.GitSignatureResult{&clashing, &fresh}))

	got, err := store.Map(f.ctx, prqRepoMain, []sha.SHA{existing.ObjectSHA, fresh.ObjectSHA})
	require.NoError(t, err)
	require.Len(t, got, 2)
	require.Equal(t, existing, got[existing.ObjectSHA])
	require.Equal(t, fresh, got[fresh.ObjectSHA])

	bad := prqNewSignatureResult(9999, prqSHA4, prqUserA)
	err = store.TryCreateAll(f.ctx, []*types.GitSignatureResult{&bad})
	require.ErrorIs(t, err, gitness_store.ErrForeignKeyViolation)
}

func TestPrqGitSignatureResultStore_UpdateAll(t *testing.T) {
	f := prqNewFixture(t)
	store := database.NewGitSignatureResultStore(f.db)

	// three signatures of user A (two keys) and one of user B.
	a1 := prqNewSignatureResult(prqRepoMain, prqSHA1, prqUserA)
	a2 := prqNewSignatureResult(prqRepoMain, prqSHA3, prqUserA)
	a2.KeyID = a1.KeyID
	a2.KeyFingerprint = a1.KeyFingerprint
	a3 := prqNewSignatureResult(prqRepoMain, prqSHA4, prqUserA)
	a3.KeyID = "other-key"
	a3.KeyFingerprint = "other-fingerprint"
	b1 := prqNewSignatureResult(prqRepoMain, prqSHA5, prqUserB)
	for _, res := range []types.GitSignatureResult{a1, a2, a3, b1} {
		require.NoError(t, store.Create(f.ctx, res))
	}

	results := func() map[string]enum.GitSignatureResult {
		t.Helper()
		got, err := store.Map(f.ctx, prqRepoMain, []sha.SHA{a1.ObjectSHA, a2.ObjectSHA, a3.ObjectSHA, b1.ObjectSHA})
		require.NoError(t, err)
		out := map[string]enum.GitSignatureResult{}
		for objectSHA, res := range got {
			out[objectSHA.String()] = res.Result
		}
		return out
	}

	// by key id.
	require.NoError(t, store.UpdateAll(f.ctx, enum.GitSignatureKeyExpired, prqUserA, []string{a1.KeyID}, nil))
	require.Equal(t, map[string]enum.GitSignatureResult{
		prqSHA1: enum.GitSignatureKeyExpired,
		prqSHA3: enum.GitSignatureKeyExpired, // same key as a1
		prqSHA4: enum.GitSignatureGood,
		prqSHA5: enum.GitSignatureGood,
	}, results())

	// by fingerprint.
	require.NoError(t, store.UpdateAll(f.ctx, enum.GitSignatureBad, prqUserA, nil, []string{a3.KeyFingerprint}))
	require.Equal(t, enum.GitSignatureBad, results()[prqSHA4])
	require.Equal(t, enum.GitSignatureKeyExpired, results()[prqSHA1])

	// no key filters: everything of the principal changes, nothing of other principals.
	require.NoError(t, store.UpdateAll(f.ctx, enum.GitSignatureUnverified, prqUserA, nil, nil))
	require.Equal(t, map[string]enum.GitSignatureResult{
		prqSHA1: enum.GitSignatureUnverified,
		prqSHA3: enum.GitSignatureUnverified,
		prqSHA4: enum.GitSignatureUnverified,
		prqSHA5: enum.GitSignatureGood,
	}, results())
}

func TestPrqGitSignatureResultStore_Delete(t *testing.T) {
	f := prqNewFixture(t)
	store := database.NewGitSignatureResultStore(f.db)

	a1 := prqNewSignatureResult(prqRepoMain, prqSHA1, prqUserA)
	a2 := prqNewSignatureResult(prqRepoMain, prqSHA3, prqUserA)
	a3 := prqNewSignatureResult(prqRepoMain, prqSHA4, prqUserA)
	b1 := prqNewSignatureResult(prqRepoMain, prqSHA5, prqUserB)
	b1.KeyID = a1.KeyID
	b1.KeyFingerprint = a1.KeyFingerprint
	for _, res := range []types.GitSignatureResult{a1, a2, a3, b1} {
		require.NoError(t, store.Create(f.ctx, res))
	}

	remaining := func() int {
		t.Helper()
		got, err := store.Map(f.ctx, prqRepoMain,
			[]sha.SHA{a1.ObjectSHA, a2.ObjectSHA, a3.ObjectSHA, b1.ObjectSHA})
		require.NoError(t, err)
		return len(got)
	}

	// empty inputs delete nothing.
	n, err := store.DeleteByKeyIDs(f.ctx, prqUserA, nil)
	require.NoError(t, err)
	require.Equal(t, int64(0), n)
	n, err = store.DeleteByKeyFingerprints(f.ctx, prqUserA, nil)
	require.NoError(t, err)
	require.Equal(t, int64(0), n)
	require.Equal(t, 4, remaining())

	// only signatures of the given principal are removed.
	n, err = store.DeleteByKeyIDs(f.ctx, prqUserA, []string{a1.KeyID, "unknown"})
	require.NoError(t, err)
	require.Equal(t, int64(1), n)
	require.Equal(t, 3, remaining())

	n, err = store.DeleteByKeyFingerprints(f.ctx, prqUserA, []string{a2.KeyFingerprint, a3.KeyFingerprint})
	require.NoError(t, err)
	require.Equal(t, int64(2), n)
	require.Equal(t, 1, remaining())

	n, err = store.DeleteByKeyFingerprints(f.ctx, prqUserA, []string{a2.KeyFingerprint})
	require.NoError(t, err)
	require.Equal(t, int64(0), n)
}

func prqNewRepoActivity(repoID int64, key string, payload types.RepoActivityPayload, at int64) *types.RepoActivity {
	return &types.RepoActivity{
		RepoID:      repoID,
		PrincipalID: prqUserA,
		Key:         key,
		Type:        payload.ActivityType(),
		Payload:     payload,
		Timestamp:   at,
	}
}

func TestPrqRepoActivityStore_CreateListCount(t *testing.T) {
	f := prqNewFixture(t)
	store := database.NewRepoActivityStore(f.db)

	created := prqNewRepoActivity(prqRepoMain, "k1", &types.RepoActivityPayloadBranchCreated{Name: "b", New: prqSHA1}, 100)
	updated := prqNewRepoActivity(prqRepoMain, "k2", &types.RepoActivityPayloadBranchUpdated{
		Name: "b", Old: prqSHA1, New: prqSHA3, Forced: true,
	}, 200)
	deleted := prqNewRepoActivity(prqRepoMain, "k3", &types.RepoActivityPayloadBranchDeleted{Name: "b", Old: prqSHA3}, 300)
	other := prqNewRepoActivity(prqRepoOther, "k4", &types.RepoActivityPayloadBranchCreated{Name: "x", New: prqSHA4}, 150)
	for _, act := range []*types.RepoActivity{deleted, created, updated, other} {
		require.NoError(t, store.Create(f.ctx, act))
	}

	// a repeated key is ignored.
	dup := prqNewRepoActivity(prqRepoMain, "k1", &types.RepoActivityPayloadBranchCreated{Name: "changed"}, 999)
	require.NoError(t, store.Create(f.ctx, dup))

	tests := []struct {
		name   string
		filter types.RepoActivityFilter
		want   []*types.RepoActivity
	}{
		{name: "all", filter: types.RepoActivityFilter{Size: 10}, want: []*types.RepoActivity{created, updated, deleted}},
		{
			name:   "after",
			filter: types.RepoActivityFilter{Size: 10, After: 100},
			want:   []*types.RepoActivity{updated, deleted},
		},
		{
			name:   "before",
			filter: types.RepoActivityFilter{Size: 10, Before: 300},
			want:   []*types.RepoActivity{created, updated},
		},
		{
			name:   "range",
			filter: types.RepoActivityFilter{Size: 10, After: 100, Before: 300},
			want:   []*types.RepoActivity{updated},
		},
		{name: "page 1", filter: types.RepoActivityFilter{Size: 2, Page: 1}, want: []*types.RepoActivity{created, updated}},
		{name: "page 2", filter: types.RepoActivityFilter{Size: 2, Page: 2}, want: []*types.RepoActivity{deleted}},
		{name: "page 3", filter: types.RepoActivityFilter{Size: 2, Page: 3}, want: []*types.RepoActivity{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			list, err := store.List(f.ctx, prqRepoMain, &tt.filter)
			require.NoError(t, err)
			require.Equal(t, tt.want, list)
		})
	}

	count, err := store.Count(f.ctx, prqRepoMain, &types.RepoActivityFilter{})
	require.NoError(t, err)
	require.Equal(t, 3, count)

	count, err = store.Count(f.ctx, prqRepoMain, &types.RepoActivityFilter{After: 100, Before: 300})
	require.NoError(t, err)
	require.Equal(t, 1, count)

	count, err = store.Count(f.ctx, prqRepoOther, &types.RepoActivityFilter{})
	require.NoError(t, err)
	require.Equal(t, 1, count)

	count, err = store.Count(f.ctx, 9999, &types.RepoActivityFilter{})
	require.NoError(t, err)
	require.Equal(t, 0, count)
}

func TestPrqRepoActivityStore_ErrorsAndNilPayload(t *testing.T) {
	f := prqNewFixture(t)
	store := database.NewRepoActivityStore(f.db)

	// the payload has to match the activity type.
	mismatch := prqNewRepoActivity(prqRepoMain, "k1", &types.RepoActivityPayloadBranchCreated{Name: "b"}, 100)
	mismatch.Type = enum.RepoActivityTypeBranchDeleted
	require.ErrorContains(t, store.Create(f.ctx, mismatch), "payload type mismatch")

	// a missing payload is stored as an empty object and read back as an empty typed payload.
	empty := &types.RepoActivity{
		RepoID:      prqRepoMain,
		PrincipalID: prqUserA,
		Key:         "k2",
		Type:        enum.RepoActivityTypeBranchUpdated,
		Timestamp:   200,
	}
	require.NoError(t, store.Create(f.ctx, empty))

	list, err := store.List(f.ctx, prqRepoMain, &types.RepoActivityFilter{Size: 10})
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, &types.RepoActivityPayloadBranchUpdated{}, list[0].Payload)

	unknownRepo := prqNewRepoActivity(9999, "k3", &types.RepoActivityPayloadBranchCreated{Name: "b"}, 100)
	require.ErrorIs(t, store.Create(f.ctx, unknownRepo), gitness_store.ErrForeignKeyViolation)

	// activities of an unknown type can't be read back.
	_, err = f.db.ExecContext(f.ctx, `INSERT INTO repo_activities (
		repo_activity_key, repo_activity_repo_id, repo_activity_principal_id,
		repo_activity_type, repo_activity_payload, repo_activity_created
	) VALUES ('k4', ?, ?, 'bogus', ?, 300)`, prqRepoMain, prqUserA, []byte("{}"))
	require.NoError(t, err)

	_, err = store.List(f.ctx, prqRepoMain, &types.RepoActivityFilter{Size: 10})
	require.ErrorContains(t, err, "unknown repo activity type")
}

func TestPrqRepoLangStore_Lifecycle(t *testing.T) {
	f := prqNewFixture(t)
	store := database.NewRepoLangStore(f.db)

	// nothing to insert.
	require.NoError(t, store.InsertByRepoID(f.ctx, prqRepoMain, nil))

	list, err := store.ListByRepoID(f.ctx, prqRepoMain)
	require.NoError(t, err)
	require.Empty(t, list)

	require.NoError(t, store.InsertByRepoID(f.ctx, prqRepoMain, []*types.RepoLangStat{
		{Language: "Go", Bytes: 500, Files: 5},
		{Language: "Python", Bytes: 900, Files: 9},
		{Language: "C", Bytes: 500, Files: 2},
	}))
	require.NoError(t, store.InsertByRepoID(f.ctx, prqRepoOther, []*types.RepoLangStat{
		{Language: "Go", Bytes: 1, Files: 1},
	}))

	// ordered by bytes descending and then by language.
	list, err = store.ListByRepoID(f.ctx, prqRepoMain)
	require.NoError(t, err)
	require.Equal(t, []*types.RepoLangStat{
		{Language: "Python", Bytes: 900, Files: 9},
		{Language: "C", Bytes: 500, Files: 2},
		{Language: "Go", Bytes: 500, Files: 5},
	}, list)

	// a repository can't have the same language twice.
	err = store.InsertByRepoID(f.ctx, prqRepoMain, []*types.RepoLangStat{{Language: "Go", Bytes: 1, Files: 1}})
	require.ErrorIs(t, err, gitness_store.ErrDuplicate)

	err = store.InsertByRepoID(f.ctx, 9999, []*types.RepoLangStat{{Language: "Go", Bytes: 1, Files: 1}})
	require.ErrorIs(t, err, gitness_store.ErrForeignKeyViolation)

	require.NoError(t, store.DeleteByRepoID(f.ctx, prqRepoMain))

	list, err = store.ListByRepoID(f.ctx, prqRepoMain)
	require.NoError(t, err)
	require.Empty(t, list)

	list, err = store.ListByRepoID(f.ctx, prqRepoOther)
	require.NoError(t, err)
	require.Len(t, list, 1)

	require.NoError(t, store.DeleteByRepoID(f.ctx, prqRepoMain))
}

func prqNewPublicKey(principalID int64, identifier, fingerprint string, created int64) *types.PublicKey {
	return &types.PublicKey{
		PrincipalID: principalID,
		Created:     created,
		Identifier:  identifier,
		Usage:       enum.PublicKeyUsageAuth,
		Fingerprint: fingerprint,
		Content:     "key content " + identifier,
		Comment:     "comment " + identifier,
		Type:        "ssh-ed25519",
		Scheme:      enum.PublicKeySchemeSSH,
		Metadata:    json.RawMessage("{}"),
	}
}

func TestPrqPublicKeyStore_CreateFindUpdate(t *testing.T) {
	f := prqNewFixture(t)
	store := database.NewPublicKeyStore(f.db)

	verified := prqBaseTime + 5
	validFrom := prqBaseTime
	validTo := prqBaseTime + 1000
	reason := enum.RevocationReasonRetired
	in := prqNewPublicKey(prqUserA, "My-Key", "fp-one", prqBaseTime)
	in.Usage = enum.PublicKeyUsageSign
	in.Scheme = enum.PublicKeySchemePGP
	in.Verified = &verified
	in.ValidFrom = &validFrom
	in.ValidTo = &validTo
	in.RevocationReason = &reason
	in.Metadata = json.RawMessage(`{"email":"a@example.com"}`)
	require.NoError(t, store.Create(f.ctx, in))
	require.NotZero(t, in.ID)

	byID, err := store.Find(f.ctx, in.ID)
	require.NoError(t, err)
	require.Equal(t, in.ID, byID.ID)
	require.Equal(t, prqUserA, byID.PrincipalID)
	require.Equal(t, "My-Key", byID.Identifier)
	require.Equal(t, enum.PublicKeyUsageSign, byID.Usage)
	require.Equal(t, enum.PublicKeySchemePGP, byID.Scheme)
	require.Equal(t, "fp-one", byID.Fingerprint)
	require.Equal(t, "key content My-Key", byID.Content)
	require.Equal(t, "comment My-Key", byID.Comment)
	require.Equal(t, "ssh-ed25519", byID.Type)
	require.Equal(t, &verified, byID.Verified)
	require.Equal(t, &validFrom, byID.ValidFrom)
	require.Equal(t, &validTo, byID.ValidTo)
	require.NotNil(t, byID.RevocationReason)
	require.Equal(t, enum.RevocationReasonRetired, *byID.RevocationReason)
	require.JSONEq(t, `{"email":"a@example.com"}`, string(byID.Metadata))

	// identifiers are case insensitive.
	byIdentifier, err := store.FindByIdentifier(f.ctx, prqUserA, "my-KEY")
	require.NoError(t, err)
	require.Equal(t, in.ID, byIdentifier.ID)

	_, err = store.FindByIdentifier(f.ctx, prqUserB, "my-key")
	require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)
	_, err = store.Find(f.ctx, 9999)
	require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)

	// update changes only validity and revocation data.
	newTo := prqBaseTime + 2000
	newReason := enum.RevocationReasonCompromised
	byID.ValidTo = &newTo
	byID.RevocationReason = &newReason
	byID.Comment = "ignored"
	require.NoError(t, store.Update(f.ctx, byID))

	updated, err := store.Find(f.ctx, in.ID)
	require.NoError(t, err)
	require.Equal(t, &newTo, updated.ValidTo)
	require.Equal(t, enum.RevocationReasonCompromised, *updated.RevocationReason)
	require.Equal(t, "comment My-Key", updated.Comment)

	// clearing the revocation.
	updated.RevocationReason = nil
	updated.ValidTo = nil
	require.NoError(t, store.Update(f.ctx, updated))
	cleared, err := store.Find(f.ctx, in.ID)
	require.NoError(t, err)
	require.Nil(t, cleared.RevocationReason)
	require.Nil(t, cleared.ValidTo)
}

func TestPrqPublicKeyStore_CreateErrors(t *testing.T) {
	f := prqNewFixture(t)
	store := database.NewPublicKeyStore(f.db)

	require.NoError(t, store.Create(f.ctx, prqNewPublicKey(prqUserA, "Key", "fp", prqBaseTime)))

	err := store.Create(f.ctx, prqNewPublicKey(prqUserA, "KEY", "fp2", prqBaseTime))
	require.ErrorIs(t, err, gitness_store.ErrDuplicate)

	// other users can use the same identifier.
	require.NoError(t, store.Create(f.ctx, prqNewPublicKey(prqUserB, "Key", "fp", prqBaseTime)))

	err = store.Create(f.ctx, prqNewPublicKey(9999, "Key", "fp", prqBaseTime))
	require.ErrorIs(t, err, gitness_store.ErrForeignKeyViolation)
}

func TestPrqPublicKeyStore_MarkAsVerifiedAndDelete(t *testing.T) {
	f := prqNewFixture(t)
	store := database.NewPublicKeyStore(f.db)

	key := prqNewPublicKey(prqUserA, "Key", "fp", prqBaseTime)
	require.NoError(t, store.Create(f.ctx, key))

	require.NoError(t, store.MarkAsVerified(f.ctx, key.ID, prqBaseTime+77))
	got, err := store.Find(f.ctx, key.ID)
	require.NoError(t, err)
	require.NotNil(t, got.Verified)
	require.Equal(t, prqBaseTime+77, *got.Verified)

	// another principal's key with the same identifier is not removed.
	require.NoError(t, store.Create(f.ctx, prqNewPublicKey(prqUserB, "Key", "fp", prqBaseTime)))

	require.NoError(t, store.DeleteByIdentifier(f.ctx, prqUserA, "KEY"))
	_, err = store.Find(f.ctx, key.ID)
	require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)

	_, err = store.FindByIdentifier(f.ctx, prqUserB, "key")
	require.NoError(t, err)

	err = store.DeleteByIdentifier(f.ctx, prqUserA, "key")
	require.Error(t, err)
	require.True(t, gitness_errors.IsNotFound(err))
}

type prqKeyFixture struct {
	*prqFixture
	store     database.PublicKeyStore
	subKeys   database.PublicKeySubKeyStore
	keysByKey map[string]*types.PublicKey
}

func prqNewKeyFixture(t *testing.T) *prqKeyFixture {
	t.Helper()

	f := prqNewFixture(t)
	kf := &prqKeyFixture{
		prqFixture: f,
		store:      database.NewPublicKeyStore(f.db),
		subKeys:    database.NewPublicKeySubKeyStore(f.db),
		keysByKey:  map[string]*types.PublicKey{},
	}

	add := func(key string, principal int64, fingerprint string, created int64, mutate func(k *types.PublicKey)) {
		k := prqNewPublicKey(principal, key, fingerprint, created)
		if mutate != nil {
			mutate(k)
		}
		require.NoError(t, kf.store.Create(f.ctx, k), key)
		kf.keysByKey[key] = k
	}

	add("alpha", prqUserA, "fp-a", 100, nil)
	add("beta", prqUserA, "fp-b", 200, func(k *types.PublicKey) {
		k.Usage = enum.PublicKeyUsageSign
		k.Scheme = enum.PublicKeySchemePGP
	})
	add("gamma", prqUserA, "fp-b", 300, func(k *types.PublicKey) {
		k.Usage = enum.PublicKeyUsageAuthSign
	})
	add("delta", prqUserB, "fp-d", 400, func(k *types.PublicKey) {
		k.Scheme = enum.PublicKeySchemePGP
	})

	return kf
}

func (kf *prqKeyFixture) identifiers(keys []types.PublicKey) []string {
	out := make([]string, len(keys))
	for i, k := range keys {
		out[i] = k.Identifier
	}
	return out
}

func TestPrqPublicKeyStore_ListAndCount(t *testing.T) {
	kf := prqNewKeyFixture(t)
	principalA := prqUserA

	newFilter := func(mutate func(f *types.PublicKeyFilter)) *types.PublicKeyFilter {
		filter := &types.PublicKeyFilter{Sort: enum.PublicKeySortCreated, Order: enum.OrderAsc}
		if mutate != nil {
			mutate(filter)
		}
		return filter
	}

	tests := []struct {
		name      string
		principal *int64
		filter    *types.PublicKeyFilter
		want      []string
	}{
		{name: "all keys of everybody", filter: newFilter(nil), want: []string{"alpha", "beta", "gamma", "delta"}},
		{
			name:      "keys of a principal",
			principal: &principalA,
			filter:    newFilter(nil),
			want:      []string{"alpha", "beta", "gamma"},
		},
		{
			name:      "sorted by identifier descending",
			principal: &principalA,
			filter: newFilter(func(f *types.PublicKeyFilter) {
				f.Sort = enum.PublicKeySortIdentifier
				f.Order = enum.OrderDesc
			}),
			want: []string{"gamma", "beta", "alpha"},
		},
		{
			name:      "default order is ascending",
			principal: &principalA,
			filter: newFilter(func(f *types.PublicKeyFilter) {
				f.Sort = enum.PublicKeySortIdentifier
				f.Order = enum.OrderDefault
			}),
			want: []string{"alpha", "beta", "gamma"},
		},
		{
			name:   "query",
			filter: newFilter(func(f *types.PublicKeyFilter) { f.Query = "ALP" }),
			want:   []string{"alpha"},
		},
		{
			name:   "single usage",
			filter: newFilter(func(f *types.PublicKeyFilter) { f.Usages = []enum.PublicKeyUsage{enum.PublicKeyUsageSign} }),
			want:   []string{"beta"},
		},
		{
			name: "multiple usages",
			filter: newFilter(func(f *types.PublicKeyFilter) {
				f.Usages = []enum.PublicKeyUsage{enum.PublicKeyUsageSign, enum.PublicKeyUsageAuthSign}
			}),
			want: []string{"beta", "gamma"},
		},
		{
			name:   "single scheme",
			filter: newFilter(func(f *types.PublicKeyFilter) { f.Schemes = []enum.PublicKeyScheme{enum.PublicKeySchemePGP} }),
			want:   []string{"beta", "delta"},
		},
		{
			name: "multiple schemes",
			filter: newFilter(func(f *types.PublicKeyFilter) {
				f.Schemes = []enum.PublicKeyScheme{enum.PublicKeySchemePGP, enum.PublicKeySchemeSSH}
			}),
			want: []string{"alpha", "beta", "gamma", "delta"},
		},
		{
			name:      "pagination",
			principal: &principalA,
			filter: newFilter(func(f *types.PublicKeyFilter) {
				f.Page = 2
				f.Size = 2
			}),
			want: []string{"gamma"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			list, err := kf.store.List(kf.ctx, tt.principal, tt.filter)
			require.NoError(t, err)
			require.Equal(t, tt.want, kf.identifiers(list))
		})
	}

	// count ignores sorting and pagination.
	count, err := kf.store.Count(kf.ctx, nil, &types.PublicKeyFilter{})
	require.NoError(t, err)
	require.Equal(t, 4, count)

	count, err = kf.store.Count(kf.ctx, &principalA, &types.PublicKeyFilter{
		Schemes: []enum.PublicKeyScheme{enum.PublicKeySchemePGP},
	})
	require.NoError(t, err)
	require.Equal(t, 1, count)

	count, err = kf.store.Count(kf.ctx, &principalA, &types.PublicKeyFilter{
		ListQueryFilter: types.ListQueryFilter{Query: "a"},
		Usages:          []enum.PublicKeyUsage{enum.PublicKeyUsageAuth, enum.PublicKeyUsageSign},
	})
	require.NoError(t, err)
	require.Equal(t, 2, count)
}

func TestPrqPublicKeyStore_ListByFingerprintAndSubKey(t *testing.T) {
	kf := prqNewKeyFixture(t)
	principalA := prqUserA
	principalB := prqUserB

	// newest first.
	list, err := kf.store.ListByFingerprint(kf.ctx, "fp-b", nil, nil, nil)
	require.NoError(t, err)
	require.Equal(t, []string{"gamma", "beta"}, kf.identifiers(list))

	list, err = kf.store.ListByFingerprint(kf.ctx, "fp-b", &principalB, nil, nil)
	require.NoError(t, err)
	require.Empty(t, list)

	list, err = kf.store.ListByFingerprint(kf.ctx, "fp-b", &principalA,
		[]enum.PublicKeyUsage{enum.PublicKeyUsageSign}, []enum.PublicKeyScheme{enum.PublicKeySchemePGP})
	require.NoError(t, err)
	require.Equal(t, []string{"beta"}, kf.identifiers(list))

	list, err = kf.store.ListByFingerprint(kf.ctx, "fp-b", nil,
		[]enum.PublicKeyUsage{enum.PublicKeyUsageSign, enum.PublicKeyUsageAuthSign}, nil)
	require.NoError(t, err)
	require.Equal(t, []string{"gamma", "beta"}, kf.identifiers(list))

	list, err = kf.store.ListByFingerprint(kf.ctx, "fp-unknown", nil, nil, nil)
	require.NoError(t, err)
	require.Empty(t, list)

	// sub keys.
	require.NoError(t, kf.subKeys.Create(kf.ctx, kf.keysByKey["beta"].ID, []string{"subkey-1", "subkey-2"}))
	require.NoError(t, kf.subKeys.Create(kf.ctx, kf.keysByKey["delta"].ID, []string{"subkey-2"}))

	list, err = kf.store.ListBySubKeyID(kf.ctx, "subkey-2", nil, nil, nil)
	require.NoError(t, err)
	require.Equal(t, []string{"delta", "beta"}, kf.identifiers(list))

	list, err = kf.store.ListBySubKeyID(kf.ctx, "subkey-2", &principalA, nil,
		[]enum.PublicKeyScheme{enum.PublicKeySchemePGP})
	require.NoError(t, err)
	require.Equal(t, []string{"beta"}, kf.identifiers(list))

	list, err = kf.store.ListBySubKeyID(kf.ctx, "subkey-1", nil,
		[]enum.PublicKeyUsage{enum.PublicKeyUsageAuth}, nil)
	require.NoError(t, err)
	require.Empty(t, list)
}

func TestPrqPublicKeySubKeyStore_CreateAndList(t *testing.T) {
	kf := prqNewKeyFixture(t)

	// nothing to create.
	require.NoError(t, kf.subKeys.Create(kf.ctx, kf.keysByKey["alpha"].ID, nil))

	ids, err := kf.subKeys.List(kf.ctx, kf.keysByKey["alpha"].ID)
	require.NoError(t, err)
	require.Empty(t, ids)

	subKeys := []string{"sub-a", "sub-b", "sub-c"}
	require.NoError(t, kf.subKeys.Create(kf.ctx, kf.keysByKey["alpha"].ID, subKeys))
	require.NoError(t, kf.subKeys.Create(kf.ctx, kf.keysByKey["beta"].ID, []string{"sub-a"}))

	ids, err = kf.subKeys.List(kf.ctx, kf.keysByKey["alpha"].ID)
	require.NoError(t, err)
	require.ElementsMatch(t, subKeys, ids)

	ids, err = kf.subKeys.List(kf.ctx, kf.keysByKey["beta"].ID)
	require.NoError(t, err)
	require.Equal(t, []string{"sub-a"}, ids)

	err = kf.subKeys.Create(kf.ctx, kf.keysByKey["alpha"].ID, []string{"sub-a"})
	require.ErrorIs(t, err, gitness_store.ErrDuplicate)

	err = kf.subKeys.Create(kf.ctx, 9999, []string{"sub-z"})
	require.ErrorIs(t, err, gitness_store.ErrForeignKeyViolation)

	// removing the key removes its sub keys.
	require.NoError(t, kf.store.DeleteByIdentifier(kf.ctx, prqUserA, "alpha"))
	ids, err = kf.subKeys.List(kf.ctx, kf.keysByKey["alpha"].ID)
	require.NoError(t, err)
	require.Empty(t, ids)
}

// TestPrqStores_CanceledContext verifies that database failures are reported by the stores
// instead of being swallowed.
func TestPrqStores_CanceledContext(t *testing.T) {
	f := prqNewFixture(t)

	ctx, cancel := context.WithCancel(f.ctx)
	cancel()

	files := database.NewPullReqFileViewStore(f.db)
	labels := database.NewPullReqLabelStore(f.db)
	labelSuggestions := database.NewPullReqLabelSuggestionStore(f.db)
	reviewerSuggestions := database.NewPullReqReviewerSuggestionStore(f.db)
	reviewers := database.NewPullReqReviewerStore(f.db, f.pCache)
	reviews := database.NewPullReqReviewStore(f.db)
	groups := database.NewPullReqFileGroupStore(f.db)
	usergroupReviewers := database.NewUsergroupReviewerStore(f.db, f.pCache, database.NewUserGroupStore(f.db))
	entries := database.NewMergeQueueEntryStore(f.db)
	queues := database.NewMergeQueueStore(f.db)
	autoMerges := database.NewAutoMergeStore(f.db)
	branches := database.NewBranchStore(f.db)
	activities := database.NewPullReqActivityStore(f.db, f.pCache)
	codeComments := database.NewCodeCommentView(f.db)
	signatures := database.NewGitSignatureResultStore(f.db)
	repoActivities := database.NewRepoActivityStore(f.db)
	repoLangs := database.NewRepoLangStore(f.db)
	publicKeys := database.NewPublicKeyStore(f.db)
	subKeys := database.NewPublicKeySubKeyStore(f.db)

	filter := &types.PublicKeyFilter{}
	calls := map[string]func() error{
		"file view upsert":   func() error { return files.Upsert(ctx, &types.PullReqFileView{PullReqID: 1}) },
		"file view delete":   func() error { return files.DeleteByFileForPrincipal(ctx, 1, 1, "a") },
		"file view obsolete": func() error { return files.MarkObsolete(ctx, 1, []string{"a"}) },
		"file view list": func() error {
			_, err := files.List(ctx, 1, 1)
			return err
		},
		"label assign":   func() error { return labels.Assign(ctx, &types.PullReqLabel{PullReqID: 1, LabelID: 1}) },
		"label unassign": func() error { _, err := labels.Unassign(ctx, 1, 1); return err },
		"label assigned": func() error { _, err := labels.ListAssigned(ctx, 1); return err },
		"label assigned by pullreqs": func() error {
			_, err := labels.ListAssignedByPullreqIDs(ctx, []int64{1})
			return err
		},
		"label value": func() error { _, err := labels.FindValueByLabelID(ctx, 1, 1); return err },
		"label suggestion create": func() error {
			return labelSuggestions.CreateMany(ctx, []*types.PullReqLabelSuggestion{{PullReqID: 1, LabelID: 1}})
		},
		"label suggestion list": func() error {
			_, err := labelSuggestions.List(ctx, 1, types.ListQueryFilter{})
			return err
		},
		"label suggestion count":  func() error { _, err := labelSuggestions.Count(ctx, 1); return err },
		"label suggestion find":   func() error { _, err := labelSuggestions.Find(ctx, 1, 1); return err },
		"label suggestion delete": func() error { return labelSuggestions.Delete(ctx, 1, 1) },
		"reviewer suggestion create": func() error {
			return reviewerSuggestions.CreateMany(ctx, []*types.PullReqReviewerSuggestion{{PullReqID: 1}})
		},
		"reviewer suggestion list": func() error {
			_, err := reviewerSuggestions.List(ctx, 1, types.Pagination{})
			return err
		},
		"reviewer suggestion count":  func() error { _, err := reviewerSuggestions.Count(ctx, 1); return err },
		"reviewer suggestion find":   func() error { _, err := reviewerSuggestions.Find(ctx, 1, 1); return err },
		"reviewer suggestion delete": func() error { return reviewerSuggestions.Delete(ctx, 1, 1) },
		"reviewer create":            func() error { return reviewers.Create(ctx, &types.PullReqReviewer{PullReqID: 1}) },
		"reviewer update":            func() error { return reviewers.Update(ctx, &types.PullReqReviewer{PullReqID: 1}) },
		"reviewer delete":            func() error { return reviewers.Delete(ctx, 1, 1) },
		"reviewer list":              func() error { _, err := reviewers.List(ctx, 1); return err },
		"reviewer count":             func() error { _, err := reviewers.Count(ctx, 1); return err },
		"review create":              func() error { return reviews.Create(ctx, &types.PullReqReview{PullReqID: 1}) },
		"review find":                func() error { _, err := reviews.Find(ctx, 1); return err },
		"file group delete":          func() error { return groups.DeleteByPrID(ctx, 1) },
		"file group list":            func() error { _, err := groups.List(ctx, 1); return err },
		"file group create": func() error {
			group := &types.PullReqFileGroupWithFiles{PullReqFileGroup: types.PullReqFileGroup{Title: "t"}}
			return groups.CreateMany(ctx, []*types.PullReqFileGroupWithFiles{group})
		},
		"usergroup reviewer create": func() error {
			return usergroupReviewers.Create(ctx, &types.UserGroupReviewer{PullReqID: 1})
		},
		"usergroup reviewer delete": func() error { return usergroupReviewers.Delete(ctx, 1, 1) },
		"usergroup reviewer list":   func() error { _, err := usergroupReviewers.List(ctx, 1); return err },
		"usergroup reviewer find":   func() error { _, err := usergroupReviewers.Find(ctx, 1, 1); return err },
		"usergroup reviewer count":  func() error { _, err := usergroupReviewers.Count(ctx, 1); return err },
		"queue entry delete all":    func() error { return entries.DeleteAllForMergeQueue(ctx, 1) },
		"queue entry delete":        func() error { return entries.Delete(ctx, 1) },
		"queue entry list":          func() error { _, err := entries.ListForMergeQueue(ctx, 1); return err },
		"queue entry count":         func() error { _, err := entries.CountForRepoAndBranch(ctx, 1, "main"); return err },
		"queue entry overdue":       func() error { _, err := entries.ListOverdueChecks(ctx, 1); return err },
		"queue entry create":        func() error { return entries.Create(ctx, &types.MergeQueueEntry{PullReqID: 1}) },
		"queue entry update":        func() error { return entries.Update(ctx, &types.MergeQueueEntry{PullReqID: 1}) },
		"queue entry branches": func() error {
			_, err := entries.BranchesWithPullReqInQueue(ctx, 1, []string{"a", "b"})
			return err
		},
		"queue delete":       func() error { return queues.Delete(ctx, 1) },
		"queue create":       func() error { return queues.Create(ctx, &types.MergeQueue{RepoID: 1}) },
		"queue update":       func() error { return queues.Update(ctx, &types.MergeQueue{ID: 1}) },
		"auto merge upsert":  func() error { return autoMerges.Upsert(ctx, &types.AutoMerge{PullReqID: 1}) },
		"auto merge delete":  func() error { _, err := autoMerges.Delete(ctx, 1); return err },
		"branch delete":      func() error { return branches.Delete(ctx, 1, "a") },
		"branch last pr":     func() error { return branches.UpdateLastPR(ctx, 1, "a", nil) },
		"branch upsert":      func() error { return branches.Upsert(ctx, 1, prqNewBranch("a", prqSHA1, 1, 1)) },
		"branch find":        func() error { _, err := branches.Find(ctx, 1, "a"); return err },
		"activity list":      func() error { _, err := activities.List(ctx, 1, &types.PullReqActivityFilter{}); return err },
		"activity count":     func() error { _, err := activities.Count(ctx, 1, &types.PullReqActivityFilter{}); return err },
		"activity authors":   func() error { _, err := activities.ListAuthorIDs(ctx, 1, 1); return err },
		"activity unresolve": func() error { _, err := activities.CountUnresolved(ctx, 1); return err },
		"activity update": func() error {
			return activities.Update(ctx, prqNewActivity(&types.PullReq{ID: 1}, 1, 0))
		},
		"code comment list": func() error { _, err := codeComments.ListNotAtSourceSHA(ctx, 1, "x"); return err },
		"code comment update": func() error {
			return codeComments.UpdateAll(ctx, []*types.CodeComment{{ID: 1}})
		},
		"signature create": func() error { return signatures.Create(ctx, prqNewSignatureResult(1, prqSHA1, 1)) },
		"signature map": func() error {
			_, err := signatures.Map(ctx, 1, []sha.SHA{sha.Must(prqSHA1)})
			return err
		},
		"signature update": func() error { return signatures.UpdateAll(ctx, enum.GitSignatureBad, 1, nil, nil) },
		"signature delete ids": func() error {
			_, err := signatures.DeleteByKeyIDs(ctx, 1, []string{"a"})
			return err
		},
		"signature delete fingerprints": func() error {
			_, err := signatures.DeleteByKeyFingerprints(ctx, 1, []string{"a"})
			return err
		},
		"repo activity list": func() error {
			_, err := repoActivities.List(ctx, 1, &types.RepoActivityFilter{Size: 1})
			return err
		},
		"repo activity count": func() error {
			_, err := repoActivities.Count(ctx, 1, &types.RepoActivityFilter{})
			return err
		},
		"repo lang insert": func() error {
			return repoLangs.InsertByRepoID(ctx, 1, []*types.RepoLangStat{{Language: "Go"}})
		},
		"repo lang delete":     func() error { return repoLangs.DeleteByRepoID(ctx, 1) },
		"repo lang list":       func() error { _, err := repoLangs.ListByRepoID(ctx, 1); return err },
		"public key find":      func() error { _, err := publicKeys.Find(ctx, 1); return err },
		"public key by ident":  func() error { _, err := publicKeys.FindByIdentifier(ctx, 1, "a"); return err },
		"public key create":    func() error { return publicKeys.Create(ctx, prqNewPublicKey(1, "a", "f", 1)) },
		"public key update":    func() error { return publicKeys.Update(ctx, prqNewPublicKey(1, "a", "f", 1)) },
		"public key delete":    func() error { return publicKeys.DeleteByIdentifier(ctx, 1, "a") },
		"public key verified":  func() error { return publicKeys.MarkAsVerified(ctx, 1, 1) },
		"public key count":     func() error { _, err := publicKeys.Count(ctx, nil, filter); return err },
		"public key list":      func() error { _, err := publicKeys.List(ctx, nil, filter); return err },
		"public key by finger": func() error { _, err := publicKeys.ListByFingerprint(ctx, "f", nil, nil, nil); return err },
		"public key by sub":    func() error { _, err := publicKeys.ListBySubKeyID(ctx, "s", nil, nil, nil); return err },
		"sub key create":       func() error { return subKeys.Create(ctx, 1, []string{"a"}) },
		"sub key list":         func() error { _, err := subKeys.List(ctx, 1); return err },
		"activity create": func() error {
			return activities.Create(ctx, prqNewActivity(&types.PullReq{ID: 1}, 1, 0))
		},
	}

	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			require.Error(t, call())
		})
	}
}
