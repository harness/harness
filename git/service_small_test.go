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

package git

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/harness/gitness/errors"

	"github.com/stretchr/testify/require"
)

func TestHunkHeader(t *testing.T) {
	tests := []struct {
		name    string
		header  HunkHeader
		isZero  bool
		isValid bool
		str     string
	}{
		{"zero", HunkHeader{}, true, false, "@@ -0,0 +0,0 @@"},
		{"single line spans", HunkHeader{OldLine: 3, OldSpan: 1, NewLine: 4, NewSpan: 1}, false, true, "@@ -3 +4 @@"},
		{"multi line", HunkHeader{OldLine: 10, OldSpan: 5, NewLine: 10, NewSpan: 7, Text: "func main()"},
			false, true, "@@ -10,5 +10,7 @@ func main()"},
		{"added only", HunkHeader{NewLine: 1, NewSpan: 3}, false, true, "@@ -0,0 +1,3 @@"},
		{"invalid span", HunkHeader{OldLine: 1, NewLine: 1, NewSpan: 1}, false, false, "@@ -1,0 +1 @@"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.isZero, tt.header.IsZero())
			require.Equal(t, tt.isValid, tt.header.IsValid())
			require.Equal(t, tt.str, tt.header.String())
		})
	}
}

func TestBlameParams_Validate(t *testing.T) {
	var nilParams *BlameParams
	require.ErrorIs(t, nilParams.Validate(), ErrNoParamsProvided)

	valid := func() *BlameParams {
		return &BlameParams{ReadParams: ReadParams{RepoUID: "r"}, GitRef: "main", Path: "a.txt"}
	}
	require.NoError(t, valid().Validate())

	for name, mutate := range map[string]func(*BlameParams){
		"no repo":       func(p *BlameParams) { p.RepoUID = "" },
		"no ref":        func(p *BlameParams) { p.GitRef = "" },
		"no path":       func(p *BlameParams) { p.Path = "" },
		"negative from": func(p *BlameParams) { p.LineFrom = -1 },
		"negative to":   func(p *BlameParams) { p.LineTo = -1 },
		"from after to": func(p *BlameParams) { p.LineFrom, p.LineTo = 5, 2 },
	} {
		t.Run(name, func(t *testing.T) {
			p := valid()
			mutate(p)
			require.Error(t, p.Validate())
		})
	}
}

func TestService_Blame(t *testing.T) {
	f := newOpsFixture(t)

	collect := func(params *BlameParams) ([]*BlamePart, error) {
		ch, chErr := f.svc.Blame(t.Context(), params)
		var parts []*BlamePart
		for part := range ch {
			parts = append(parts, part)
		}
		return parts, <-chErr
	}

	parts, err := collect(&BlameParams{ReadParams: f.read(), GitRef: "main", Path: "a.txt"})
	require.NoError(t, err)
	require.Len(t, parts, 1)
	require.Equal(t, []string{"a"}, parts[0].Lines)
	require.Equal(t, f.head(t.Context(), t), parts[0].Commit.SHA.String())
	require.Nil(t, parts[0].Previous)

	_, err = collect(&BlameParams{ReadParams: f.read(), GitRef: "main"})
	require.Error(t, err)
}

func TestRepackParams_Validate(t *testing.T) {
	expire := time.Now()
	require.NoError(t, RepackParams{Strategy: RepackStrategyIncrementalWithUnreachable}.Validate())
	require.Error(t, RepackParams{Strategy: RepackStrategyIncrementalWithUnreachable, WriteBitmap: true}.Validate())
	require.Error(t, RepackParams{
		Strategy: RepackStrategyIncrementalWithUnreachable, WriteMultiPackIndex: true,
	}.Validate())
	require.Error(t, RepackParams{Strategy: RepackStrategyFullWithCruft}.Validate())
	require.NoError(t, RepackParams{Strategy: RepackStrategyFullWithCruft, CruftExpireBefore: expire}.Validate())
	require.NoError(t, RepackParams{Strategy: RepackStrategyFullWithUnreachable}.Validate())
	require.NoError(t, RepackParams{Strategy: RepackStrategyGeometric}.Validate())
	require.Error(t, RepackParams{Strategy: "bogus"}.Validate())
}

