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

package repo

import (
	"context"
	"errors"
	"testing"
	"time"

	repoevents "github.com/harness/gitness/app/events/repo"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/events"
)

type recordingLastPushStore struct {
	store.RepoStore
	repoID   int64
	pushedAt int64
	deadline time.Time
	hasDue   bool
	err      error
}

func (r *recordingLastPushStore) UpdateLastGITPush(ctx context.Context, repoID, pushedAt int64) error {
	r.repoID = repoID
	r.pushedAt = pushedAt
	r.deadline, r.hasDue = ctx.Deadline()
	return r.err
}

func TestHandleLastGITPush_UsesEventTimestamp(t *testing.T) {
	pushedAt := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	store := &recordingLastPushStore{}
	service := &Service{repoStore: store}

	err := service.handleLastGITPush(context.Background(), &events.Event[*repoevents.PushedPayload]{
		Timestamp: pushedAt,
		Payload:   &repoevents.PushedPayload{Base: repoevents.Base{RepoID: 42}},
	})
	if err != nil {
		t.Fatalf("handleLastGITPush() error = %v", err)
	}
	if store.repoID != 42 || store.pushedAt != pushedAt.UnixMilli() {
		t.Fatalf("UpdateLastGITPush(%d, %d), want (42, %d)", store.repoID, store.pushedAt, pushedAt.UnixMilli())
	}
	if !store.hasDue {
		t.Fatal("UpdateLastGITPush context has no deadline")
	}
	remain := time.Until(store.deadline)
	if remain <= 0 || remain > lastGITPushTimeout {
		t.Fatalf("deadline remaining %s, want within %s", remain, lastGITPushTimeout)
	}
}

func TestHandleLastGITPush_ReturnsStoreError(t *testing.T) {
	store := &recordingLastPushStore{err: errors.New("locked")}
	service := &Service{repoStore: store}

	err := service.handleLastGITPush(context.Background(), &events.Event[*repoevents.PushedPayload]{
		Timestamp: time.Now(),
		Payload:   &repoevents.PushedPayload{Base: repoevents.Base{RepoID: 7}},
	})
	if err == nil {
		t.Fatal("handleLastGITPush() succeeded, want the store error")
	}
}
