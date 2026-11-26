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

package importer

import (
	"context"
	"testing"

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/git/api"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

const (
	svaDroneYAML = "kind: pipeline\ntype: docker\nname: default\nsteps:\n" +
		"- name: test\n  image: golang\n  commands:\n  - go test ./...\n"
	svaGitHubYAML = "name: CI\non: [push]\njobs:\n  build:\n    runs-on: ubuntu-latest\n    steps:\n" +
		"    - uses: actions/checkout@v3\n    - run: echo hello\n"
)

var svaPrincipal = &types.Principal{ID: 3, DisplayName: "System", Email: "system@example.com"}

func svaNewImporter(g *svaGit) *Importer {
	return &Importer{
		defaultBranch: "main",
		urlProvider:   svaURLProvider{},
		git:           g,
		tx:            svaTx{},
		pipelineStore: &svaPipelineStore{},
		triggerStore:  &svaTriggerStore{},
	}
}

func svaMatchFilesByKey(files map[string][]api.FileContent) func(*git.MatchFilesParams) (*git.MatchFilesOutput, error) {
	return func(params *git.MatchFilesParams) (*git.MatchFilesOutput, error) {
		return &git.MatchFilesOutput{Files: files[params.DirPath+"|"+params.Pattern]}, nil
	}
}

func TestSvaPipelineOptionEnum(t *testing.T) {
	t.Parallel()

	require.Equal(t, []any{PipelineOptionConvert, PipelineOptionIgnore}, PipelineOption("").Enum())
	require.Len(t, ProviderType("").Enum(), 8)
}

func TestSvaImporterCreateGitRepository(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		var got *git.CreateRepositoryParams
		imp := svaNewImporter(&svaGit{createRepo: func(p *git.CreateRepositoryParams) (*git.CreateRepositoryOutput, error) {
			got = p
			return &git.CreateRepositoryOutput{UID: "git-uid"}, nil
		}})

		uid, err := imp.createGitRepository(context.Background(), svaPrincipal, 7)
		require.NoError(t, err)
		require.Equal(t, "git-uid", uid)
		require.Equal(t, "main", got.DefaultBranch)
		require.Equal(t, "System", got.Actor.Name)
		require.Equal(t, "system@example.com", got.Author.Email)
		require.NotEmpty(t, got.EnvVars)
		require.Nil(t, got.Files)
	})

	t.Run("git failure", func(t *testing.T) {
		t.Parallel()

		imp := svaNewImporter(&svaGit{createRepo: func(*git.CreateRepositoryParams) (*git.CreateRepositoryOutput, error) {
			return nil, errSvaBoom
		}})

		_, err := imp.createGitRepository(context.Background(), svaPrincipal, 7)
		require.ErrorIs(t, err, errSvaBoom)
	})
}

func TestSvaImporterSyncGitRepository(t *testing.T) {
	t.Parallel()

	repo := &types.Repository{ID: 7, GitUID: "git-uid", DefaultBranch: "trunk"}

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		var got *git.SyncRepositoryParams
		imp := svaNewImporter(&svaGit{syncRepo: func(p *git.SyncRepositoryParams) (*git.SyncRepositoryOutput, error) {
			got = p
			return &git.SyncRepositoryOutput{DefaultBranch: "develop"}, nil
		}})

		branch, err := imp.syncGitRepository(context.Background(), svaPrincipal, repo, "https://u:p@example.com/r.git")
		require.NoError(t, err)
		require.Equal(t, "develop", branch)
		require.Equal(t, "https://u:p@example.com/r.git", got.Source)
		require.Equal(t, "git-uid", got.RepoUID)
		require.Equal(t, "trunk", got.DefaultBranch)
		require.False(t, got.CreateIfNotExists)
		require.Equal(t, []string{"refs/heads/*:refs/heads/*", "refs/tags/*:refs/tags/*"}, got.RefSpecs)
	})

	t.Run("git failure", func(t *testing.T) {
		t.Parallel()

		imp := svaNewImporter(&svaGit{syncRepo: func(*git.SyncRepositoryParams) (*git.SyncRepositoryOutput, error) {
			return nil, errSvaBoom
		}})

		_, err := imp.syncGitRepository(context.Background(), svaPrincipal, repo, "x")
		require.ErrorIs(t, err, errSvaBoom)
	})
}

