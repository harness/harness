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

package jsonnet

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/harness/gitness/app/pipeline/file"
	"github.com/harness/gitness/types"

	"github.com/stretchr/testify/require"
)

type infFileService struct {
	files map[string]string
	calls []string
	err   error
}

func (s *infFileService) Get(_ context.Context, _ *types.Repository, path, _ string) (*file.File, error) {
	s.calls = append(s.calls, path)
	if s.err != nil {
		return nil, s.err
	}
	data, ok := s.files[path]
	if !ok {
		return nil, errors.New("file not found: " + path)
	}
	return &file.File{Data: []byte(data)}, nil
}

func infFixtures() (*types.Repository, *types.Pipeline, *types.Execution) {
	repo := &types.Repository{
		Identifier:    "myrepo",
		Path:          "space/sub/myrepo",
		GitURL:        "https://example.invalid/space/sub/myrepo.git",
		DefaultBranch: "main",
	}
	pipeline := &types.Pipeline{ConfigPath: ".drone.jsonnet"}
	execution := &types.Execution{
		Event:  "push",
		Target: "feature",
		After:  "abc123",
		Ref:    "refs/heads/feature",
		Params: map[string]string{"color": "blue"},
	}
	return repo, pipeline, execution
}

func TestInfParse_PlainDocument(t *testing.T) {
	repo, pipeline, execution := infFixtures()
	out, err := Parse(repo, true, pipeline, execution, &file.File{Data: []byte(`{ kind: "pipeline" }`)}, nil, 0)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(out, "---\n"))
	require.Contains(t, out, `"kind": "pipeline"`)
}

func TestInfParse_ArrayProducesDocumentStream(t *testing.T) {
	repo, pipeline, execution := infFixtures()
	src := `[ { name: "one" }, { name: "two" } ]`
	out, err := Parse(repo, true, pipeline, execution, &file.File{Data: []byte(src)}, nil, 0)
	require.NoError(t, err)
	require.Equal(t, 2, strings.Count(out, "---\n"))
	require.Contains(t, out, `"one"`)
	require.Contains(t, out, `"two"`)
}

func TestInfParse_ExternalVariables(t *testing.T) {
	repo, pipeline, execution := infFixtures()
	src := `{
		repo: std.extVar("repo.identifier"),
		namespace: std.extVar("repo.namespace"),
		slug: std.extVar("repo.slug"),
		private: std.extVar("repo.private"),
		branch: std.extVar("repo.branch"),
		config: std.extVar("repo.config"),
		event: std.extVar("build.event"),
		target: std.extVar("build.target"),
		commit: std.extVar("build.commit"),
		color: std.extVar("build.param.color"),
	}`

	out, err := Parse(repo, false, pipeline, execution, &file.File{Data: []byte(src)}, nil, 0)
	require.NoError(t, err)
	for _, want := range []string{
		`"repo": "myrepo"`,
		`"namespace": "space/sub"`,
		`"slug": "space/sub/myrepo"`,
		`"private": "true"`,
		`"branch": "main"`,
		`"config": ".drone.jsonnet"`,
		`"event": "push"`,
		`"target": "feature"`,
		`"commit": "abc123"`,
		`"color": "blue"`,
	} {
		require.Contains(t, out, want)
	}

	out, err = Parse(repo, true, pipeline, execution,
		&file.File{Data: []byte(`{ private: std.extVar("repo.private") }`)}, nil, 0)
	require.NoError(t, err)
	require.Contains(t, out, `"private": "false"`)
}

func TestInfParse_RepoWithoutSlashInPath(t *testing.T) {
	repo, pipeline, _ := infFixtures()
	repo.Path = "flat"
	out, err := Parse(repo, true, pipeline, nil,
		&file.File{Data: []byte(`{ ns: std.extVar("repo.namespace") }`)}, nil, 0)
	require.NoError(t, err)
	require.Contains(t, out, `"ns": "flat"`)
}

func TestInfParse_SyntaxError(t *testing.T) {
	repo, pipeline, execution := infFixtures()
	_, err := Parse(repo, true, pipeline, execution, &file.File{Data: []byte(`{ broken`)}, nil, 0)
	require.Error(t, err)
}

func TestInfParse_Imports(t *testing.T) {
	repo, pipeline, execution := infFixtures()
	execution.Ref = "refs/heads/main"
	fs := &infFileService{files: map[string]string{
		"lib/steps.libsonnet": `{ step: "from-lib" }`,
	}}
	src := `local lib = import "lib/steps.libsonnet"; { a: lib.step, again: (import "lib/steps.libsonnet").step }`

	out, err := Parse(repo, true, pipeline, execution, &file.File{Data: []byte(src)}, fs, 10)
	require.NoError(t, err)
	require.Contains(t, out, `"a": "from-lib"`)
	require.Contains(t, out, `"again": "from-lib"`)
	require.Equal(t, []string{"lib/steps.libsonnet"}, fs.calls, "second import must be served from the cache")
}

func TestInfParse_ImportErrors(t *testing.T) {
	repo, pipeline, execution := infFixtures()

	t.Run("file service error", func(t *testing.T) {
		fs := &infFileService{err: errors.New("git down")}
		_, err := Parse(repo, true, pipeline, execution,
			&file.File{Data: []byte(`import "x.libsonnet"`)}, fs, 10)
		require.ErrorContains(t, err, "git down")
	})
	t.Run("import limit", func(t *testing.T) {
		fs := &infFileService{files: map[string]string{"a.libsonnet": "{}", "b.libsonnet": "{}"}}
		src := `[ import "a.libsonnet", import "b.libsonnet" ]`
		_, err := Parse(repo, true, pipeline, execution, &file.File{Data: []byte(src)}, fs, 1)
		require.ErrorContains(t, err, "import limit exceeded")
	})
	t.Run("imports disabled without limit", func(t *testing.T) {
		fs := &infFileService{files: map[string]string{"a.libsonnet": "{}"}}
		_, err := Parse(repo, true, pipeline, execution,
			&file.File{Data: []byte(`import "a.libsonnet"`)}, fs, 0)
		require.Error(t, err)
		require.Empty(t, fs.calls)
	})
}

func TestInfImporter_RejectsParentTraversal(t *testing.T) {
	i := &importer{execution: &types.Execution{}, fileService: &infFileService{}, limit: 5}
	_, _, err := i.Import("../outside.jsonnet", "x.libsonnet")
	require.ErrorContains(t, err, "cannot resolve import")
}

func TestInfImporter_ResolvesRelativeToImportingFile(t *testing.T) {
	fs := &infFileService{files: map[string]string{"dir/sub/lib.libsonnet": "{}"}}
	i := &importer{execution: &types.Execution{Ref: "refs/heads/x"}, fileService: fs, limit: 5}
	_, foundAt, err := i.Import("dir/main.jsonnet", "sub/lib.libsonnet")
	require.NoError(t, err)
	require.Equal(t, "dir/sub/lib.libsonnet", foundAt)
	require.Equal(t, 1, i.count)
}
