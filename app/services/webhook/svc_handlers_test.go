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

package webhook

import (
	"context"
	"errors"
	"testing"

	checkevents "github.com/harness/gitness/app/events/check"
	gitevents "github.com/harness/gitness/app/events/git"
	mergequeueevents "github.com/harness/gitness/app/events/mergequeue"
	pullreqevents "github.com/harness/gitness/app/events/pullreq"
	gitnesserrors "github.com/harness/gitness/errors"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func svcEvent[T any](payload T) *events.Event[T] {
	return &events.Event[T]{ID: "evt", Payload: payload}
}

// svcGet walks a decoded JSON object following the provided keys.
func svcGet(t *testing.T, m map[string]any, keys ...string) any {
	t.Helper()
	var cur any = m
	for _, k := range keys {
		obj, ok := cur.(map[string]any)
		require.True(t, ok, "expected object at %q", k)
		cur = obj[k]
	}
	return cur
}

func svcPRBase() pullreqevents.Base {
	src := int64(1)
	return pullreqevents.Base{PullReqID: 100, SourceRepoID: &src, TargetRepoID: 1, PrincipalID: 5, Number: 7}
}

func TestSvcGitReferenceHandlers(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		trigger     enum.WebhookTrigger
		invoke      func(s *Service) error
		wantSHA     string
		wantOldSHA  string
		wantRef     string
		wantCommit  bool
		wantCommits bool
	}{
		{
			name:    "branch created",
			trigger: enum.WebhookTriggerBranchCreated,
			invoke: func(s *Service) error {
				return s.handleEventBranchCreated(ctx, svcEvent(&gitevents.BranchCreatedPayload{
					RepoID: 1, PrincipalID: 5, Ref: "refs/heads/dev", SHA: svcSHAOne,
				}))
			},
			wantSHA: svcSHAOne, wantOldSHA: types.NilSHA, wantRef: "refs/heads/dev", wantCommit: true,
		},
		{
			name:    "branch updated",
			trigger: enum.WebhookTriggerBranchUpdated,
			invoke: func(s *Service) error {
				return s.handleEventBranchUpdated(ctx, svcEvent(&gitevents.BranchUpdatedPayload{
					RepoID: 1, PrincipalID: 5, Ref: "refs/heads/dev", OldSHA: svcSHAOne, NewSHA: svcSHATwo, Forced: true,
				}))
			},
			wantSHA: svcSHATwo, wantOldSHA: svcSHAOne, wantRef: "refs/heads/dev", wantCommit: true, wantCommits: true,
		},
		{
			name:    "branch deleted",
			trigger: enum.WebhookTriggerBranchDeleted,
			invoke: func(s *Service) error {
				return s.handleEventBranchDeleted(ctx, svcEvent(&gitevents.BranchDeletedPayload{
					RepoID: 1, PrincipalID: 5, Ref: "refs/heads/dev", SHA: svcSHAOne,
				}))
			},
			wantSHA: types.NilSHA, wantOldSHA: svcSHAOne, wantRef: "refs/heads/dev",
		},
		{
			name:    "tag created",
			trigger: enum.WebhookTriggerTagCreated,
			invoke: func(s *Service) error {
				return s.handleEventTagCreated(ctx, svcEvent(&gitevents.TagCreatedPayload{
					RepoID: 1, PrincipalID: 5, Ref: "refs/tags/v1", SHA: svcSHAOne,
				}))
			},
			wantSHA: svcSHAOne, wantOldSHA: types.NilSHA, wantRef: "refs/tags/v1", wantCommit: true,
		},
		{
			name:    "tag updated",
			trigger: enum.WebhookTriggerTagUpdated,
			invoke: func(s *Service) error {
				return s.handleEventTagUpdated(ctx, svcEvent(&gitevents.TagUpdatedPayload{
					RepoID: 1, PrincipalID: 5, Ref: "refs/tags/v1", OldSHA: svcSHAOne, NewSHA: svcSHATwo,
				}))
			},
			wantSHA: svcSHATwo, wantOldSHA: svcSHAOne, wantRef: "refs/tags/v1", wantCommit: true, wantCommits: true,
		},
		{
			name:    "tag deleted",
			trigger: enum.WebhookTriggerTagDeleted,
			invoke: func(s *Service) error {
				return s.handleEventTagDeleted(ctx, svcEvent(&gitevents.TagDeletedPayload{
					RepoID: 1, PrincipalID: 5, Ref: "refs/tags/v1", SHA: svcSHAOne,
				}))
			},
			wantSHA: types.NilSHA, wantOldSHA: svcSHAOne, wantRef: "refs/tags/v1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newSvcHarness(t)
			h.addRepoHook(tt.trigger)

			require.NoError(t, tt.invoke(h.svc))

			got := h.receiver.last(t)
			assert.Equal(t, string(tt.trigger), got.Trigger)
			assert.Equal(t, string(tt.trigger), svcGet(t, got.Body, "trigger"))
			assert.Equal(t, tt.wantRef, svcGet(t, got.Body, "ref", "name"))
			assert.Equal(t, tt.wantSHA, svcGet(t, got.Body, "sha"))
			assert.Equal(t, tt.wantOldSHA, svcGet(t, got.Body, "old_sha"))
			assert.Equal(t, "root/repo", svcGet(t, got.Body, "repo", "path"))
			assert.Equal(t, "repo", svcGet(t, got.Body, "repo", "uid"))
			assert.Equal(t, "alice", svcGet(t, got.Body, "principal", "uid"))
			if tt.wantCommit {
				assert.NotNil(t, svcGet(t, got.Body, "commit"))
				assert.NotNil(t, svcGet(t, got.Body, "head_commit"))
			} else {
				assert.Nil(t, svcGet(t, got.Body, "commit"))
			}
			if tt.wantCommits {
				commits, ok := svcGet(t, got.Body, "commits").([]any)
				require.True(t, ok)
				assert.Len(t, commits, 2)
				assert.EqualValues(t, 2, svcGet(t, got.Body, "total_commits_count"))
			}
		})
	}
}

