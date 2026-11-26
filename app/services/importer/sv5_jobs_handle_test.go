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

package importer

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

type sv5Connector struct {
	ConnectorService

	info AccessInfo
	err  error
	got  []ConnectorDef
}

func (c *sv5Connector) GetAccessInfo(_ context.Context, def ConnectorDef) (AccessInfo, error) {
	c.got = append(c.got, def)
	return c.info, c.err
}

type sv5LinkedStore struct {
	store.LinkedRepoStore

	linked    map[int64]*types.LinkedRepo
	findErr   error
	list      []types.LinkedRepo
	listErr   error
	updateErr error
	updated   []int64
}

func (s *sv5LinkedStore) Find(_ context.Context, repoID int64) (*types.LinkedRepo, error) {
	if s.findErr != nil {
		return nil, s.findErr
	}
	return s.linked[repoID], nil
}

func (s *sv5LinkedStore) List(context.Context, int) ([]types.LinkedRepo, error) {
	return s.list, s.listErr
}

func (s *sv5LinkedStore) UpdateOptLock(
	_ context.Context, l *types.LinkedRepo, fn func(*types.LinkedRepo) error,
) (*types.LinkedRepo, error) {
	if s.updateErr != nil {
		return nil, s.updateErr
	}
	cp := *l
	if err := fn(&cp); err != nil {
		return nil, err
	}
	s.updated = append(s.updated, cp.RepoID)
	return &cp, nil
}

func sv5LinkData(t *testing.T) string {
	t.Helper()
	raw, err := json.Marshal(JobLinkRepoInput{RepoID: 5, IsPublic: true})
	require.NoError(t, err)
	return base64.StdEncoding.EncodeToString(raw)
}

func sv5NewLinkJob(t *testing.T) (*JobRepositoryLink, *sv5ImportEnv, *sv5Connector, *sv5LinkedStore) {
	t.Helper()
	env := sv5NewImportEnv(t)
	conn := &sv5Connector{info: AccessInfo{Username: "u", Password: "p", URL: "https://1.1.1.1/o/r.git"}}
	linked := &sv5LinkedStore{linked: map[int64]*types.LinkedRepo{
		5: {RepoID: 5, ConnectorPath: "space", ConnectorIdentifier: "conn", ConnectorRepo: "o/r"},
	}}
	job := NewJobRepositoryLink(
		nil, sv5URLProvider{}, env.git, conn, env.repos, linked, sv5RepoFinder(),
		env.sse, env.indexer, sv5Reporter(t),
	)
	return job, env, conn, linked
}

func TestSv5JobRepositoryLinkHandle(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		job, env, conn, _ := sv5NewLinkJob(t)
		res, err := job.Handle(ctx, sv5LinkData(t), nil)
		require.NoError(t, err)
		require.Empty(t, res)

		require.Equal(t, ConnectorDef{Path: "space", Identifier: "conn", RepoIdentifier: "o/r"}, conn.got[0])
		require.Equal(t, enum.RepoStateActive, env.repos.repo.State)
		require.Equal(t, "git-uid-1", env.repos.repo.GitUID)
		require.Equal(t, "https://u:p@1.1.1.1/o/r.git", env.git.syncs[0].Source)
		require.Equal(t, 1, env.indexer.calls)
		require.Equal(t, []enum.SSEType{enum.SSETypeRepositoryImportCompleted}, env.sse.published)
	})

	t.Run("index failure is tolerated", func(t *testing.T) {
		job, env, _, _ := sv5NewLinkJob(t)
		env.indexer.err = errSv5Boom
		_, err := job.Handle(ctx, sv5LinkData(t), nil)
		require.NoError(t, err)
	})

	tests := []struct {
		name   string
		mutate func(j *JobRepositoryLink, env *sv5ImportEnv, conn *sv5Connector, linked *sv5LinkedStore)
		data   string
		msg    string
	}{
		{"bad base64", nil, "!!!", "base64"},
		{"bad json", nil, base64.StdEncoding.EncodeToString([]byte("{")), "unmarshal"},
		{"repo find", func(_ *JobRepositoryLink, env *sv5ImportEnv, _ *sv5Connector, _ *sv5LinkedStore) {
			env.repos.findErr = errSv5Boom
		}, "", "find repo by id"},
		{"linked find", func(_ *JobRepositoryLink, _ *sv5ImportEnv, _ *sv5Connector, l *sv5LinkedStore) {
			l.findErr = errSv5Boom
		}, "", "find linked repo"},
		{"connector", func(_ *JobRepositoryLink, _ *sv5ImportEnv, c *sv5Connector, _ *sv5LinkedStore) {
			c.err = errSv5Boom
		}, "", "access info from connector"},
		{"bad url", func(_ *JobRepositoryLink, _ *sv5ImportEnv, c *sv5Connector, _ *sv5LinkedStore) {
			c.info.URL = "://bad"
		}, "", "parse git clone URL"},
		{"not importing", func(_ *JobRepositoryLink, env *sv5ImportEnv, _ *sv5Connector, _ *sv5LinkedStore) {
			env.repos.repo.State = enum.RepoStateActive
		}, "", "is not being imported"},
		{"create git repo", func(_ *JobRepositoryLink, env *sv5ImportEnv, _ *sv5Connector, _ *sv5LinkedStore) {
			env.git.createErr = errSv5Boom
		}, "", "create empty git repository"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			job, env, conn, linked := sv5NewLinkJob(t)
			if tc.mutate != nil {
				tc.mutate(job, env, conn, linked)
			}
			data := tc.data
			if data == "" {
				data = sv5LinkData(t)
			}
			_, err := job.Handle(ctx, data, nil)
			require.ErrorContains(t, err, tc.msg)
		})
	}

	t.Run("sync failure cleans up", func(t *testing.T) {
		job, env, _, _ := sv5NewLinkJob(t)
		env.git.syncErr = errSv5Boom
		_, err := job.Handle(ctx, sv5LinkData(t), nil)
		require.ErrorIs(t, err, errSv5Boom)
		require.ErrorContains(t, err, "failed to import repository")
		require.Equal(t, 1, env.git.deletes)
		require.Empty(t, env.sse.published)
	})

	t.Run("cleanup failure is tolerated", func(t *testing.T) {
		job, env, _, _ := sv5NewLinkJob(t)
		env.git.syncErr = errSv5Boom
		env.git.deleteErr = errSv5Boom
		_, err := job.Handle(ctx, sv5LinkData(t), nil)
		require.ErrorIs(t, err, errSv5Boom)
	})
}

