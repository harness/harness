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

package branch

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	gitevents "github.com/harness/gitness/app/events/git"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/git/sha"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/stream"
	"github.com/harness/gitness/types"

	"github.com/rs/zerolog/log"
)

const (
	eventsReaderGroupName = "gitness:branch"
	refsBranchPrefix      = "refs/heads/"
)

type Config struct {
	EventReaderName string
	Concurrency     int
	MaxRetries      int
}

// Prepare validates the configuration.
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

type Service struct {
	branchStore store.BranchStore
}

func New(
	ctx context.Context,
	config Config,
	branchStore store.BranchStore,
	gitReaderFactory *events.ReaderFactory[*gitevents.Reader],
) (*Service, error) {
	if err := config.Prepare(); err != nil {
		return nil, fmt.Errorf("provided branch service config is invalid: %w", err)
	}
	log.Ctx(ctx).Info().Msgf("[branch service] event reader name: %s, concurrency: %d, maxRetries: %d",
		config.EventReaderName, config.Concurrency, config.MaxRetries)

	service := &Service{
		branchStore: branchStore,
	}

	_, err := gitReaderFactory.Launch(ctx, eventsReaderGroupName, config.EventReaderName,
		func(r *gitevents.Reader) error {
			const idleTimeout = 10 * time.Second
			r.Configure(
				stream.WithConcurrency(config.Concurrency),
				stream.WithHandlerOptions(
					stream.WithIdleTimeout(idleTimeout),
					stream.WithMaxRetries(config.MaxRetries),
				))

			_ = r.RegisterBranchCreated(service.handleEventBranchCreated)
			_ = r.RegisterBranchUpdated(service.handleEventBranchUpdated)
			_ = r.RegisterBranchDeleted(service.handleEventBranchDeleted)

			return nil
		})
	if err != nil {
		return nil, fmt.Errorf("failed to launch git events reader: %w", err)
	}

	return service, nil
}

func ExtractBranchName(ref string) string {
	return strings.TrimPrefix(ref, refsBranchPrefix)
}

func (s *Service) handleEventBranchCreated(
	ctx context.Context,
	event *events.Event[*gitevents.BranchCreatedPayload],
) error {
	branchName := ExtractBranchName(event.Payload.Ref)

	branchSHA := sha.Must(event.Payload.SHA)

	now := time.Now().UnixMilli()
	branch := &types.BranchTable{
		Name:      branchName,
		SHA:       branchSHA,
		CreatedBy: event.Payload.PrincipalID,
		Created:   now,
		UpdatedBy: event.Payload.PrincipalID,
		Updated:   now,
	}

	err := s.branchStore.Upsert(ctx, event.Payload.RepoID, branch)
	if err != nil {
		return fmt.Errorf("failed to create branch in database: %w", err)
	}

	return nil
}

// handleEventBranchUpdated handles the branch updated event.
func (s *Service) handleEventBranchUpdated(
	ctx context.Context,
	event *events.Event[*gitevents.BranchUpdatedPayload],
) error {
	branchName := ExtractBranchName(event.Payload.Ref)

	branchSHA := sha.Must(event.Payload.NewSHA)

	now := time.Now().UnixMilli()
	branch := &types.BranchTable{
		Name:      branchName,
		SHA:       branchSHA,
		CreatedBy: event.Payload.PrincipalID,
		Created:   now,
		UpdatedBy: event.Payload.PrincipalID,
		Updated:   now,
	}

	if err := s.branchStore.Upsert(ctx, event.Payload.RepoID, branch); err != nil {
		return fmt.Errorf("failed to upsert branch in database: %w", err)
	}
	return nil
}

// handleEventBranchDeleted handles the branch deleted event.
func (s *Service) handleEventBranchDeleted(
	ctx context.Context,
	event *events.Event[*gitevents.BranchDeletedPayload],
) error {
	branchName := ExtractBranchName(event.Payload.Ref)

	err := s.branchStore.Delete(
		ctx,
		event.Payload.RepoID,
		branchName,
	)
	if err != nil && !errors.Is(err, gitness_store.ErrResourceNotFound) {
		return fmt.Errorf("failed to delete branch from database: %w", err)
	}

	return nil
}
