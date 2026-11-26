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

package repo

import (
	"context"
	"testing"
	"time"

	checkevents "github.com/harness/gitness/app/events/check"
	gitevents "github.com/harness/gitness/app/events/git"
	mergequeueevents "github.com/harness/gitness/app/events/mergequeue"
	"github.com/harness/gitness/app/services/importer"
	"github.com/harness/gitness/app/services/locker"
	"github.com/harness/gitness/app/services/mergequeue"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/job"
	"github.com/harness/gitness/lock"
	gitnessstore "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"

	"github.com/stretchr/testify/require"
)

// crpJobStore is a job.Store accepting recurring job registrations and serving one preconfigured job.
type crpJobStore struct {
	job.Store
	found     *job.Job
	findErr   error
	createErr error
}

func (s crpJobStore) Create(context.Context, *job.Job) error { return s.createErr }

func (crpJobStore) Upsert(context.Context, *job.Job) error { return nil }

func (s crpJobStore) Find(context.Context, string) (*job.Job, error) {
	if s.findErr != nil {
		return nil, s.findErr
	}

	if s.found == nil {
		return nil, gitnessstore.ErrResourceNotFound
	}

	return s.found, nil
}

// withLinkJobs wires the linked repository job handler backed by the provided job store.
func (e *crpEnv) withLinkJobs(t *testing.T, jobs crpJobStore) {
	t.Helper()

	executor := job.NewExecutor(jobs, nil)
	scheduler, err := job.NewScheduler(jobs, executor, nil, nil, "test", 1, time.Hour)
	require.NoError(t, err)

	e.c.importLinked = importer.NewJobRepositoryLink(
		scheduler, e.c.urlProvider, e.git, e.c.connectorService, e.c.repoStore, e.c.linkedRepoStore,
		e.c.repoFinder, nil, e.c.indexer, e.c.eventReporter,
	)
}

// crpEncrypter is an encrypt.Encrypter that doesn't encrypt.
type crpEncrypter struct{}

func (crpEncrypter) Encrypt(plaintext string) ([]byte, error) { return []byte(plaintext), nil }

func (crpEncrypter) Decrypt(ciphertext []byte) (string, error) { return string(ciphertext), nil }

// withImportJobs wires a repository import job handler backed by the provided job store.
func (e *crpEnv) withImportJobs(t *testing.T, jobs crpJobStore) {
	t.Helper()
	e.withImporter(t, jobs, nil)
}

// withImporter wires a repository import job handler backed by the provided job store and importer.
func (e *crpEnv) withImporter(t *testing.T, jobs crpJobStore, imp *importer.Importer) {
	t.Helper()

	executor := job.NewExecutor(jobs, nil)
	scheduler, err := job.NewScheduler(
		jobs, executor, lock.NewInMemory(lock.Config{
			App: "test", Namespace: "test", Expiry: time.Minute, Tries: 3, RetryDelay: 10 * time.Millisecond,
		}), nil, "test", 1, time.Hour,
	)
	require.NoError(t, err)

	jobRepository, err := importer.ProvideJobRepositoryImport(crpEncrypter{}, scheduler, executor, imp)
	require.NoError(t, err)

	e.c.importer = jobRepository

	referenceSync, err := importer.ProvideJobReferenceSync(
		&types.Config{}, e.c.urlProvider, e.git, e.c.repoStore, e.c.repoFinder,
		scheduler, executor, e.c.indexer, e.c.eventReporter,
	)
	require.NoError(t, err)

	e.c.referenceSync = referenceSync
}

// crpMQEntryStore is a store.MergeQueueEntryStore reporting a fixed set of branches as queued.
type crpMQEntryStore struct {
	store.MergeQueueEntryStore
	queued map[string]struct{}
	err    error
	// queueErr is returned when looking up the merge queue of a branch (not found by default).
	queueErr error
}

// crpMergeQueueStore is a store.MergeQueueStore without any queue.
type crpMergeQueueStore struct {
	store.MergeQueueStore
	err error
}

func (s crpMergeQueueStore) FindByRepoAndBranch(context.Context, int64, string) (*types.MergeQueue, error) {
	if s.err != nil {
		return nil, s.err
	}

	return nil, gitnessstore.ErrResourceNotFound
}

func (s *crpMQEntryStore) BranchesWithPullReqInQueue(
	_ context.Context, _ int64, branches []string,
) (map[string]struct{}, error) {
	if s.err != nil {
		return nil, s.err
	}

	out := map[string]struct{}{}
	for _, b := range branches {
		if _, ok := s.queued[b]; ok {
			out[b] = struct{}{}
		}
	}

	return out, nil
}

// withMergeQueue wires a real merge queue service backed by in-memory events and the provided entry store.
func (e *crpEnv) withMergeQueue(ctx context.Context, t *testing.T, entries *crpMQEntryStore) {
	t.Helper()

	ctx, cancel := context.WithCancel(ctx)
	t.Cleanup(cancel)

	system, err := events.ProvideSystem(events.Config{
		Mode:            events.ModeInMemory,
		MaxStreamLength: 100,
	}, nil, events.NewNoopCollector())
	require.NoError(t, err)

	gitReporter, err := gitevents.NewReporter(system)
	require.NoError(t, err)

	mqReporter, err := mergequeueevents.NewReporter(system)
	require.NoError(t, err)

	checkFactory, err := checkevents.NewReaderFactory(system)
	require.NoError(t, err)

	mqFactory, err := mergequeueevents.NewReaderFactory(system)
	require.NoError(t, err)

	executor := job.NewExecutor(crpJobStore{}, nil)
	scheduler, err := job.NewScheduler(crpJobStore{}, executor, nil, nil, "test", 1, time.Hour)
	require.NoError(t, err)

	cfg := &types.Config{}
	cfg.MergeQueue.CheckEventsConcurrency = 1
	cfg.MergeQueue.QueueEventsConcurrency = 1

	svc, err := mergequeue.NewService(
		ctx, cfg, e.git, crpTx{}, gitReporter, mqReporter, checkFactory, mqFactory,
		e.c.repoFinder, nil, nil, nil, nil, crpMergeQueueStore{err: entries.queueErr}, entries,
		e.c.protectionManager, nil,
		e.c.urlProvider, locker.NewLocker(lock.NewInMemory(lock.Config{})), scheduler, executor,
	)
	require.NoError(t, err)

	e.c.mergeQueueService = svc
}