func TestSvcGitReferenceHandlers_Errors(t *testing.T) {
	ctx := context.Background()
	created := svcEvent(&gitevents.BranchCreatedPayload{RepoID: 1, PrincipalID: 5, Ref: "refs/heads/dev", SHA: svcSHAOne})

	t.Run("hook without matching trigger is not executed", func(t *testing.T) {
		h := newSvcHarness(t)
		h.addRepoHook(enum.WebhookTriggerTagCreated)
		require.NoError(t, h.svc.handleEventBranchCreated(ctx, created))
		assert.Equal(t, 0, h.receiver.count())
	})
	t.Run("missing commit discards the event", func(t *testing.T) {
		h := newSvcHarness(t)
		h.addRepoHook()
		ev := svcEvent(&gitevents.BranchCreatedPayload{RepoID: 1, PrincipalID: 5, SHA: svcSHATwo + "x"})
		err := h.svc.handleEventBranchCreated(ctx, ev)
		require.Error(t, err)
		assert.True(t, svcIsDiscard(err))
	})
	t.Run("other git errors retry the event", func(t *testing.T) {
		h := newSvcHarness(t)
		h.addRepoHook()
		h.git.getErr = errors.New("git down")
		err := h.svc.handleEventBranchCreated(ctx, created)
		require.Error(t, err)
		assert.False(t, svcIsDiscard(err))
	})
	t.Run("missing repo discards the event", func(t *testing.T) {
		h := newSvcHarness(t)
		ev := svcEvent(&gitevents.BranchDeletedPayload{RepoID: 99, PrincipalID: 5})
		err := h.svc.handleEventBranchDeleted(ctx, ev)
		require.Error(t, err)
		assert.True(t, svcIsDiscard(err))
	})
	t.Run("repo store failure retries the event", func(t *testing.T) {
		h := newSvcHarness(t)
		h.repos.err = errors.New("db down")
		err := h.svc.handleEventBranchDeleted(ctx, svcEvent(&gitevents.BranchDeletedPayload{RepoID: 1, PrincipalID: 5}))
		require.Error(t, err)
		assert.False(t, svcIsDiscard(err))
	})
	t.Run("missing principal discards the event", func(t *testing.T) {
		h := newSvcHarness(t)
		err := h.svc.handleEventBranchDeleted(ctx, svcEvent(&gitevents.BranchDeletedPayload{RepoID: 1, PrincipalID: 99}))
		require.Error(t, err)
		assert.True(t, svcIsDiscard(err))
	})
	t.Run("update without commits fails", func(t *testing.T) {
		h := newSvcHarness(t)
		h.git.listResult = nil
		err := h.svc.handleEventBranchUpdated(ctx, svcEvent(&gitevents.BranchUpdatedPayload{
			RepoID: 1, PrincipalID: 5, OldSHA: svcSHAOne, NewSHA: svcSHATwo,
		}))
		require.Error(t, err)
	})
	t.Run("update with unknown commits discards", func(t *testing.T) {
		h := newSvcHarness(t)
		h.git.listErr = gitnesserrors.NotFound("commits not found")
		err := h.svc.handleEventBranchUpdated(ctx, svcEvent(&gitevents.BranchUpdatedPayload{
			RepoID: 1, PrincipalID: 5, OldSHA: svcSHAOne, NewSHA: svcSHATwo,
		}))
		require.Error(t, err)
		assert.True(t, svcIsDiscard(err))
	})
	t.Run("retriable receiver failure retries the event", func(t *testing.T) {
		h := newSvcHarness(t)
		h.addRepoHook()
		h.receiver.status = 503
		require.Error(t, h.svc.handleEventBranchCreated(ctx, created))
	})
}

