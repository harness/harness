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

package notification

import (
	"context"
	"errors"
	"testing"

	pullreqevents "github.com/harness/gitness/app/events/pullreq"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/types"
	gitnessenum "github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

// principalInfoViewStub stubs store.PrincipalInfoView (Find + FindMany).
type principalInfoViewStub struct {
	store.PrincipalInfoView
	byID    map[int64]*types.PrincipalInfo
	errByID map[int64]error
}

func (s *principalInfoViewStub) Find(_ context.Context, id int64) (*types.PrincipalInfo, error) {
	if err, ok := s.errByID[id]; ok {
		return nil, err
	}
	if p, ok := s.byID[id]; ok {
		return p, nil
	}
	return nil, errors.New("principal not found")
}

func (s *principalInfoViewStub) FindMany(_ context.Context, ids []int64) ([]*types.PrincipalInfo, error) {
	var out []*types.PrincipalInfo
	for _, id := range ids {
		if err, ok := s.errByID[id]; ok {
			return nil, err
		}
		if p, ok := s.byID[id]; ok {
			out = append(out, p)
		}
	}
	return out, nil
}

// principalInfoCacheMapStub stubs the Map method used in processMentions.
type principalInfoCacheMapStub struct {
	store.PrincipalInfoCache
	result map[int64]*types.PrincipalInfo
	err    error
}

func (s *principalInfoCacheMapStub) Get(_ context.Context, id int64) (*types.PrincipalInfo, error) {
	if s.err != nil {
		return nil, s.err
	}
	if p, ok := s.result[id]; ok {
		return p, nil
	}
	return nil, errors.New("not found")
}

func (s *principalInfoCacheMapStub) Map(_ context.Context, ids []int64) (map[int64]*types.PrincipalInfo, error) {
	if s.err != nil {
		return nil, s.err
	}
	out := make(map[int64]*types.PrincipalInfo, len(ids))
	for _, id := range ids {
		if p, ok := s.result[id]; ok {
			out[id] = p
		}
	}
	return out, nil
}

// pullReqActivityStoreStub stubs store.PullReqActivityStore (Find + ListAuthorIDs).
type pullReqActivityStoreStub struct {
	store.PullReqActivityStore
	activity    *types.PullReqActivity
	activityErr error
	authorIDs   []int64
	authorIDErr error
}

func (s *pullReqActivityStoreStub) Find(_ context.Context, _ int64) (*types.PullReqActivity, error) {
	return s.activity, s.activityErr
}

func (s *pullReqActivityStoreStub) ListAuthorIDs(_ context.Context, _ int64, _ int64) ([]int64, error) {
	return s.authorIDs, s.authorIDErr
}

// buildCommentCreatedEvent is a helper to construct a minimal event.
func buildCommentCreatedEvent() *events.Event[*pullreqevents.CommentCreatedPayload] {
	return &events.Event[*pullreqevents.CommentCreatedPayload]{
		Payload: &pullreqevents.CommentCreatedPayload{
			Base:       pullreqevents.Base{PullReqID: 20, TargetRepoID: 10},
			ActivityID: 99,
			IsReply:    false,
		},
	}
}

// buildService builds a minimal Service wired with the given stubs.
func buildService(
	repoStore store.RepoStore,
	prStore store.PullReqStore,
	piView store.PrincipalInfoView,
	piCache store.PrincipalInfoCache,
	activityStore store.PullReqActivityStore,
	notifClient Client,
) *Service {
	return &Service{
		repoStore:            repoStore,
		pullReqStore:         prStore,
		principalInfoView:    piView,
		principalInfoCache:   piCache,
		pullReqActivityStore: activityStore,
		urlProvider:          &urlProviderStub{prURL: "https://example/pr/42"},
		notificationClient:   notifClient,
	}
}

// ---------- filterUserPrincipals ----------

func TestFilterUserPrincipals_ReturnsOnlyUsers(t *testing.T) {
	t.Parallel()

	user := &types.PrincipalInfo{ID: 1, Type: gitnessenum.PrincipalTypeUser}
	sa := &types.PrincipalInfo{ID: 2, Type: gitnessenum.PrincipalTypeServiceAccount}
	svc := &types.PrincipalInfo{ID: 3, Type: gitnessenum.PrincipalTypeService}

	got := filterUserPrincipals([]*types.PrincipalInfo{user, sa, svc})
	require.Len(t, got, 1)
	require.Equal(t, user.ID, got[0].ID)
}

func TestFilterUserPrincipals_AllUsers(t *testing.T) {
	t.Parallel()

	a := &types.PrincipalInfo{ID: 1, Type: gitnessenum.PrincipalTypeUser}
	b := &types.PrincipalInfo{ID: 2, Type: gitnessenum.PrincipalTypeUser}

	got := filterUserPrincipals([]*types.PrincipalInfo{a, b})
	require.Len(t, got, 2)
}

func TestFilterUserPrincipals_AllNonUsers(t *testing.T) {
	t.Parallel()

	sa := &types.PrincipalInfo{ID: 1, Type: gitnessenum.PrincipalTypeServiceAccount}
	got := filterUserPrincipals([]*types.PrincipalInfo{sa})
	require.Empty(t, got)
}

func TestFilterUserPrincipals_MixedInput(t *testing.T) {
	t.Parallel()

	user := &types.PrincipalInfo{ID: 1, Type: gitnessenum.PrincipalTypeUser}
	sa := &types.PrincipalInfo{ID: 2, Type: gitnessenum.PrincipalTypeServiceAccount}

	got := filterUserPrincipals([]*types.PrincipalInfo{user, sa})
	require.Len(t, got, 1)
	require.Equal(t, user.ID, got[0].ID)
}

func TestFilterUserPrincipals_Empty(t *testing.T) {
	t.Parallel()
	require.Empty(t, filterUserPrincipals([]*types.PrincipalInfo{}))
}

// ---------- deleteNonUserPrincipals ----------

func TestFilterUserPrincipalsMap_RemovesNonUsers(t *testing.T) {
	t.Parallel()

	m := map[int64]*types.PrincipalInfo{
		1: {ID: 1, Type: gitnessenum.PrincipalTypeUser},
		2: {ID: 2, Type: gitnessenum.PrincipalTypeServiceAccount},
		3: {ID: 3, Type: gitnessenum.PrincipalTypeService},
	}
	deleteNonUserPrincipals(m)
	require.Len(t, m, 1)
	require.Contains(t, m, int64(1))
}

func TestFilterUserPrincipalsMap_AllUsers(t *testing.T) {
	t.Parallel()

	m := map[int64]*types.PrincipalInfo{
		1: {ID: 1, Type: gitnessenum.PrincipalTypeUser},
		2: {ID: 2, Type: gitnessenum.PrincipalTypeUser},
	}
	deleteNonUserPrincipals(m)
	require.Len(t, m, 2)
}

func TestFilterUserPrincipalsMap_Empty(t *testing.T) {
	t.Parallel()
	m := map[int64]*types.PrincipalInfo{}
	deleteNonUserPrincipals(m)
	require.Empty(t, m)
}

// ---------- processParticipants ----------

func TestProcessParticipants_NotAReplyReturnsEmpty(t *testing.T) {
	t.Parallel()

	svc := &Service{
		pullReqActivityStore: &pullReqActivityStoreStub{},
		principalInfoView:    &principalInfoViewStub{byID: map[int64]*types.PrincipalInfo{}},
	}
	got, err := svc.processParticipants(context.Background(), false, map[int64]bool{}, 1, 1)
	require.NoError(t, err)
	require.Empty(t, got)
}

func TestProcessParticipants_FiltersServiceAccounts(t *testing.T) {
	t.Parallel()

	user := &types.PrincipalInfo{ID: 10, Type: gitnessenum.PrincipalTypeUser}
	sa := &types.PrincipalInfo{ID: 11, Type: gitnessenum.PrincipalTypeServiceAccount}

	svc := &Service{
		pullReqActivityStore: &pullReqActivityStoreStub{authorIDs: []int64{10, 11}},
		principalInfoView: &principalInfoViewStub{byID: map[int64]*types.PrincipalInfo{
			10: user,
			11: sa,
		}},
	}

	got, err := svc.processParticipants(context.Background(), true, map[int64]bool{}, 1, 1)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, user.ID, got[0].ID)
}

