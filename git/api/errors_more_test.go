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

package api

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPushErrors(t *testing.T) {
	inner := errors.New("inner")

	ood := &PushOutOfDateError{StdOut: "out", StdErr: "err", Err: inner}
	require.Contains(t, ood.Error(), "PushOutOfDate Error")
	require.ErrorIs(t, ood.Unwrap(), inner)

	rejected := &PushRejectedError{StdOut: "out", StdErr: "err", Err: inner}
	require.Contains(t, rejected.Error(), "PushRejected")
	require.ErrorIs(t, rejected.Unwrap(), inner)
	require.True(t, IsErrPushRejected(rejected))
	require.False(t, IsErrPushRejected(inner))

	more := &MoreThanOneError{StdOut: "out", StdErr: "err", Err: inner}
	require.Contains(t, more.Error(), "MoreThanOneError Error")
	require.True(t, IsErrMoreThanOne(more))
	require.False(t, IsErrMoreThanOne(inner))
}

func TestPushRejectedError_GenerateMessage(t *testing.T) {
	tests := []struct {
		name   string
		stdErr string
		want   string
	}{
		{"no remote lines", "something else", ""},
		{"single line", "remote: nope\n", "nope"},
		{"multiple lines", "remote: first\nremote: second\n! [remote rejected]", "first\nsecond"},
		{"no trailing newline", "remote: only", "only"},
		{"stops at non remote line", "remote: a\nlocal: b\n", "a"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &PushRejectedError{StdErr: tt.stdErr}
			e.GenerateMessage()
			require.Equal(t, tt.want, e.Message)
		})
	}
}

func TestUnrelatedHistoriesError(t *testing.T) {
	e := &UnrelatedHistoriesError{BaseRef: "refs/heads/main", HeadRef: "refs/tags/v1"}
	require.Equal(t, "main and v1 have entirely different commit histories.", e.Error())
	require.Equal(t, map[string]any{"base_ref": "refs/heads/main", "head_ref": "refs/tags/v1"}, e.Map())
	require.Equal(t, "unrelated commit histories error", (&UnrelatedHistoriesError{}).Error())

	require.True(t, e.Is(&UnrelatedHistoriesError{BaseRef: "refs/heads/main", HeadRef: "refs/tags/v1"}))
	require.False(t, e.Is(&UnrelatedHistoriesError{BaseRef: "other", HeadRef: "refs/tags/v1"}))
	require.False(t, e.Is(errors.New("x")))

	require.True(t, IsUnrelatedHistoriesError(e))
	require.Same(t, e, AsUnrelatedHistoriesError(e))
	require.False(t, IsUnrelatedHistoriesError(errors.New("x")))
	require.Nil(t, AsUnrelatedHistoriesError(errors.New("x")))
}

func TestIsMergeBaseNonUniqueError(t *testing.T) {
	require.True(t, IsMergeBaseNonUniqueError(ErrMergeBaseNonUnique))
	require.False(t, IsMergeBaseNonUniqueError(errors.New("x")))
}

func TestProcessGitErrorf_Mapping(t *testing.T) {
	notFound := processGitErrorf(errors.New("fatal: no such file or directory"), "x")
	require.Contains(t, notFound.Error(), "repository not found")

	conflict := processGitErrorf(errors.New("fatal: reference already exists"), "x")
	require.Contains(t, conflict.Error(), "reference already exists")

	revision := processGitErrorf(errors.New("fatal: unknown revision"), "x")
	require.Contains(t, revision.Error(), "revision not found")
	revision = processGitErrorf(errors.New("fatal: ambiguous argument"), "x")
	require.Contains(t, revision.Error(), "revision not found")

	withRefs := processGitErrorf(errors.New("fatal: no merge base"), "merge base %s %s", " main ", " feature ")
	var unrelated *UnrelatedHistoriesError
	require.True(t, errors.As(withRefs, &unrelated))
	require.Equal(t, "main", unrelated.BaseRef)
	require.Equal(t, "feature", unrelated.HeadRef)

	withoutRefs := processGitErrorf(errors.New("fatal: no merge base"), "merge base")
	require.True(t, errors.As(withoutRefs, &unrelated))
}