func TestSvcPullReqHandlers(t *testing.T) {
	ctx := context.Background()
	base := svcPRBase()
	valueID := int64(4)
	resolvedBy := int64(6)

	tests := []struct {
		name    string
		trigger enum.WebhookTrigger
		invoke  func(s *Service) error
		check   func(t *testing.T, body map[string]any)
	}{
		{
			name: "created", trigger: enum.WebhookTriggerPullReqCreated,
			invoke: func(s *Service) error {
				return s.handleEventPullReqCreated(ctx, svcEvent(&pullreqevents.CreatedPayload{
					Base: base, SourceBranch: "feature", TargetBranch: "main", SourceSHA: svcSHATwo,
				}))
			},
			check: func(t *testing.T, body map[string]any) {
				assert.Equal(t, svcSHATwo, svcGet(t, body, "sha"))
				assert.NotNil(t, svcGet(t, body, "commits"))
			},
		},
		{
			name: "reopened", trigger: enum.WebhookTriggerPullReqReopened,
			invoke: func(s *Service) error {
				return s.handleEventPullReqReopened(ctx, svcEvent(&pullreqevents.ReopenedPayload{
					Base: base, SourceBranch: "feature", SourceSHA: svcSHATwo,
				}))
			},
			check: func(t *testing.T, body map[string]any) {
				assert.Equal(t, "refs/heads/main", svcGet(t, body, "target_ref", "name"))
			},
		},
		{
			name: "branch updated", trigger: enum.WebhookTriggerPullReqBranchUpdated,
			invoke: func(s *Service) error {
				return s.handleEventPullReqBranchUpdated(ctx, svcEvent(&pullreqevents.BranchUpdatedPayload{
					Base: base, OldSHA: svcSHAOne, NewSHA: svcSHATwo, Forced: true,
				}))
			},
			check: func(t *testing.T, body map[string]any) {
				assert.Equal(t, svcSHAOne, svcGet(t, body, "old_sha"))
				assert.Equal(t, true, svcGet(t, body, "forced"))
				assert.EqualValues(t, 2, svcGet(t, body, "total_commits_count"))
			},
		},
		{
			name: "closed", trigger: enum.WebhookTriggerPullReqClosed,
			invoke: func(s *Service) error {
				return s.handleEventPullReqClosed(ctx, svcEvent(&pullreqevents.ClosedPayload{
					Base: base, SourceBranch: "feature", SourceSHA: svcSHAOne,
				}))
			},
			check: func(t *testing.T, body map[string]any) {
				assert.Equal(t, "refs/heads/feature", svcGet(t, body, "ref", "name"))
			},
		},
		{
			name: "merged", trigger: enum.WebhookTriggerPullReqMerged,
			invoke: func(s *Service) error {
				return s.handleEventPullReqMerged(ctx, svcEvent(&pullreqevents.MergedPayload{
					Base: base, MergeMethod: enum.MergeMethodSquash, MergeSHA: svcSHATwo, SourceSHA: svcSHAOne,
				}))
			},
			check: func(t *testing.T, body map[string]any) {
				assert.Equal(t, svcSHATwo, svcGet(t, body, "pull_req", "merge_sha"))
			},
		},
		{
			name: "comment created", trigger: enum.WebhookTriggerPullReqCommentCreated,
			invoke: func(s *Service) error {
				return s.handleEventPullReqComment(ctx, svcEvent(&pullreqevents.CommentCreatedPayload{
					Base: base, ActivityID: 200, SourceSHA: svcSHAOne,
				}))
			},
			check: func(t *testing.T, body map[string]any) {
				assert.Equal(t, "a comment", svcGet(t, body, "comment", "text"))
			},
		},
		{
			name: "comment updated", trigger: enum.WebhookTriggerPullReqCommentUpdated,
			invoke: func(s *Service) error {
				return s.handleEventPullReqCommentUpdated(ctx, svcEvent(&pullreqevents.CommentUpdatedPayload{
					Base: base, ActivityID: 200,
				}))
			},
			check: func(t *testing.T, body map[string]any) {
				assert.Equal(t, "a comment", svcGet(t, body, "comment", "text"))
			},
		},
		{
			name: "comment status updated", trigger: enum.WebhookTriggerPullReqCommentStatusUpdated,
			invoke: func(s *Service) error {
				return s.handleEventPullReqCommentStatusUpdated(ctx, svcEvent(&pullreqevents.CommentStatusUpdatedPayload{
					Base: base, ActivityID: 200, NewStatus: enum.PullReqCommentStatusResolved, NewResolvedBy: &resolvedBy,
				}))
			},
			check: func(t *testing.T, body map[string]any) {
				assert.Equal(t, string(enum.PullReqCommentStatusActive), svcGet(t, body, "status"))
			},
		},
		{
			name: "label assigned with value", trigger: enum.WebhookTriggerPullReqLabelAssigned,
			invoke: func(s *Service) error {
				return s.handleEventPullReqLabelAssigned(ctx, svcEvent(&pullreqevents.LabelAssignedPayload{
					Base: base, LabelID: 3, ValueID: &valueID,
				}))
			},
			check: func(t *testing.T, body map[string]any) {
				assert.Equal(t, "bug", svcGet(t, body, "label", "key"))
				assert.Equal(t, "high", svcGet(t, body, "label", "value"))
			},
		},
		{
			name: "label assigned without value", trigger: enum.WebhookTriggerPullReqLabelAssigned,
			invoke: func(s *Service) error {
				return s.handleEventPullReqLabelAssigned(ctx, svcEvent(&pullreqevents.LabelAssignedPayload{
					Base: base, LabelID: 3,
				}))
			},
			check: func(t *testing.T, body map[string]any) {
				assert.Equal(t, "bug", svcGet(t, body, "label", "key"))
				assert.Nil(t, svcGet(t, body, "label", "value"))
			},
		},
		{
			name: "updated", trigger: enum.WebhookTriggerPullReqUpdated,
			invoke: func(s *Service) error {
				return s.handleEventPullReqUpdated(ctx, svcEvent(&pullreqevents.UpdatedPayload{
					Base: base, TitleChanged: true, TitleOld: "old", TitleNew: "new",
				}))
			},
			check: func(t *testing.T, body map[string]any) {
				assert.Equal(t, true, svcGet(t, body, "title_changed"))
				assert.Equal(t, "new", svcGet(t, body, "title_new"))
			},
		},
		{
			name: "review submitted", trigger: enum.WebhookTriggerPullReqReviewSubmitted,
			invoke: func(s *Service) error {
				return s.handleEventPullReqReviewSubmitted(ctx, svcEvent(&pullreqevents.ReviewSubmittedPayload{
					Base: base, ReviewerID: 6, Decision: enum.PullReqReviewDecisionApproved,
				}))
			},
			check: func(t *testing.T, body map[string]any) {
				assert.Equal(t, "approved", svcGet(t, body, "review_decision"))
				assert.Equal(t, "bob", svcGet(t, body, "reviewer", "uid"))
			},
		},
		{
			name: "reviewer assigned", trigger: enum.WebhookTriggerPullReqReviewerAssigned,
			invoke: func(s *Service) error {
				return s.handleEventPullReqReviewerAssigned(ctx, svcEvent(&pullreqevents.ReviewerAddedPayload{
					Base: base, ReviewerID: 6,
				}))
			},
			check: func(t *testing.T, body map[string]any) {
				assert.Equal(t, "bob", svcGet(t, body, "reviewer", "uid"))
			},
		},
		{
			name: "reviewer unassigned", trigger: enum.WebhookTriggerPullReqReviewerUnassigned,
			invoke: func(s *Service) error {
				return s.handleEventPullReqReviewerUnassigned(ctx, svcEvent(&pullreqevents.ReviewerRemovedPayload{
					Base: base, ReviewerID: 6,
				}))
			},
			check: func(t *testing.T, body map[string]any) {
				assert.Equal(t, "bob", svcGet(t, body, "reviewer", "uid"))
			},
		},
		{
			name: "usergroup reviewer assigned", trigger: enum.WebhookTriggerPullReqReviewerAssigned,
			invoke: func(s *Service) error {
				return s.handleEventPullReqUserGroupReviewerAssigned(ctx,
					svcEvent(&pullreqevents.UserGroupReviewerAddedPayload{
						Base: base, UserGroupReviewerIDs: []int64{8},
					}))
			},
			check: func(t *testing.T, body map[string]any) {
				assert.Equal(t, "devs", svcGet(t, body, "requested_user_group", "identifier"))
			},
		},
		{
			name: "usergroup reviewer unassigned", trigger: enum.WebhookTriggerPullReqReviewerUnassigned,
			invoke: func(s *Service) error {
				return s.handleEventPullReqUserGroupReviewerUnassigned(ctx,
					svcEvent(&pullreqevents.UserGroupReviewerRemovedPayload{Base: base, UserGroupID: 8}))
			},
			check: func(t *testing.T, body map[string]any) {
				assert.Equal(t, "Developers", svcGet(t, body, "requested_user_group", "name"))
			},
		},
		{
			name: "target branch changed", trigger: enum.WebhookTriggerPullReqTargetBranchChanged,
			invoke: func(s *Service) error {
				return s.handleEventPullReqTargetBranchChanged(ctx, svcEvent(&pullreqevents.TargetBranchChangedPayload{
					Base: base, SourceSHA: svcSHAOne, OldTargetBranch: "old", OldMergeBaseSHA: svcSHATwo,
				}))
			},
			check: func(t *testing.T, body map[string]any) {
				assert.Equal(t, "old", svcGet(t, body, "old_target_branch"))
				assert.Equal(t, svcSHATwo, svcGet(t, body, "old_merge_base_sha"))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newSvcHarness(t)
			h.addRepoHook(tt.trigger)

			require.NoError(t, tt.invoke(h.svc))

			got := h.receiver.last(t)
			assert.Equal(t, string(tt.trigger), got.Trigger)
			assert.EqualValues(t, 7, svcGet(t, got.Body, "pull_req", "number"))
			assert.Equal(t, "refs/heads/feature", svcGet(t, got.Body, "ref", "name"))
			assert.Equal(t, "root/repo", svcGet(t, got.Body, "repo", "path"))
			tt.check(t, got.Body)
		})
	}
}

