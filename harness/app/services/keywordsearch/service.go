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

package keywordsearch

import (
	"context"
	"errors"
	"fmt"
	"time"

	gitevents "github.com/harness/gitness/app/events/git"
	repoevents "github.com/harness/gitness/app/events/repo"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/stream"
)

const groupGitEvents = "gitness:keywordsearch"

type Config struct {
	EventReaderName string
	Concurrency     int
	MaxRetries      int
}

func (c *Config) Prepare() error {
	if c == nil {
		return errors.New("config is required")
	}
	if c.EventReaderName == "" {
		return errors.New("config.EventReaderName is required")
	}
	if c.Concurrency < 1 {
		return errors.New("config.Concurrency has to be a positive number")
	}
	if c.MaxRetries < 0 {
		return errors.New("config.MaxRetries can't be negative")
	}
	return nil
}

// Service is responsible for indexing of repository for keyword search.
type Service struct {
	config    Config
	indexer   Indexer
	repoStore store.RepoStore
}

func NewService(
	ctx context.Context,
	config Config,
	gitReaderFactory *events.ReaderFactory[*gitevents.Reader],
	repoReaderFactory *events.ReaderFactory[*repoevents.Reader],
	repoStore store.RepoStore,
	indexer Indexer,
) (*Service, error) {
	if err := config.Prepare(); err != nil {
		return nil, fmt.Errorf("provided codesearch service config is invalid: %w", err)
	}
	service := &Service{
		config:    config,
		repoStore: repoStore,
		indexer:   indexer,
	}

	_, err := gitReaderFactory.Launch(ctx, groupGitEvents, config.EventReaderName,
		func(r *gitevents.Reader) error {
			const idleTimeout = 1 * time.Minute
			r.Configure(
				stream.WithConcurrency(config.Concurrency),
				stream.WithHandlerOptions(
					stream.WithIdleTimeout(idleTimeout),
					stream.WithMaxRetries(config.MaxRetries),
				))

			// register events
			_ = r.RegisterBranchCreated(service.handleEventBranchCreated)
			_ = r.RegisterBranchUpdated(service.handleEventBranchUpdated)

			return nil
		})
	if err != nil {
		return nil, fmt.Errorf("failed to launch git event reader for webhooks: %w", err)
	}

	_, err = repoReaderFactory.Launch(ctx, groupGitEvents, config.EventReaderName,
		func(r *repoevents.Reader) error {
			const idleTimeout = 1 * time.Minute
			r.Configure(
				stream.WithConcurrency(config.Concurrency),
				stream.WithHandlerOptions(
					stream.WithIdleTimeout(idleTimeout),
					stream.WithMaxRetries(config.MaxRetries),
				))

			_ = r.RegisterDefaultBranchUpdated((service.handleUpdateDefaultBranch))
			return nil
		})
	if err != nil {
		return nil, fmt.Errorf("failed to launch reader factory for repo git group: %w", err)
	}

	return service, nil
}
