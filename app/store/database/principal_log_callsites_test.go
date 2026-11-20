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

package database

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/guregu/null"
	"github.com/rs/zerolog"
)

// requireLoggedAt fails the test unless every message was logged at the expected level.
func requireLoggedAt(t *testing.T, buf *bytes.Buffer, level string, msgs ...string) {
	t.Helper()

	want := fmt.Sprintf(`"level":"%s"`, level)
	for _, msg := range msgs {
		found := false
		for _, line := range bytes.Split(buf.Bytes(), []byte("\n")) {
			if bytes.Contains(line, []byte(msg)) && bytes.Contains(line, []byte(want)) {
				found = true
			}
		}
		if !found {
			t.Errorf("%q not logged at %s; log: %s", msg, level, buf.String())
		}
	}
}

// debugCtx returns a context carrying a logger that writes every level to buf.
func debugCtx(buf *bytes.Buffer) context.Context {
	return zerolog.New(buf).Level(zerolog.DebugLevel).WithContext(context.Background())
}

// principalLoadCases are the two outcomes every principal-info call site has to distinguish: a cancelled
// request is the caller's doing and logs at debug, anything else is a server fault and logs at error.
var principalLoadCases = []struct {
	name      string
	err       error
	wantLevel string
}{
	{"context canceled is debug", fmt.Errorf("cache: failed to find one: %w", context.Canceled), "debug"},
	{"other failure is error", fmt.Errorf("db down"), "error"},
}

func TestMapPullReqActivity_PrincipalLoadErrorLevel(t *testing.T) {
	for _, tt := range principalLoadCases {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			ctx := debugCtx(&buf)

			s := &PullReqActivityStore{pCache: principalCacheStub{err: tt.err}}
			act := &pullReqActivity{
				CreatedBy:  1,
				ResolvedBy: null.Int{NullInt64: sql.NullInt64{Int64: 2, Valid: true}},
				Metadata:   []byte("{}"),
			}

			got, err := s.mapPullReqActivity(ctx, act)
			if err != nil {
				t.Fatalf("mapPullReqActivity: %v", err)
			}
			if got == nil {
				t.Fatal("expected activity")
			}

			requireLoggedAt(t, &buf, tt.wantLevel,
				"failed to load PR activity author", "failed to load PR activity resolver")
		})
	}
}

func TestMapPullReqReviewer_PrincipalLoadErrorLevel(t *testing.T) {
	for _, tt := range principalLoadCases {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			ctx := debugCtx(&buf)

			s := &PullReqReviewerStore{pCache: principalCacheStub{err: tt.err}}

			got := s.mapPullReqReviewer(ctx, &pullReqReviewer{CreatedBy: 1, PrincipalID: 2})
			if got == nil {
				t.Fatal("expected reviewer")
			}

			requireLoggedAt(t, &buf, tt.wantLevel,
				"failed to load PR reviewer addedBy", "failed to load PR reviewer principal")
		})
	}
}

func TestMapToRule_PrincipalLoadErrorLevel(t *testing.T) {
	for _, tt := range principalLoadCases {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			ctx := debugCtx(&buf)

			s := &RuleStore{pCache: principalCacheStub{err: tt.err}}

			s.mapToRule(ctx, &rule{CreatedBy: 1, Definition: "{}", RepoTarget: "{}"})

			requireLoggedAt(t, &buf, tt.wantLevel, "failed to load rule creator")
		})
	}
}
