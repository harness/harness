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

package enum

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGtxMergeMethod_Sanitize(t *testing.T) {
	tests := []struct {
		name   string
		in     MergeMethod
		want   MergeMethod
		wantOK bool
	}{
		{name: "merge", in: MergeMethodMerge, want: MergeMethodMerge, wantOK: true},
		{name: "squash", in: MergeMethodSquash, want: MergeMethodSquash, wantOK: true},
		{name: "rebase", in: MergeMethodRebase, want: MergeMethodRebase, wantOK: true},
		{name: "fast-forward", in: MergeMethodFastForward, want: MergeMethodFastForward, wantOK: true},
		{name: "empty falls back to merge", in: "", want: MergeMethodMerge},
		{name: "unknown falls back to merge", in: "gtx-unknown", want: MergeMethodMerge},
		{name: "case sensitive", in: "SQUASH", want: MergeMethodMerge},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := test.in.Sanitize()
			assert.Equal(t, test.want, got)
			assert.Equal(t, test.wantOK, ok)
		})
	}

	for _, method := range MergeMethods {
		got, ok := method.Sanitize()
		assert.True(t, ok)
		assert.Equal(t, method, got)
	}
}

func TestGtxRefType_String(t *testing.T) {
	tests := []struct {
		in   RefType
		want string
	}{
		{RefTypeRaw, "raw"},
		{RefTypeBranch, "branch"},
		{RefTypeTag, "tag"},
		{RefTypePullReqHead, "head"},
		{RefTypePullReqMerge, "merge"},
		{RefTypePullReqMergeQueue, "merge_queue"},
		{RefType(99), ""},
	}

	for _, test := range tests {
		t.Run(test.want, func(t *testing.T) {
			assert.Equal(t, test.want, test.in.String())
		})
	}
}