func TestSvcPullReqHandlers_Errors(t *testing.T) {
	ctx := context.Background()
	base := svcPRBase()

	t.Run("missing pull request discards", func(t *testing.T) {
		h := newSvcHarness(t)
		b := base
		b.PullReqID = 999
		err := h.svc.handleEventPullReqUpdated(ctx, svcEvent(&pullreqevents.UpdatedPayload{Base: b}))
		require.Error(t, err)
		assert.True(t, svcIsDiscard(err))
	})
	t.Run("missing target repo", func(t *testing.T) {
		h := newSvcHarness(t)
		h.prs.prs[100].TargetRepoID = 77
		err := h.svc.handleEventPullReqUpdated(ctx, svcEvent(&pullreqevents.UpdatedPayload{Base: base}))
		require.Error(t, err)
		assert.True(t, svcIsDiscard(err))
	})
	t.Run("source repo deleted discards", func(t *testing.T) {
		h := newSvcHarness(t)
		h.prs.prs[100].SourceRepoID = nil
		err := h.svc.handleEventPullReqUpdated(ctx, svcEvent(&pullreqevents.UpdatedPayload{Base: base}))
		require.Error(t, err)
		assert.True(t, svcIsDiscard(err))
	})
	t.Run("fork source repo is used", func(t *testing.T) {
		h := newSvcHarness(t)
		h.addRepoHook()
		fork := int64(2)
		h.prs.prs[100].SourceRepoID = &fork
		require.NoError(t, h.svc.handleEventPullReqUpdated(ctx, svcEvent(&pullreqevents.UpdatedPayload{Base: base})))
		got := h.receiver.last(t)
		assert.Equal(t, "root/fork", svcGet(t, got.Body, "ref", "repo", "path"))
		assert.Equal(t, "root/repo", svcGet(t, got.Body, "target_ref", "repo", "path"))
	})
	t.Run("fork source repo missing discards", func(t *testing.T) {
		h := newSvcHarness(t)
		fork := int64(77)
		h.prs.prs[100].SourceRepoID = &fork
		err := h.svc.handleEventPullReqUpdated(ctx, svcEvent(&pullreqevents.UpdatedPayload{Base: base}))
		require.Error(t, err)
		assert.True(t, svcIsDiscard(err))
	})
	t.Run("missing activity", func(t *testing.T) {
		h := newSvcHarness(t)
		err := h.svc.handleEventPullReqComment(ctx, svcEvent(&pullreqevents.CommentCreatedPayload{
			Base: base, ActivityID: 999, SourceSHA: svcSHAOne,
		}))
		require.Error(t, err)
		err = h.svc.handleEventPullReqCommentUpdated(ctx, svcEvent(&pullreqevents.CommentUpdatedPayload{
			Base: base, ActivityID: 999,
		}))
		require.Error(t, err)
		err = h.svc.handleEventPullReqCommentStatusUpdated(ctx, svcEvent(&pullreqevents.CommentStatusUpdatedPayload{
			Base: base, ActivityID: 999,
		}))
		require.Error(t, err)
	})
	t.Run("missing label and label value", func(t *testing.T) {
		h := newSvcHarness(t)
		require.Error(t, h.svc.handleEventPullReqLabelAssigned(ctx, svcEvent(&pullreqevents.LabelAssignedPayload{
			Base: base, LabelID: 999,
		})))
		missing := int64(999)
		require.Error(t, h.svc.handleEventPullReqLabelAssigned(ctx, svcEvent(&pullreqevents.LabelAssignedPayload{
			Base: base, LabelID: 3, ValueID: &missing,
		})))
	})
	t.Run("missing reviewer", func(t *testing.T) {
		h := newSvcHarness(t)
		require.Error(t, h.svc.handleEventPullReqReviewSubmitted(ctx, svcEvent(&pullreqevents.ReviewSubmittedPayload{
			Base: base, ReviewerID: 999,
		})))
		require.Error(t, h.svc.handleEventPullReqReviewerAssigned(ctx, svcEvent(&pullreqevents.ReviewerAddedPayload{
			Base: base, ReviewerID: 999,
		})))
		require.Error(t, h.svc.handleEventPullReqReviewerUnassigned(ctx, svcEvent(&pullreqevents.ReviewerRemovedPayload{
			Base: base, ReviewerID: 999,
		})))
	})
	t.Run("missing user group", func(t *testing.T) {
		h := newSvcHarness(t)
		require.Error(t, h.svc.handleEventPullReqUserGroupReviewerAssigned(ctx,
			svcEvent(&pullreqevents.UserGroupReviewerAddedPayload{Base: base, UserGroupReviewerIDs: []int64{999}})))
		require.Error(t, h.svc.handleEventPullReqUserGroupReviewerUnassigned(ctx,
			svcEvent(&pullreqevents.UserGroupReviewerRemovedPayload{Base: base, UserGroupID: 999})))
	})
	t.Run("commit lookups fail", func(t *testing.T) {
		h := newSvcHarness(t)
		h.git.getErr = errors.New("git down")
		h.git.listErr = errors.New("git down")
		require.Error(t, h.svc.handleEventPullReqCreated(ctx, svcEvent(&pullreqevents.CreatedPayload{Base: base})))
		require.Error(t, h.svc.handleEventPullReqReopened(ctx, svcEvent(&pullreqevents.ReopenedPayload{Base: base})))
		require.Error(t, h.svc.handleEventPullReqBranchUpdated(ctx,
			svcEvent(&pullreqevents.BranchUpdatedPayload{Base: base})))
		require.Error(t, h.svc.handleEventPullReqClosed(ctx, svcEvent(&pullreqevents.ClosedPayload{Base: base})))
		require.Error(t, h.svc.handleEventPullReqMerged(ctx, svcEvent(&pullreqevents.MergedPayload{Base: base})))
		require.Error(t, h.svc.handleEventPullReqComment(ctx,
			svcEvent(&pullreqevents.CommentCreatedPayload{Base: base, ActivityID: 200})))
		require.Error(t, h.svc.handleEventPullReqTargetBranchChanged(ctx,
			svcEvent(&pullreqevents.TargetBranchChangedPayload{Base: base})))
	})
}