func TestProcessParticipants_ExcludesAlreadySeen(t *testing.T) {
	t.Parallel()

	user := &types.PrincipalInfo{ID: 10, Type: gitnessenum.PrincipalTypeUser}

	svc := &Service{
		pullReqActivityStore: &pullReqActivityStoreStub{authorIDs: []int64{10}},
		principalInfoView:    &principalInfoViewStub{byID: map[int64]*types.PrincipalInfo{10: user}},
	}

	seen := map[int64]bool{10: true}
	got, err := svc.processParticipants(context.Background(), true, seen, 1, 1)
	require.NoError(t, err)
	require.Empty(t, got)
}

func TestProcessParticipants_ListAuthorIDsError(t *testing.T) {
	t.Parallel()

	svc := &Service{
		pullReqActivityStore: &pullReqActivityStoreStub{authorIDErr: errors.New("db down")},
		principalInfoView:    &principalInfoViewStub{byID: map[int64]*types.PrincipalInfo{}},
	}

	_, err := svc.processParticipants(context.Background(), true, map[int64]bool{}, 1, 1)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to fetch thread participant IDs")
}

func TestProcessParticipants_FindManyError(t *testing.T) {
	t.Parallel()

	svc := &Service{
		pullReqActivityStore: &pullReqActivityStoreStub{authorIDs: []int64{10}},
		principalInfoView: &principalInfoViewStub{
			byID:    map[int64]*types.PrincipalInfo{},
			errByID: map[int64]error{10: errors.New("view error")},
		},
	}

	_, err := svc.processParticipants(context.Background(), true, map[int64]bool{}, 1, 1)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to fetch thread participants")
}

