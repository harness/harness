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

package aigateway

import (
	"context"
	"testing"
)

func TestUsageIsAttributedFromContext(t *testing.T) {
	g := newTestGateway(t, &fakeOpenRouter{}, 1000)
	us := &fakeUsageStore{}
	g.usage = us

	if _, err := g.Chat(WithAttribution(context.Background(), 11, 22), ClassManual, chatRequest()); err != nil {
		t.Fatal(err)
	}
	// only a repo: the space stays unset; no attribution at all: both unset
	if _, err := g.Chat(WithAttribution(context.Background(), 0, 5), ClassManual, chatRequest()); err != nil {
		t.Fatal(err)
	}
	if _, err := g.Chat(context.Background(), ClassManual, chatRequest()); err != nil {
		t.Fatal(err)
	}

	if len(us.rows) != 3 {
		t.Fatalf("expected 3 usage rows, got %d", len(us.rows))
	}
	if r := us.rows[0]; r.SpaceID == nil || *r.SpaceID != 11 || r.RepoID == nil || *r.RepoID != 22 {
		t.Fatalf("unexpected attribution %+v", r)
	}
	if r := us.rows[1]; r.SpaceID != nil || r.RepoID == nil || *r.RepoID != 5 {
		t.Fatalf("unexpected attribution %+v", r)
	}
	if r := us.rows[2]; r.SpaceID != nil || r.RepoID != nil {
		t.Fatalf("expected no attribution, got %+v", r)
	}
}