func sv5NewRefSyncJob(t *testing.T) (*JobReferenceSync, *sv5ImportEnv) {
	t.Helper()
	env := sv5NewImportEnv(t)
	env.repos.repo.GitUID = "target-git"
	finder := sv5RepoFinder(
		&types.RepositoryCore{ID: 1, Path: "space/src", GitUID: "src-git", DefaultBranch: "srcmain"},
		&types.RepositoryCore{ID: 2, Path: "space/dst", GitUID: "dst-git", DefaultBranch: "dstmain"},
	)
	return &JobReferenceSync{
		defaultBranch: "main",
		urlProvider:   sv5URLProvider{},
		git:           env.git,
		repoStore:     env.repos,
		repoFinder:    finder,
		indexer:       env.indexer,
		eventReporter: sv5Reporter(t),
	}, env
}

func TestSv5JobReferenceSyncHandle(t *testing.T) {
	ctx := context.Background()
	data := func(typ RefSpecType) string {
		raw, err := json.Marshal(ReferenceSyncInput{
			SourceRepoID: 1, TargetRepoID: 2, RefSpecType: typ,
			SourceRef: "refs/heads/a", TargetRef: "refs/heads/dstmain",
		})
		require.NoError(t, err)
		return string(raw)
	}

	tests := []struct {
		typ       RefSpecType
		refSpecs  []string
		wantIndex int
	}{
		{RefSpecTypeReference, []string{"refs/heads/a:refs/heads/dstmain"}, 1},
		{RefSpecTypeDefaultBranch, []string{"refs/heads/srcmain:refs/heads/dstmain"}, 1},
		{RefSpecTypeAllBranches, []string{"refs/heads/*:refs/heads/*"}, 1},
		{RefSpecTypeBranchesAndTags, []string{"refs/heads/*:refs/heads/*", "refs/tags/*:refs/tags/*"}, 1},
	}
	for _, tc := range tests {
		t.Run(string(tc.typ), func(t *testing.T) {
			job, env := sv5NewRefSyncJob(t)
			res, err := job.Handle(ctx, data(tc.typ), nil)
			require.NoError(t, err)
			require.Empty(t, res)
			require.Len(t, env.git.syncs, 1)
			require.Equal(t, tc.refSpecs, env.git.syncs[0].RefSpecs)
			require.Equal(t, "src-git", env.git.syncs[0].Source)
			require.Equal(t, "dst-git", env.git.syncs[0].RepoUID)
			require.Equal(t, tc.wantIndex, env.indexer.calls)
			require.Equal(t, enum.RepoStateActive, env.repos.repo.State)
		})
	}

	t.Run("non default target ref is not indexed", func(t *testing.T) {
		job, env := sv5NewRefSyncJob(t)
		raw, err := json.Marshal(ReferenceSyncInput{
			SourceRepoID: 1, TargetRepoID: 2, RefSpecType: RefSpecTypeReference,
			SourceRef: "refs/heads/a", TargetRef: "refs/heads/other",
		})
		require.NoError(t, err)
		_, err = job.Handle(ctx, string(raw), nil)
		require.NoError(t, err)
		require.Zero(t, env.indexer.calls)
	})

	errTests := map[string]func(j *JobReferenceSync, env *sv5ImportEnv) string{
		"bad json": func(*JobReferenceSync, *sv5ImportEnv) string { return "{" },
		"unknown source": func(*JobReferenceSync, *sv5ImportEnv) string {
			raw, _ := json.Marshal(ReferenceSyncInput{SourceRepoID: 99, TargetRepoID: 2})
			return string(raw)
		},
		"unknown target": func(*JobReferenceSync, *sv5ImportEnv) string {
			raw, _ := json.Marshal(ReferenceSyncInput{SourceRepoID: 1, TargetRepoID: 99})
			return string(raw)
		},
		"sync error": func(_ *JobReferenceSync, env *sv5ImportEnv) string {
			env.git.syncErr = errSv5Boom
			return data(RefSpecTypeAllBranches)
		},
		"repo find error": func(_ *JobReferenceSync, env *sv5ImportEnv) string {
			env.repos.findErr = errSv5Boom
			return data(RefSpecTypeAllBranches)
		},
	}
	for name, setup := range errTests {
		t.Run("error "+name, func(t *testing.T) {
			job, env := sv5NewRefSyncJob(t)
			_, err := job.Handle(ctx, setup(job, env), nil)
			require.Error(t, err)
		})
	}
}