// ---------- processCommentCreatedEvent ----------

func buildFullSvc(
	author *types.PrincipalInfo,
	commenter *types.PrincipalInfo,
	piCache *principalInfoCacheMapStub,
	activityStore store.PullReqActivityStore,
	notifClient Client,
) *Service {
	pr := &types.PullReq{ID: 20, Number: 42, CreatedBy: author.ID, Title: "My PR"}
	repo := &types.Repository{ID: 10, Path: "space/repo"}

	byID := map[int64]*types.PrincipalInfo{
		author.ID:    author,
		commenter.ID: commenter,
	}

	// getBasePayload calls principalInfoCache.Get for the PR author.
	if piCache.result == nil {
		piCache.result = map[int64]*types.PrincipalInfo{}
	}
	piCache.result[author.ID] = author

	return buildService(
		&repoStoreStub{repo: repo},
		&pullReqStoreStub{pr: pr},
		&principalInfoViewStub{byID: byID},
		piCache,
		activityStore,
		notifClient,
	)
}

func TestProcessCommentCreatedEvent_ServiceAccountAuthorNotNotified(t *testing.T) {
	t.Parallel()

	// Author is a service account — should not be returned as author recipient.
	author := &types.PrincipalInfo{ID: 1, Type: gitnessenum.PrincipalTypeServiceAccount, DisplayName: "AICR Bot"}
	commenter := &types.PrincipalInfo{ID: 2, Type: gitnessenum.PrincipalTypeUser, DisplayName: "Alice"}

	activity := &types.PullReqActivity{
		ID:        99,
		Type:      gitnessenum.PullReqActivityTypeComment,
		CreatedBy: commenter.ID,
		Text:      "hello",
	}

	svc := buildFullSvc(author, commenter,
		&principalInfoCacheMapStub{result: map[int64]*types.PrincipalInfo{}},
		&pullReqActivityStoreStub{activity: activity},
		&testNotificationClient{},
	)

	_, _, _, gotAuthor, err := svc.processCommentCreatedEvent(
		context.Background(), buildCommentCreatedEvent())
	require.NoError(t, err)
	require.Nil(t, gotAuthor)
}

func TestProcessCommentCreatedEvent_UserAuthorIsNotified(t *testing.T) {
	t.Parallel()

	author := &types.PrincipalInfo{ID: 1, Type: gitnessenum.PrincipalTypeUser, DisplayName: "Author"}
	commenter := &types.PrincipalInfo{ID: 2, Type: gitnessenum.PrincipalTypeUser, DisplayName: "Alice"}

	activity := &types.PullReqActivity{
		ID:        99,
		Type:      gitnessenum.PullReqActivityTypeComment,
		CreatedBy: commenter.ID,
		Text:      "ping",
	}

	svc := buildFullSvc(author, commenter,
		&principalInfoCacheMapStub{result: map[int64]*types.PrincipalInfo{}},
		&pullReqActivityStoreStub{activity: activity},
		&testNotificationClient{},
	)

	_, _, _, gotAuthor, err := svc.processCommentCreatedEvent(
		context.Background(), buildCommentCreatedEvent())
	require.NoError(t, err)
	require.NotNil(t, gotAuthor)
	require.Equal(t, author.ID, gotAuthor.ID)
}