func TestSvcTriggerReviewerWebhook_UnexpectedTrigger(t *testing.T) {
	h := newSvcHarness(t)
	err := h.svc.triggerReviewerWebhook(context.Background(), enum.WebhookTriggerBranchCreated, "evt", 5, 100,
		func() (PullReqReviewerSegment, error) { return PullReqReviewerSegment{}, nil })
	require.Error(t, err)
}

func TestSvcExtractCodeCommentInfoIfAvailable(t *testing.T) {
	assert.Nil(t, extractCodeCommentInfoIfAvailable(&types.PullReqActivity{Kind: enum.PullReqActivityKindComment}))

	code := extractCodeCommentInfoIfAvailable(&types.PullReqActivity{
		Type: enum.PullReqActivityTypeCodeComment,
		Kind: enum.PullReqActivityKindChangeComment,
		CodeComment: &types.CodeCommentFields{
			Outdated: true, Path: "a.go", LineNew: 3, SpanNew: 1, LineOld: 2, SpanOld: 1,
			MergeBaseSHA: svcSHAOne, SourceSHA: svcSHATwo,
		},
	})
	require.NotNil(t, code)
	assert.Equal(t, "a.go", code.Path)
	assert.True(t, code.Outdated)
}

func TestSvcCheckReportedHandler(t *testing.T) {
	ctx := context.Background()
	payload := func(status enum.CheckStatus, sha string) *checkevents.ReportedPayload {
		return &checkevents.ReportedPayload{
			Base: checkevents.Base{RepoID: 1, SHA: sha}, Identifier: "build", Status: status,
		}
	}

	t.Run("non terminal status is ignored", func(t *testing.T) {
		h := newSvcHarness(t)
		require.NoError(t, h.svc.handleEventCheckReported(ctx, svcEvent(payload(enum.CheckStatusRunning, svcSHAOne))))
		assert.Nil(t, h.prs.lastFilter)
	})
	t.Run("invalid sha discards", func(t *testing.T) {
		h := newSvcHarness(t)
		err := h.svc.handleEventCheckReported(ctx, svcEvent(payload(enum.CheckStatusSuccess, "not-a-sha")))
		require.Error(t, err)
		assert.True(t, svcIsDiscard(err))
	})
	t.Run("list failure", func(t *testing.T) {
		h := newSvcHarness(t)
		h.prs.listErr = errors.New("db down")
		require.Error(t, h.svc.handleEventCheckReported(ctx, svcEvent(payload(enum.CheckStatusSuccess, svcSHAOne))))
	})
	t.Run("fans out to open pull requests", func(t *testing.T) {
		h := newSvcHarness(t)
		h.addRepoHook(enum.WebhookTriggerPullReqCheckCompleted)
		h.prs.listResult = []*types.PullReq{h.prs.prs[100]}

		require.NoError(t, h.svc.handleEventCheckReported(ctx, svcEvent(payload(enum.CheckStatusFailure, svcSHAOne))))

		require.NotNil(t, h.prs.lastFilter)
		assert.Equal(t, int64(1), h.prs.lastFilter.SourceRepoID)
		assert.Equal(t, []enum.PullReqState{enum.PullReqStateOpen}, h.prs.lastFilter.States)

		got := h.receiver.last(t)
		assert.Equal(t, string(enum.WebhookTriggerPullReqCheckCompleted), got.Trigger)
		assert.Equal(t, "build", svcGet(t, got.Body, "check", "identifier"))
		assert.Equal(t, "failure", svcGet(t, got.Body, "check", "status"))
	})
	t.Run("per pull request failures do not fail the event", func(t *testing.T) {
		h := newSvcHarness(t)
		h.prs.listResult = []*types.PullReq{{ID: 555, Number: 9, CreatedBy: 5}}
		require.NoError(t, h.svc.handleEventCheckReported(ctx, svcEvent(payload(enum.CheckStatusSuccess, svcSHAOne))))
	})
}

