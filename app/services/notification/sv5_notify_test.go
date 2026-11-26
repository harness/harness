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

package notification

import (
	"context"
	"errors"
	"testing"

	pullreqevents "github.com/harness/gitness/app/events/pullreq"
	"github.com/harness/gitness/app/services/notification/mailer"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

var errSv5Boom = errors.New("boom")

type sv5Client struct {
	Client

	branchRecipients []*types.PrincipalInfo
	branchPayload    *PullReqBranchUpdatedPayload
	branchErr        error
	reviewPayload    *ReviewSubmittedPayload
	reviewRecipients []*types.PrincipalInfo
	reviewErr        error
	statePayloads    []*PullReqStateChangedPayload
	stateErr         error
}

func (c *sv5Client) SendPullReqBranchUpdated(
	_ context.Context, r []*types.PrincipalInfo, p *PullReqBranchUpdatedPayload,
) error {
	c.branchRecipients, c.branchPayload = r, p
	return c.branchErr
}

func (c *sv5Client) SendReviewSubmitted(
	_ context.Context, r []*types.PrincipalInfo, p *ReviewSubmittedPayload,
) error {
	c.reviewRecipients, c.reviewPayload = r, p
	return c.reviewErr
}

func (c *sv5Client) SendPullReqStateChanged(
	_ context.Context, _ []*types.PrincipalInfo, p *PullReqStateChangedPayload,
) error {
	c.statePayloads = append(c.statePayloads, p)
	return c.stateErr
}

type sv5RepoStore struct {
	store.RepoStore

	err error
}

func (s sv5RepoStore) Find(context.Context, int64) (*types.Repository, error) {
	return &types.Repository{ID: 1, Identifier: "repo", Path: "space/repo"}, s.err
}

type sv5PullReqStore struct{ store.PullReqStore }

func (sv5PullReqStore) Find(context.Context, int64) (*types.PullReq, error) {
	return &types.PullReq{ID: 20, Number: 7, Title: "Title", CreatedBy: 1}, nil
}

type sv5PrincipalCache struct {
	store.PrincipalInfoCache

	failID int64
}

func (c sv5PrincipalCache) Get(_ context.Context, id int64) (*types.PrincipalInfo, error) {
	if id == c.failID {
		return nil, errSv5Boom
	}
	return &types.PrincipalInfo{ID: id, Type: enum.PrincipalTypeUser, Email: "u@example.com"}, nil
}

type sv5ReviewerStore struct {
	store.PullReqReviewerStore

	reviewers []*types.PullReqReviewer
	err       error
}

func (s sv5ReviewerStore) List(context.Context, int64) ([]*types.PullReqReviewer, error) {
	return s.reviewers, s.err
}

type sv5GroupReviewerStore struct {
	store.UserGroupReviewerStore

	err error
}

func (s sv5GroupReviewerStore) List(context.Context, int64) ([]*types.UserGroupReviewer, error) {
	return nil, s.err
}

type sv5WatcherStore struct {
	store.PullReqWatcherStore

	watchers []*types.PullReqWatcher
	err      error
}

func (s sv5WatcherStore) List(context.Context, int64) ([]*types.PullReqWatcher, error) {
	return s.watchers, s.err
}

type sv5URLProvider struct{ url.Provider }

func (sv5URLProvider) GenerateUIPRURL(_ context.Context, repoPath string, _ int64) string {
	return "https://ui/" + repoPath
}

type sv5Fixture struct {
	client    *sv5Client
	repoErr   error
	failID    int64
	reviewers []*types.PullReqReviewer
	revErr    error
	groupErr  error
	watchers  []*types.PullReqWatcher
	watchErr  error
}

func (f *sv5Fixture) service() *Service {
	f.client = &sv5Client{}
	return &Service{
		notificationClient:     f.client,
		repoStore:              sv5RepoStore{err: f.repoErr},
		pullReqStore:           sv5PullReqStore{},
		principalInfoCache:     sv5PrincipalCache{failID: f.failID},
		pullReqReviewersStore:  sv5ReviewerStore{reviewers: f.reviewers, err: f.revErr},
		userGroupReviewerStore: sv5GroupReviewerStore{err: f.groupErr},
		pullReqWatcherStore:    sv5WatcherStore{watchers: f.watchers, err: f.watchErr},
		urlProvider:            sv5URLProvider{},
	}
}

func sv5Base() pullreqevents.Base {
	return pullreqevents.Base{PullReqID: 20, TargetRepoID: 1, PrincipalID: 3, Number: 7}
}

func sv5Watcher(id int64) *types.PullReqWatcher {
	return &types.PullReqWatcher{
		PrincipalID: id,
		Watcher:     types.PrincipalInfo{ID: id, Type: enum.PrincipalTypeUser, Email: "w@example.com"},
	}
}

func TestSv5NotifyBranchUpdated(t *testing.T) {
	ctx := context.Background()
	ev := &events.Event[*pullreqevents.BranchUpdatedPayload]{
		Payload: &pullreqevents.BranchUpdatedPayload{Base: sv5Base(), NewSHA: "abc"},
	}

	t.Run("no recipients sends nothing", func(t *testing.T) {
		f := &sv5Fixture{}
		require.NoError(t, f.service().notifyPullReqBranchUpdated(ctx, ev))
		require.Nil(t, f.client.branchPayload)
	})

	t.Run("sends to reviewers and watchers", func(t *testing.T) {
		f := &sv5Fixture{
			reviewers: []*types.PullReqReviewer{{PrincipalID: 4}},
			watchers:  []*types.PullReqWatcher{sv5Watcher(5), sv5Watcher(3), sv5Watcher(4)},
		}
		require.NoError(t, f.service().notifyPullReqBranchUpdated(ctx, ev))
		require.NotNil(t, f.client.branchPayload)
		require.Equal(t, "abc", f.client.branchPayload.NewSHA)
		require.Equal(t, int64(3), f.client.branchPayload.Committer.ID)
		// reviewer 4 and watcher 5; committer 3 and duplicate reviewer 4 are skipped from watchers.
		require.Len(t, f.client.branchRecipients, 2)
		require.Equal(t, int64(4), f.client.branchRecipients[0].ID)
		require.Equal(t, int64(5), f.client.branchRecipients[1].ID)
	})

	errCases := map[string]*sv5Fixture{
		"base":      {repoErr: errSv5Boom},
		"committer": {failID: 3},
		"reviewers": {revErr: errSv5Boom},
		"reviewer principal": {
			reviewers: []*types.PullReqReviewer{{PrincipalID: 9}}, failID: 9,
		},
		"watchers": {watchErr: errSv5Boom},
	}
	for name, f := range errCases {
		t.Run("process error "+name, func(t *testing.T) {
			require.ErrorIs(t, f.service().notifyPullReqBranchUpdated(ctx, ev), errSv5Boom)
		})
	}

	t.Run("send error", func(t *testing.T) {
		f := &sv5Fixture{reviewers: []*types.PullReqReviewer{{PrincipalID: 4}}}
		s := f.service()
		f.client.branchErr = errSv5Boom
		s.notificationClient = f.client
		require.ErrorIs(t, s.notifyPullReqBranchUpdated(ctx, ev), errSv5Boom)
	})
}

func TestSv5NotifyReviewSubmitted(t *testing.T) {
	ctx := context.Background()
	ev := &events.Event[*pullreqevents.ReviewSubmittedPayload]{
		Payload: &pullreqevents.ReviewSubmittedPayload{
			Base: sv5Base(), ReviewerID: 8, Decision: enum.PullReqReviewDecisionApproved,
		},
	}

	f := &sv5Fixture{watchers: []*types.PullReqWatcher{sv5Watcher(6), sv5Watcher(8)}}
	require.NoError(t, f.service().notifyReviewSubmitted(ctx, ev))
	require.Equal(t, enum.PullReqReviewDecisionApproved, f.client.reviewPayload.Decision)
	require.Equal(t, int64(8), f.client.reviewPayload.Reviewer.ID)
	require.Equal(t, int64(1), f.client.reviewPayload.Author.ID)
	// Author first, then the non-skipped watcher.
	require.Len(t, f.client.reviewRecipients, 2)
	require.Equal(t, int64(6), f.client.reviewRecipients[1].ID)

	errCases := map[string]*sv5Fixture{
		"base":     {repoErr: errSv5Boom},
		"author":   {failID: 1},
		"reviewer": {failID: 8},
		"watchers": {watchErr: errSv5Boom},
	}
	for name, ef := range errCases {
		t.Run("error "+name, func(t *testing.T) {
			require.Error(t, ef.service().notifyReviewSubmitted(ctx, ev))
		})
	}

	t.Run("send error", func(t *testing.T) {
		s := (&sv5Fixture{}).service()
		s.notificationClient = &sv5Client{reviewErr: errSv5Boom}
		require.ErrorIs(t, s.notifyReviewSubmitted(ctx, ev), errSv5Boom)
	})
}

func TestSv5NotifyPullReqState(t *testing.T) {
	ctx := context.Background()
	base := sv5Base()

	merged := &events.Event[*pullreqevents.MergedPayload]{Payload: &pullreqevents.MergedPayload{Base: base}}
	closed := &events.Event[*pullreqevents.ClosedPayload]{Payload: &pullreqevents.ClosedPayload{Base: base}}
	reopened := &events.Event[*pullreqevents.ReopenedPayload]{Payload: &pullreqevents.ReopenedPayload{Base: base}}

	f := &sv5Fixture{}
	s := f.service()
	require.NoError(t, s.notifyPullReqStateMerged(ctx, merged))
	require.NoError(t, s.notifyPullReqStateClosed(ctx, closed))
	require.NoError(t, s.notifyPullReqStateReOpened(ctx, reopened))
	require.Len(t, f.client.statePayloads, 3)
	require.Equal(t, PullReqStateMerged, f.client.statePayloads[0].State)
	require.Equal(t, PullReqStateClosed, f.client.statePayloads[1].State)
	require.Equal(t, PullReqStateReopened, f.client.statePayloads[2].State)
	require.Equal(t, int64(3), f.client.statePayloads[0].ChangedBy.ID)

	// Process failures.
	bad := (&sv5Fixture{repoErr: errSv5Boom}).service()
	require.Error(t, bad.notifyPullReqStateMerged(ctx, merged))
	require.Error(t, bad.notifyPullReqStateClosed(ctx, closed))
	require.Error(t, bad.notifyPullReqStateReOpened(ctx, reopened))

	// Send failures.
	send := (&sv5Fixture{}).service()
	send.notificationClient = &sv5Client{stateErr: errSv5Boom}
	require.ErrorIs(t, send.notifyPullReqStateMerged(ctx, merged), errSv5Boom)
	require.ErrorIs(t, send.notifyPullReqStateClosed(ctx, closed), errSv5Boom)
	require.ErrorIs(t, send.notifyPullReqStateReOpened(ctx, reopened), errSv5Boom)
}

type sv5Mailer struct {
	sent []mailer.Payload
	err  error
}

func (m *sv5Mailer) Send(_ context.Context, p mailer.Payload) error {
	m.sent = append(m.sent, p)
	return m.err
}

func sv5BasePayload() *BasePullReqPayload {
	return &BasePullReqPayload{
		Repo:       &types.Repository{ID: 1, Identifier: "repo", Path: "space/repo"},
		PullReq:    &types.PullReq{ID: 10, Number: 42, Title: "My PR"},
		Author:     &types.PrincipalInfo{ID: 1, DisplayName: "Author", Email: "author@example.com"},
		PullReqURL: "https://example.com/pr/42",
	}
}

func TestSv5MailClientSenders(t *testing.T) {
	ctx := context.Background()
	base := sv5BasePayload()
	user := &types.PrincipalInfo{ID: 2, DisplayName: "Alice", Email: "alice@example.com"}
	recipients := []*types.PrincipalInfo{user}

	senders := map[string]func(c MailClient) error{
		"comment author": func(c MailClient) error {
			return c.SendCommentPRAuthor(ctx, recipients, &CommentPayload{Base: base, Commenter: user, Text: "hi"})
		},
		"comment mentions": func(c MailClient) error {
			return c.SendCommentMentions(ctx, recipients, &CommentPayload{Base: base, Commenter: user, Text: "hi"})
		},
		"comment participants": func(c MailClient) error {
			return c.SendCommentParticipants(ctx, recipients, &CommentPayload{Base: base, Commenter: user, Text: "hi"})
		},
		"reviewer added": func(c MailClient) error {
			return c.SendReviewerAdded(ctx, recipients, &ReviewerAddedPayload{Base: base, Reviewer: user})
		},
		"user group reviewer added": func(c MailClient) error {
			return c.SendUserGroupReviewerAdded(ctx, recipients,
				&ReviewersAddedPayload{Base: base, ReviewerCount: 1, ReviewerNames: "Alice"})
		},
		"branch updated": func(c MailClient) error {
			return c.SendPullReqBranchUpdated(ctx, recipients,
				&PullReqBranchUpdatedPayload{Base: base, Committer: user, NewSHA: "abc"})
		},
		"review submitted": func(c MailClient) error {
			return c.SendReviewSubmitted(ctx, recipients, &ReviewSubmittedPayload{
				Base: base, Author: base.Author, Reviewer: user, Decision: enum.PullReqReviewDecisionApproved,
			})
		},
		"state changed": func(c MailClient) error {
			return c.SendPullReqStateChanged(ctx, recipients,
				&PullReqStateChangedPayload{Base: base, ChangedBy: user, State: PullReqStateMerged})
		},
	}

	for name, send := range senders {
		t.Run(name, func(t *testing.T) {
			m := &sv5Mailer{}
			require.NoError(t, send(NewMailClient(m)))
			require.Len(t, m.sent, 1)
			require.Equal(t, []string{"alice@example.com"}, m.sent[0].ToRecipients)
			require.Equal(t, "[repo] My PR (PR #42)", m.sent[0].Subject)
			require.Equal(t, "space/repo", m.sent[0].RepoRef)
			require.NotEmpty(t, m.sent[0].Body)

			m = &sv5Mailer{err: errSv5Boom}
			require.ErrorIs(t, send(NewMailClient(m)), errSv5Boom)
		})
	}
}

func TestSv5GetHTMLBodyErrors(t *testing.T) {
	// A payload that doesn't match the template fields makes execution fail.
	_, err := GetHTMLBody(TemplateCommentPRAuthor, struct{}{})
	require.ErrorContains(t, err, "failed to execute template")
}