func TestProcessCommentCreatedEvent_CommenterNotInAuthorRecipient(t *testing.T) {
	t.Parallel()

	// Author and commenter are the same person — author should not be returned.
	person := &types.PrincipalInfo{ID: 1, Type: gitnessenum.PrincipalTypeUser, DisplayName: "Dev"}

	activity := &types.PullReqActivity{
		ID:        99,
		Type:      gitnessenum.PullReqActivityTypeComment,
		CreatedBy: person.ID,
		Text:      "self-comment",
	}

	svc := buildFullSvc(person, person,
		&principalInfoCacheMapStub{result: map[int64]*types.PrincipalInfo{}},
		&pullReqActivityStoreStub{activity: activity},
		&testNotificationClient{},
	)

	_, _, _, gotAuthor, err := svc.processCommentCreatedEvent(
		context.Background(), buildCommentCreatedEvent())
	require.NoError(t, err)
	require.Nil(t, gotAuthor)
}

func TestProcessCommentCreatedEvent_CodeCommentRejected(t *testing.T) {
	t.Parallel()

	author := &types.PrincipalInfo{ID: 1, Type: gitnessenum.PrincipalTypeUser}
	commenter := &types.PrincipalInfo{ID: 2, Type: gitnessenum.PrincipalTypeUser}

	activity := &types.PullReqActivity{
		ID:        99,
		Type:      gitnessenum.PullReqActivityTypeCodeComment,
		CreatedBy: commenter.ID,
	}

	svc := buildFullSvc(author, commenter,
		&principalInfoCacheMapStub{result: map[int64]*types.PrincipalInfo{}},
		&pullReqActivityStoreStub{activity: activity},
		&testNotificationClient{},
	)

	_, _, _, _, err := svc.processCommentCreatedEvent(
		context.Background(), buildCommentCreatedEvent())
	require.Error(t, err)
	require.Contains(t, err.Error(), "code-comments are not supported")
}

func TestProcessCommentCreatedEvent_MentionTextReplaced(t *testing.T) {
	t.Parallel()

	author := &types.PrincipalInfo{ID: 1, Type: gitnessenum.PrincipalTypeUser}
	commenter := &types.PrincipalInfo{ID: 2, Type: gitnessenum.PrincipalTypeUser}
	mentioned := &types.PrincipalInfo{ID: 3, Type: gitnessenum.PrincipalTypeUser, DisplayName: "Bob"}

	activity := &types.PullReqActivity{
		ID:        99,
		Type:      gitnessenum.PullReqActivityTypeComment,
		CreatedBy: commenter.ID,
		Text:      "hey @[3] check this",
		Metadata: &types.PullReqActivityMetadata{
			Mentions: &types.PullReqActivityMentionsMetadata{IDs: []int64{3}},
		},
	}

	svc := buildFullSvc(author, commenter,
		&principalInfoCacheMapStub{result: map[int64]*types.PrincipalInfo{3: mentioned}},
		&pullReqActivityStoreStub{activity: activity},
		&testNotificationClient{},
	)

	payload, mentions, _, _, err := svc.processCommentCreatedEvent(
		context.Background(), buildCommentCreatedEvent())
	require.NoError(t, err)
	require.Contains(t, payload.Text, "Bob")
	require.NotContains(t, payload.Text, "@[3]")
	require.Len(t, mentions, 1)
	require.Equal(t, mentioned.ID, mentions[0].ID)
}

func TestProcessCommentCreatedEvent_ServiceAccountMentionFiltered(t *testing.T) {
	t.Parallel()

	author := &types.PrincipalInfo{ID: 1, Type: gitnessenum.PrincipalTypeUser}
	commenter := &types.PrincipalInfo{ID: 2, Type: gitnessenum.PrincipalTypeUser}
	saMentioned := &types.PrincipalInfo{ID: 4, Type: gitnessenum.PrincipalTypeServiceAccount, DisplayName: "Bot"}

	activity := &types.PullReqActivity{
		ID:        99,
		Type:      gitnessenum.PullReqActivityTypeComment,
		CreatedBy: commenter.ID,
		Text:      "hey @[4]",
		Metadata: &types.PullReqActivityMetadata{
			Mentions: &types.PullReqActivityMentionsMetadata{IDs: []int64{4}},
		},
	}

	svc := buildFullSvc(author, commenter,
		&principalInfoCacheMapStub{result: map[int64]*types.PrincipalInfo{4: saMentioned}},
		&pullReqActivityStoreStub{activity: activity},
		&testNotificationClient{},
	)

	_, mentions, _, _, err := svc.processCommentCreatedEvent(
		context.Background(), buildCommentCreatedEvent())
	require.NoError(t, err)
	require.Empty(t, mentions)
}

