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
	"errors"

	"github.com/harness/gitness/app/store"
	gitnessurl "github.com/harness/gitness/app/url"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/job"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
)

var errSvaBoom = errors.New("boom")

// svaURLProvider returns a fixed internal API URL.
type svaURLProvider struct {
	gitnessurl.Provider
}

func (svaURLProvider) GetInternalAPIURL(context.Context) string { return "http://internal.test" }

// svaGit is a git.Interface fake driven by function fields. Unset functions panic when called.
type svaGit struct {
	git.Interface

	createRepo func(params *git.CreateRepositoryParams) (*git.CreateRepositoryOutput, error)
	syncRepo   func(params *git.SyncRepositoryParams) (*git.SyncRepositoryOutput, error)
	deleteRepo func(params *git.DeleteRepositoryParams) error
	matchFiles func(params *git.MatchFilesParams) (*git.MatchFilesOutput, error)
	commit     func(params *git.CommitFilesParams) (git.CommitFilesResponse, error)
}

func (g *svaGit) CreateRepository(
	_ context.Context, params *git.CreateRepositoryParams,
) (*git.CreateRepositoryOutput, error) {
	return g.createRepo(params)
}

func (g *svaGit) SyncRepository(
	_ context.Context, params *git.SyncRepositoryParams,
) (*git.SyncRepositoryOutput, error) {
	return g.syncRepo(params)
}

func (g *svaGit) DeleteRepository(_ context.Context, params *git.DeleteRepositoryParams) error {
	return g.deleteRepo(params)
}

func (g *svaGit) MatchFiles(_ context.Context, params *git.MatchFilesParams) (*git.MatchFilesOutput, error) {
	return g.matchFiles(params)
}

func (g *svaGit) CommitFiles(_ context.Context, params *git.CommitFilesParams) (git.CommitFilesResponse, error) {
	return g.commit(params)
}

// svaTx runs the transaction function directly, optionally failing instead.
type svaTx struct {
	err error
}

func (t svaTx) WithTx(ctx context.Context, fn func(ctx context.Context) error, _ ...any) error {
	if t.err != nil {
		return t.err
	}
	return fn(ctx)
}

type svaPipelineStore struct {
	store.PipelineStore

	created []*types.Pipeline
	err     error
}

func (s *svaPipelineStore) Create(_ context.Context, pipeline *types.Pipeline) error {
	if s.err != nil {
		return s.err
	}
	pipeline.ID = int64(len(s.created) + 1)
	s.created = append(s.created, pipeline)
	return nil
}

type svaTriggerStore struct {
	store.TriggerStore

	created []*types.Trigger
	err     error
}

func (s *svaTriggerStore) Create(_ context.Context, trigger *types.Trigger) error {
	if s.err != nil {
		return s.err
	}
	s.created = append(s.created, trigger)
	return nil
}

// svaJobStore is an in-memory job.Store keeping only what the tests need.
type svaJobStore struct {
	job.Store

	created  []*job.Job
	upserted []*job.Job
	jobs     map[string]*job.Job
	err      error
}

func (s *svaJobStore) Create(_ context.Context, j *job.Job) error {
	if s.err != nil {
		return s.err
	}
	s.created = append(s.created, j)
	return nil
}

func (s *svaJobStore) Upsert(_ context.Context, j *job.Job) error {
	if s.err != nil {
		return s.err
	}
	s.upserted = append(s.upserted, j)
	return nil
}

func (s *svaJobStore) Find(_ context.Context, uid string) (*job.Job, error) {
	if s.err != nil {
		return nil, s.err
	}
	j, ok := s.jobs[uid]
	if !ok {
		return nil, gitness_store.ErrResourceNotFound
	}
	return j, nil
}

// svaEncrypter reverses the plaintext. It is deterministic and trivially invertible.
type svaEncrypter struct {
	encryptErr error
	decryptErr error
}

func svaReverse(in []byte) []byte {
	out := make([]byte, len(in))
	for i, b := range in {
		out[len(in)-1-i] = b
	}
	return out
}

func (e svaEncrypter) Encrypt(plaintext string) ([]byte, error) {
	if e.encryptErr != nil {
		return nil, e.encryptErr
	}
	return svaReverse([]byte(plaintext)), nil
}

func (e svaEncrypter) Decrypt(ciphertext []byte) (string, error) {
	if e.decryptErr != nil {
		return "", e.decryptErr
	}
	return string(svaReverse(ciphertext)), nil
}
