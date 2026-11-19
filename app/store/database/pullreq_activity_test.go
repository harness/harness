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
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// activityPrincipalInfoCacheStub resolves single principal infos as well.
type activityPrincipalInfoCacheStub struct {
	principalInfoCacheStub
}

func (activityPrincipalInfoCacheStub) Get(_ context.Context, id int64) (*types.PrincipalInfo, error) {
	return &types.PrincipalInfo{ID: id}, nil
}

type extendedCodeCommentPayload struct {
	types.PullRequestActivityPayloadCodeComment
	Extra string `json:"extra"`
}

func TestPullReqActivityStore_ReviewComment(t *testing.T) {
	db, teardown := setupDB(t)
	defer teardown()

	principalStore, spaceStore, spacePathStore, repoStore := setupStores(t, db)
	ctx := context.Background()

	createUser(ctx, t, principalStore)
	createSpace(ctx, t, spaceStore, spacePathStore, userID, 1, 0)
	createRepo(ctx, t, repoStore, 1, 1, 0)
	insertPullReqRaw(ctx, t, db, 100, 1, nil)

	activities := database.NewPullReqActivityStore(db, activityPrincipalInfoCacheStub{})

	codeComment := func(kind enum.PullReqActivityKind) *types.PullReqActivity {
		return &types.PullReqActivity{
			Type: enum.PullReqActivityTypeCodeComment,
			Kind: kind,
			CodeComment: &types.CodeCommentFields{
				MergeBaseSHA: "base", SourceSHA: "source", Path: "a.go", LineNew: 81, SpanNew: 1, LineOld: 80, SpanOld: 1,
			},
		}
	}

	resolved := int64(1)
	acts := []*types.PullReqActivity{
		{Type: enum.PullReqActivityTypeComment, Kind: enum.PullReqActivityKindComment},
		codeComment(enum.PullReqActivityKindChangeComment),
		{Type: enum.PullReqActivityTypeComment, Kind: enum.PullReqActivityKindReviewComment},
		codeComment(enum.PullReqActivityKindReviewComment),
		{Type: enum.PullReqActivityTypeComment, Kind: enum.PullReqActivityKindComment, Resolved: &resolved},
	}
	for i, act := range acts {
		act.CreatedBy = userID
		act.RepoID = 1
		act.PullReqID = 100
		act.Order = int64(i + 1)
		act.Text = "comment"

		var err error
		if act.Type == enum.PullReqActivityTypeCodeComment {
			err = act.SetPayload(&extendedCodeCommentPayload{
				PullRequestActivityPayloadCodeComment: types.PullRequestActivityPayloadCodeComment{
					Title: "@@ -80 +81 @@", Lines: []string{"+x"},
				},
				Extra: "k1",
			})
		} else {
			err = act.SetPayload(types.PullRequestActivityPayloadComment{})
		}
		if err != nil {
			t.Fatalf("SetPayload: %v", err)
		}
		if err = activities.Create(ctx, act); err != nil {
			t.Fatalf("Create: %v", err)
		}
	}

	find := func(id int64) *types.PullReqActivity {
		t.Helper()
		list, err := activities.List(ctx, 100, &types.PullReqActivityFilter{})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		for _, act := range list {
			if act.ID == id {
				return act
			}
		}
		t.Fatalf("activity %d not found", id)
		return nil
	}

	review := find(acts[3].ID)
	if !review.IsValidCodeComment() || review.CodeComment.Path != "a.go" || review.CodeComment.LineNew != 81 {
		t.Fatalf("review comment on code = %+v", review.CodeComment)
	}

	review.Text = "edited"
	if err := activities.Update(ctx, review); err != nil {
		t.Fatalf("Update: %v", err)
	}

	payload := &extendedCodeCommentPayload{}
	if err := json.Unmarshal(find(acts[3].ID).PayloadRaw, payload); err != nil {
		t.Fatalf("Unmarshal payload: %v", err)
	}
	if payload.Extra != "k1" || payload.Title != "@@ -80 +81 @@" {
		t.Fatalf("payload after update = %+v", payload)
	}

	count, err := activities.CountUnresolved(ctx, 100)
	if err != nil {
		t.Fatalf("CountUnresolved: %v", err)
	}
	if count != 2 {
		t.Fatalf("CountUnresolved = %d, want 2", count)
	}
}