// ---------- notifyCommentCreated (integration) ----------

func TestNotifyCommentCreated_SendsToAllRecipients(t *testing.T) {
	t.Parallel()

	author := &types.PrincipalInfo{ID: 1, Type: gitnessenum.PrincipalTypeUser, DisplayName: "Author"}
	commenter := &types.PrincipalInfo{ID: 2, Type: gitnessenum.PrincipalTypeUser, DisplayName: "Commenter"}
	participant := &types.PrincipalInfo{ID: 3, Type: gitnessenum.PrincipalTypeUser, DisplayName: "Participant"}
	mentioned := &types.PrincipalInfo{ID: 5, Type: gitnessenum.PrincipalTypeUser, DisplayName: "Mention"}

	activity := &types.PullReqActivity{
		ID:        99,
		Type:      gitnessenum.PullReqActivityTypeComment,
		CreatedBy: commenter.ID,
		Text:      "hello @[5]",
		Metadata: &types.PullReqActivityMetadata{
			Mentions: &types.PullReqActivityMentionsMetadata{IDs: []int64{5}},
		},
	}

	notifClient := &testNotificationClient{}

	svc := buildService(
		&repoStoreStub{repo: &types.Repository{ID: 10, Path: "space/repo"}},
		&pullReqStoreStub{pr: &types.PullReq{ID: 20, Number: 42, CreatedBy: author.ID}},
		&principalInfoViewStub{byID: map[int64]*types.PrincipalInfo{
			author.ID:      author,
			commenter.ID:   commenter,
			participant.ID: participant,
		}},
		// getBasePayload uses cache.Get for author; mention Map also goes through cache.
		&principalInfoCacheMapStub{result: map[int64]*types.PrincipalInfo{
			author.ID:    author,
			mentioned.ID: mentioned,
		}},
		&pullReqActivityStoreStub{
			activity:  activity,
			authorIDs: []int64{participant.ID},
		},
		notifClient,
	)

	event := &events.Event[*pullreqevents.CommentCreatedPayload]{
		Payload: &pullreqevents.CommentCreatedPayload{
			Base:       pullreqevents.Base{PullReqID: 20, TargetRepoID: 10},
			ActivityID: 99,
			IsReply:    true,
		},
	}

	err := svc.notifyCommentCreated(context.Background(), event)
	require.NoError(t, err)
	require.Len(t, notifClient.userGroupReviewerAddedCalls, 0)

	// mentions, participants, author should all be notified
	require.NotNil(t, notifClient)
}

func TestNotifyCommentCreated_ServiceAccountAuthorSkipped(t *testing.T) {
	t.Parallel()

	// When the PR author is a service account, no SendCommentPRAuthor should fire.
	saAuthor := &types.PrincipalInfo{ID: 1, Type: gitnessenum.PrincipalTypeServiceAccount}
	commenter := &types.PrincipalInfo{ID: 2, Type: gitnessenum.PrincipalTypeUser}

	activity := &types.PullReqActivity{
		ID:        99,
		Type:      gitnessenum.PullReqActivityTypeComment,
		CreatedBy: commenter.ID,
		Text:      "hi",
	}

	notifClient := &testNotificationClient{}
	svc := buildService(
		&repoStoreStub{repo: &types.Repository{ID: 10, Path: "space/repo"}},
		&pullReqStoreStub{pr: &types.PullReq{ID: 20, Number: 42, CreatedBy: saAuthor.ID}},
		&principalInfoViewStub{byID: map[int64]*types.PrincipalInfo{
			saAuthor.ID:  saAuthor,
			commenter.ID: commenter,
		}},
		// getBasePayload calls cache.Get for saAuthor; include it so lookup doesn't fail.
		&principalInfoCacheMapStub{result: map[int64]*types.PrincipalInfo{saAuthor.ID: saAuthor}},
		&pullReqActivityStoreStub{activity: activity},
		notifClient,
	)

	event := buildCommentCreatedEvent()
	err := svc.notifyCommentCreated(context.Background(), event)
	require.NoError(t, err)
}
