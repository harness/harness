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
	"testing"

	"github.com/harness/gitness/types"

	"github.com/guregu/null"
)

type principalCacheStub struct {
	err error
}

func (c principalCacheStub) Stats() (int64, int64)        { return 0, 0 }
func (c principalCacheStub) Evict(context.Context, int64) {}
func (c principalCacheStub) Get(context.Context, int64) (*types.PrincipalInfo, error) {
	return nil, c.err
}
func (c principalCacheStub) Map(context.Context, []int64) (map[int64]*types.PrincipalInfo, error) {
	return nil, c.err
}

func TestMapPullReq_PrincipalLoadErrorLevel(t *testing.T) {
	for _, tt := range principalLoadCases {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			ctx := debugCtx(&buf)

			s := &PullReqStore{pCache: principalCacheStub{err: tt.err}}
			pr := &pullReq{CreatedBy: 1, MergedBy: null.Int{NullInt64: sql.NullInt64{Int64: 2, Valid: true}}}

			got := s.mapPullReq(ctx, pr)
			if got == nil {
				t.Fatal("expected pull request")
			}

			requireLoggedAt(t, &buf, tt.wantLevel,
				"failed to load PR author", "failed to load PR merger")
		})
	}
}
