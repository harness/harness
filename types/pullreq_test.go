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

package types

import (
	"testing"

	"github.com/harness/gitness/types/enum"

	"github.com/gotidy/ptr"
	"github.com/stretchr/testify/assert"
)

func TestPullReq_IsLinked(t *testing.T) {
	t.Parallel()

	linked := enum.PullReqTypeLinked
	normal := enum.PullReqTypeNormal

	tests := []struct {
		name string
		pr   PullReq
		want bool
	}{
		{
			name: "nil type",
			pr:   PullReq{},
			want: false,
		},
		{
			name: "normal type",
			pr:   PullReq{Type: &normal},
			want: false,
		},
		{
			name: "linked type",
			pr:   PullReq{Type: &linked},
			want: true,
		},
		{
			name: "linked type via ptr",
			pr:   PullReq{Type: ptr.Of(enum.PullReqTypeLinked)},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.pr.IsLinked())
		})
	}
}