func TestSvcMergeQueueHandlers(t *testing.T) {
	ctx := context.Background()
	base := mergequeueevents.Base{RepoID: 1, Branch: "main"}

	t.Run("checks requested", func(t *testing.T) {
		h := newSvcHarness(t)
		h.addRepoHook(enum.WebhookTriggerMergeQueueChecksRequested)
		require.NoError(t, h.svc.handleEventMergeQueueChecksRequested(ctx,
			svcEvent(&mergequeueevents.ChecksRequestedPayload{Base: base, PrincipalID: 5, CommitSHA: svcSHAOne})))
		got := h.receiver.last(t)
		assert.Equal(t, "main", svcGet(t, got.Body, "merge_queue", "branch"))
		assert.Equal(t, svcSHAOne, svcGet(t, got.Body, "commit_sha"))
		assert.Equal(t, svcSHAOne, svcGet(t, got.Body, "sha"))
	})
	t.Run("checks requested commit lookup fails", func(t *testing.T) {
		h := newSvcHarness(t)
		h.git.getErr = errors.New("git down")
		require.Error(t, h.svc.handleEventMergeQueueChecksRequested(ctx,
			svcEvent(&mergequeueevents.ChecksRequestedPayload{Base: base, PrincipalID: 5, CommitSHA: svcSHAOne})))
	})
	t.Run("checks canceled", func(t *testing.T) {
		h := newSvcHarness(t)
		h.addRepoHook(enum.WebhookTriggerMergeQueueChecksCanceled)
		require.NoError(t, h.svc.handleEventMergeQueueChecksCanceled(ctx,
			svcEvent(&mergequeueevents.ChecksCanceledPayload{Base: base, PrincipalID: 5, CommitSHA: svcSHATwo})))
		got := h.receiver.last(t)
		assert.Equal(t, string(enum.WebhookTriggerMergeQueueChecksCanceled), got.Trigger)
		assert.Equal(t, svcSHATwo, svcGet(t, got.Body, "sha"))
	})
}