func TestSvaImporterDeleteGitRepository(t *testing.T) {
	t.Parallel()

	repo := &types.Repository{ID: 7, GitUID: "git-uid"}

	tests := []struct {
		name    string
		gitErr  error
		wantErr bool
	}{
		{name: "success"},
		{name: "not found is ignored", gitErr: errors.NotFound("gone")},
		{name: "other failure", gitErr: errSvaBoom, wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			var got *git.DeleteRepositoryParams
			imp := svaNewImporter(&svaGit{deleteRepo: func(p *git.DeleteRepositoryParams) error {
				got = p
				return test.gitErr
			}})

			err := imp.deleteGitRepository(context.Background(), svaPrincipal, repo)
			require.Equal(t, "git-uid", got.RepoUID)
			if test.wantErr {
				require.ErrorIs(t, err, errSvaBoom)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestSvaImporterMatchFiles(t *testing.T) {
	t.Parallel()

	repo := &types.Repository{GitUID: "git-uid"}

	t.Run("maps files", func(t *testing.T) {
		t.Parallel()

		var got *git.MatchFilesParams
		imp := svaNewImporter(&svaGit{matchFiles: func(p *git.MatchFilesParams) (*git.MatchFilesOutput, error) {
			got = p
			return &git.MatchFilesOutput{Files: []api.FileContent{{Path: "a.yml", Content: []byte("x")}}}, nil
		}})

		files, err := imp.matchFiles(context.Background(), repo, "main", "dir", "*.yml", 10)
		require.NoError(t, err)
		require.Equal(t, []pipelineFile{{OriginalPath: "a.yml", Content: []byte("x")}}, files)
		require.Equal(t, "git-uid", got.RepoUID)
		require.Equal(t, "main", got.Ref)
		require.Equal(t, "dir", got.DirPath)
		require.Equal(t, "*.yml", got.Pattern)
		require.Equal(t, 10, got.MaxSize)
	})

	t.Run("git failure", func(t *testing.T) {
		t.Parallel()

		imp := svaNewImporter(&svaGit{matchFiles: func(*git.MatchFilesParams) (*git.MatchFilesOutput, error) {
			return nil, errSvaBoom
		}})

		_, err := imp.matchFiles(context.Background(), repo, "main", "", "x", 1)
		require.ErrorIs(t, err, errSvaBoom)
	})
}

type svaFailingConverter struct{}

func (svaFailingConverter) ConvertBytes([]byte) ([]byte, error) { return nil, errSvaBoom }

type svaPrefixConverter struct{}

func (svaPrefixConverter) ConvertBytes(in []byte) ([]byte, error) {
	return append([]byte("converted:"), in...), nil
}

func TestSvaConvertPipelineFiles(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	gen := func() pipelineConverter { return svaPrefixConverter{} }

	t.Run("single file uses the default pipeline name", func(t *testing.T) {
		t.Parallel()

		got := convertPipelineFiles(ctx, []pipelineFile{{OriginalPath: "x/ci.yml", Content: []byte("a")}}, gen)
		require.Equal(t, []pipelineFile{{
			Name: "default-pipeline", OriginalPath: "x/ci.yml",
			ConvertedPath: ".harness/pipeline.yaml", Content: []byte("converted:a"),
		}}, got)
	})

	t.Run("multiple files are named after the original file", func(t *testing.T) {
		t.Parallel()

		got := convertPipelineFiles(ctx, []pipelineFile{
			{OriginalPath: ".github/workflows/build.yml", Content: []byte("a")},
			{OriginalPath: ".github/workflows/lint.yaml", Content: []byte("b")},
		}, gen)
		require.Len(t, got, 2)
		require.Equal(t, "pipeline-build", got[0].Name)
		require.Equal(t, ".harness/build.yaml", got[0].ConvertedPath)
		require.Equal(t, "pipeline-lint", got[1].Name)
		require.Equal(t, ".harness/lint.yaml", got[1].ConvertedPath)
	})

	t.Run("failed conversions are skipped", func(t *testing.T) {
		t.Parallel()

		got := convertPipelineFiles(ctx, []pipelineFile{{OriginalPath: "a.yml"}},
			func() pipelineConverter { return svaFailingConverter{} })
		require.Empty(t, got)
	})
}

func TestSvaConvertPipelines(t *testing.T) {
	t.Parallel()

	repo := &types.Repository{GitUID: "git-uid", DefaultBranch: "main"}

	tests := []struct {
		name      string
		files     map[string][]api.FileContent
		wantPaths []string
		wantNone  bool
	}{
		{
			name:      "drone",
			files:     map[string][]api.FileContent{"|.drone.yml": {{Path: ".drone.yml", Content: []byte(svaDroneYAML)}}},
			wantPaths: []string{".harness/pipeline.yaml"},
		},
		{
			name: "github workflows from both extensions",
			files: map[string][]api.FileContent{
				".github/workflows|*.yml":  {{Path: ".github/workflows/a.yml", Content: []byte(svaGitHubYAML)}},
				".github/workflows|*.yaml": {{Path: ".github/workflows/b.yaml", Content: []byte(svaGitHubYAML)}},
			},
			wantPaths: []string{".harness/a.yaml", ".harness/b.yaml"},
		},
		{
			name:     "unconvertible drone file falls through to nothing",
			files:    map[string][]api.FileContent{"|.drone.yml": {{Path: ".drone.yml", Content: []byte("::: [")}}},
			wantNone: true,
		},
		{
			name:     "no pipelines",
			files:    map[string][]api.FileContent{},
			wantNone: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			imp := svaNewImporter(&svaGit{matchFiles: svaMatchFilesByKey(test.files)})

			got := imp.convertPipelines(context.Background(), repo)
			if test.wantNone {
				require.Empty(t, got)
				return
			}

			paths := make([]string, len(got))
			for i := range got {
				paths[i] = got[i].ConvertedPath
				require.NotEmpty(t, got[i].Content)
			}
			require.Equal(t, test.wantPaths, paths)
		})
	}

	t.Run("search failures are tolerated", func(t *testing.T) {
		t.Parallel()

		imp := svaNewImporter(&svaGit{matchFiles: func(*git.MatchFilesParams) (*git.MatchFilesOutput, error) {
			return nil, errSvaBoom
		}})
		require.Empty(t, imp.convertPipelines(context.Background(), repo))
	})
}

func TestSvaConvertPipelinesPrecedence(t *testing.T) {
	t.Parallel()

	// A convertible bitbucket, gitlab or circle pipeline is found only when the earlier formats are absent.
	repo := &types.Repository{GitUID: "git-uid", DefaultBranch: "main"}

	const (
		bitbucketYAML = "pipelines:\n  default:\n    - step:\n        script:\n          - echo hello\n"
		gitlabYAML    = "build:\n  script:\n    - echo hello\n"
		circleYAML    = "version: 2.1\njobs:\n  build:\n    docker:\n      - image: cimg/base:stable\n" +
			"    steps:\n      - run: echo hello\nworkflows:\n  main:\n    jobs:\n      - build\n"
	)

	tests := []struct {
		name string
		key  string
		path string
		body string
	}{
		{name: "bitbucket", key: "|bitbucket-pipelines.yml", path: "bitbucket-pipelines.yml", body: bitbucketYAML},
		{name: "gitlab", key: "|.gitlab-ci.yml", path: ".gitlab-ci.yml", body: gitlabYAML},
		{name: "circle", key: ".circleci|config.yml", path: ".circleci/config.yml", body: circleYAML},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			imp := svaNewImporter(&svaGit{matchFiles: svaMatchFilesByKey(map[string][]api.FileContent{
				test.key: {{Path: test.path, Content: []byte(test.body)}},
			})})

			got := imp.convertPipelines(context.Background(), repo)
			require.Len(t, got, 1)
			require.Equal(t, ".harness/pipeline.yaml", got[0].ConvertedPath)
		})
	}
}

func TestSvaProcessPipelines(t *testing.T) {
	t.Parallel()

	repo := &types.Repository{ID: 7, GitUID: "git-uid", DefaultBranch: "main"}
	drone := map[string][]api.FileContent{"|.drone.yml": {{Path: ".drone.yml", Content: []byte(svaDroneYAML)}}}

	t.Run("commits converted files and creates pipelines with triggers", func(t *testing.T) {
		t.Parallel()

		var committed *git.CommitFilesParams
		imp := svaNewImporter(&svaGit{
			matchFiles: svaMatchFilesByKey(drone),
			commit: func(p *git.CommitFilesParams) (git.CommitFilesResponse, error) {
				committed = p
				return git.CommitFilesResponse{}, nil
			},
		})

		err := imp.processPipelines(context.Background(), svaPrincipal, repo, "convert", enum.GitOpTypeAPIContent)
		require.NoError(t, err)

		require.Equal(t, "convert", committed.Message)
		require.Equal(t, "main", committed.Branch)
		require.Equal(t, "main", committed.NewBranch)
		require.Len(t, committed.Actions, 1)
		require.Equal(t, git.CreateAction, committed.Actions[0].Action)
		require.Equal(t, ".harness/pipeline.yaml", committed.Actions[0].Path)

		pipelines, ok := imp.pipelineStore.(*svaPipelineStore)
		require.True(t, ok)
		require.Len(t, pipelines.created, 1)
		require.Equal(t, "default-pipeline", pipelines.created[0].Identifier)
		require.Equal(t, int64(7), pipelines.created[0].RepoID)
		require.Equal(t, ".harness/pipeline.yaml", pipelines.created[0].ConfigPath)

		triggers, ok := imp.triggerStore.(*svaTriggerStore)
		require.True(t, ok)
		require.Len(t, triggers.created, 1)
		require.Equal(t, "default", triggers.created[0].Identifier)
		require.Equal(t, pipelines.created[0].ID, triggers.created[0].PipelineID)
		require.Len(t, triggers.created[0].Actions, 3)
	})

	t.Run("nothing to convert", func(t *testing.T) {
		t.Parallel()

		imp := svaNewImporter(&svaGit{matchFiles: svaMatchFilesByKey(nil)})
		require.NoError(t, imp.processPipelines(
			context.Background(), svaPrincipal, repo, "convert", enum.GitOpTypeManageRepo))
	})

	errTests := []struct {
		name  string
		setup func(imp *Importer, g *svaGit)
	}{
		{name: "commit", setup: func(_ *Importer, g *svaGit) {
			g.commit = func(*git.CommitFilesParams) (git.CommitFilesResponse, error) {
				return git.CommitFilesResponse{}, errSvaBoom
			}
		}},
		{name: "transaction", setup: func(imp *Importer, _ *svaGit) { imp.tx = svaTx{err: errSvaBoom} }},
		{name: "pipeline creation", setup: func(imp *Importer, _ *svaGit) {
			imp.pipelineStore = &svaPipelineStore{err: errSvaBoom}
		}},
		{name: "trigger creation", setup: func(imp *Importer, _ *svaGit) {
			imp.triggerStore = &svaTriggerStore{err: errSvaBoom}
		}},
	}

	for _, test := range errTests {
		t.Run("error "+test.name, func(t *testing.T) {
			t.Parallel()

			g := &svaGit{
				matchFiles: svaMatchFilesByKey(drone),
				commit:     func(*git.CommitFilesParams) (git.CommitFilesResponse, error) { return git.CommitFilesResponse{}, nil },
			}
			imp := svaNewImporter(g)
			test.setup(imp, g)

			err := imp.processPipelines(context.Background(), svaPrincipal, repo, "convert", enum.GitOpTypeAPIContent)
			require.ErrorIs(t, err, errSvaBoom)
		})
	}
}