func TestService_RepackObjects(t *testing.T) {
	f := newOpsFixture(t)

	for _, params := range []RepackParams{
		{Strategy: RepackStrategyIncrementalWithUnreachable},
		{Strategy: RepackStrategyFullWithUnreachable},
		{Strategy: RepackStrategyFullWithUnreachable, WriteMultiPackIndex: true},
		{Strategy: RepackStrategyFullWithCruft, CruftExpireBefore: time.Now().Add(-time.Hour)},
		{Strategy: RepackStrategyGeometric},
	} {
		require.NoError(t, f.svc.repackObjects(t.Context(), f.repoPath, params), "%+v", params)
		require.FileExists(t, filepath.Join(f.repoPath, ".harness-full-repack-timestamp"))
	}

	require.Error(t, f.svc.repackObjects(t.Context(), f.repoPath, RepackParams{Strategy: "bogus"}))
	require.Error(t, f.svc.repackObjects(t.Context(), filepath.Join(f.repoPath, "missing"),
		RepackParams{Strategy: RepackStrategyFullWithUnreachable}))
}

func TestService_PushRemote(t *testing.T) {
	f := newOpsFixture(t)

	var nilParams *PushRemoteParams
	require.ErrorIs(t, nilParams.Validate(), ErrNoParamsProvided)
	require.Error(t, (&PushRemoteParams{}).Validate())
	require.Error(t, (&PushRemoteParams{ReadParams: f.read()}).Validate())

	remote := filepath.Join(t.TempDir(), "remote.git")
	runGit(t, t.TempDir(), "init", "--bare", remote)

	require.NoError(t, f.svc.PushRemote(t.Context(), &PushRemoteParams{ReadParams: f.read(), RemoteURL: remote}))
	require.Equal(t, f.head(t.Context(), t), runGit(t, remote, "rev-parse", "main"))

	err := f.svc.PushRemote(t.Context(), &PushRemoteParams{
		ReadParams: f.read(), RemoteURL: filepath.Join(t.TempDir(), "missing.git"),
	})
	require.Error(t, err)
}

func TestService_PushRemote_EmptyRepo(t *testing.T) {
	requireGit(t)
	s, _ := newTestService(t)
	out, err := s.CreateRepository(t.Context(), &CreateRepositoryParams{
		Actor: Identity{Name: "A", Email: "a@example.com"}, DefaultBranch: "main",
	})
	require.NoError(t, err)

	err = s.PushRemote(t.Context(), &PushRemoteParams{
		ReadParams: ReadParams{RepoUID: out.UID}, RemoteURL: t.TempDir(),
	})
	// NOTE: HasBranches reports an empty repository as "true".
	require.True(t, errors.IsInvalidArgument(err), "got %v", err)
}

func TestService_Submodule(t *testing.T) {
	requireGit(t)
	s, _ := newTestService(t)
	out, err := s.CreateRepository(t.Context(), &CreateRepositoryParams{
		Actor:         Identity{Name: "A", Email: "a@example.com"},
		DefaultBranch: "main",
		Files: []File{{
			Path:    ".gitmodules",
			Content: []byte("[submodule \"lib\"]\n\tpath = vendor/lib\n\turl = https://example.com/lib.git\n"),
		}},
	})
	require.NoError(t, err)
	read := ReadParams{RepoUID: out.UID}

	got, err := s.GetSubmodule(t.Context(), &GetSubmoduleParams{ReadParams: read, GitREF: "main", Path: "vendor/lib"})
	require.NoError(t, err)
	require.Equal(t, Submodule{Name: "vendor/lib", URL: "https://example.com/lib.git"}, got.Submodule)

	_, err = s.GetSubmodule(t.Context(), &GetSubmoduleParams{ReadParams: read, GitREF: "missing", Path: "vendor/lib"})
	require.Error(t, err)
}

func TestService_ResolveRevisionAndMatchFiles(t *testing.T) {
	f := newOpsFixture(t)

	out, err := f.svc.ResolveRevision(t.Context(), ResolveRevisionParams{ReadParams: f.read(), Revision: "main"})
	require.NoError(t, err)
	require.Equal(t, f.head(t.Context(), t), out.SHA.String())

	_, err = f.svc.ResolveRevision(t.Context(), ResolveRevisionParams{ReadParams: f.read(), Revision: "missing"})
	require.Error(t, err)

	files, err := f.svc.MatchFiles(t.Context(), &MatchFilesParams{
		ReadParams: f.read(), Ref: "main", DirPath: "dir", Pattern: "*.txt", MaxSize: 100,
	})
	require.NoError(t, err)
	require.Len(t, files.Files, 1)
	require.Equal(t, "dir/b.txt", files.Files[0].Path)

	_, err = f.svc.MatchFiles(t.Context(), &MatchFilesParams{
		ReadParams: f.read(), Ref: "main", DirPath: "dir", Pattern: "[", MaxSize: 100,
	})
	require.Error(t, err)
}