func TestSv5JobSyncLinkedRepositoriesHandle(t *testing.T) {
	ctx := context.Background()

	newJob := func(t *testing.T) (*JobSyncLinkedRepositories, *sv5Git, *sv5Connector, *sv5LinkedStore) {
		t.Helper()
		sv5Bootstrap(t)
		g := &sv5Git{}
		conn := &sv5Connector{info: AccessInfo{Username: "u", Password: "p", URL: "https://1.1.1.1/o/r.git"}}
		linked := &sv5LinkedStore{list: []types.LinkedRepo{
			{RepoID: 1, ConnectorIdentifier: "c1"},
			{RepoID: 99, ConnectorIdentifier: "missing-repo"},
			{RepoID: 2, ConnectorIdentifier: "c2"},
		}}
		finder := sv5RepoFinder(
			&types.RepositoryCore{ID: 1, GitUID: "git-1"},
			&types.RepositoryCore{ID: 2, GitUID: "git-2"},
		)
		return NewJobSyncLinkedRepositories("main", sv5URLProvider{}, g, finder, linked, nil, &sv5Indexer{}, conn),
			g, conn, linked
	}

	t.Run("syncs all resolvable repositories", func(t *testing.T) {
		job, g, _, linked := newJob(t)
		var progress []int
		res, err := job.Handle(ctx, "", func(p int, _ string) error {
			progress = append(progress, p)
			return nil
		})
		require.NoError(t, err)
		require.Empty(t, res)
		require.Len(t, g.syncs, 2)
		require.Equal(t, "git-1", g.syncs[0].RepoUID)
		require.Equal(t, "https://u:p@1.1.1.1/o/r.git", g.syncs[0].Source)
		require.Equal(t, []int64{1, 2}, linked.updated)
		require.Equal(t, []int{0, 66}, progress)
	})

	t.Run("tolerated per-repo failures", func(t *testing.T) {
		job, g, conn, linked := newJob(t)
		linked.updateErr = errSv5Boom
		_, err := job.Handle(ctx, "", func(int, string) error { return errSv5Boom })
		require.NoError(t, err)
		require.Len(t, g.syncs, 2)

		conn.err = errSv5Boom
		g.syncs = nil
		_, err = job.Handle(ctx, "", func(int, string) error { return nil })
		require.NoError(t, err)
		require.Empty(t, g.syncs)

		conn.err = nil
		conn.info.URL = "://bad"
		_, err = job.Handle(ctx, "", func(int, string) error { return nil })
		require.NoError(t, err)
		require.Empty(t, g.syncs)
	})

	t.Run("list error", func(t *testing.T) {
		job, _, _, linked := newJob(t)
		linked.listErr = errSv5Boom
		_, err := job.Handle(ctx, "", nil)
		require.ErrorIs(t, err, errSv5Boom)
	})

	t.Run("sync error aborts", func(t *testing.T) {
		job, g, _, _ := newJob(t)
		g.syncErr = errSv5Boom
		_, err := job.Handle(ctx, "", nil)
		require.ErrorContains(t, err, "failed to sync repository")
	})
}
