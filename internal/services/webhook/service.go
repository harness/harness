// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package webhook

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/harness/gitness/events"
	"github.com/harness/gitness/gitrpc"
	gitevents "github.com/harness/gitness/internal/events/git"
	pullreqevents "github.com/harness/gitness/internal/events/pullreq"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/internal/url"
	"github.com/harness/gitness/stream"
)

const (
	eventsReaderGroupName = "gitness:webhook"
)

type Config struct {
	EventReaderName     string `envconfig:"GITNESS_WEBHOOK_EVENT_READER_NAME"`
	Concurrency         int    `envconfig:"GITNESS_WEBHOOK_CONCURRENCY" default:"4"`
	MaxRetries          int    `envconfig:"GITNESS_WEBHOOK_MAX_RETRIES" default:"3"`
	AllowPrivateNetwork bool   `envconfig:"GITNESS_WEBHOOK_ALLOW_PRIVATE_NETWORK" default:"false"`
	AllowLoopback       bool   `envconfig:"GITNESS_WEBHOOK_ALLOW_LOOPBACK" default:"false"`
}

func (c *Config) Validate() error {
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

// Service is responsible for processing webhook events.
type Service struct {
	webhookStore          store.WebhookStore
	webhookExecutionStore store.WebhookExecutionStore
	urlProvider           *url.Provider
	repoStore             store.RepoStore
	pullreqStore          store.PullReqStore
	principalStore        store.PrincipalStore
	gitRPCClient          gitrpc.Interface

	secureHTTPClient   *http.Client
	insecureHTTPClient *http.Client
}

func NewService(ctx context.Context, config Config,
	gitReaderFactory *events.ReaderFactory[*gitevents.Reader],
	prReaderFactory *events.ReaderFactory[*pullreqevents.Reader],
	webhookStore store.WebhookStore, webhookExecutionStore store.WebhookExecutionStore,
	repoStore store.RepoStore, pullreqStore store.PullReqStore, urlProvider *url.Provider,
	principalStore store.PrincipalStore, gitRPCClient gitrpc.Interface) (*Service, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("provided config is invalid: %w", err)
	}
	service := &Service{
		webhookStore:          webhookStore,
		webhookExecutionStore: webhookExecutionStore,
		repoStore:             repoStore,
		pullreqStore:          pullreqStore,
		urlProvider:           urlProvider,
		principalStore:        principalStore,
		gitRPCClient:          gitRPCClient,

		secureHTTPClient:   newHTTPClient(config.AllowLoopback, config.AllowPrivateNetwork, false),
		insecureHTTPClient: newHTTPClient(config.AllowLoopback, config.AllowPrivateNetwork, true),
	}

	_, err := gitReaderFactory.Launch(ctx, eventsReaderGroupName, config.EventReaderName,
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
			_ = r.RegisterBranchDeleted(service.handleEventBranchDeleted)

			_ = r.RegisterTagCreated(service.handleEventTagCreated)
			_ = r.RegisterTagUpdated(service.handleEventTagUpdated)
			_ = r.RegisterTagDeleted(service.handleEventTagDeleted)

			return nil
		})
	if err != nil {
		return nil, fmt.Errorf("failed to launch git event reader for webhooks: %w", err)
	}

	_, err = prReaderFactory.Launch(ctx, eventsReaderGroupName, config.EventReaderName,
		func(r *pullreqevents.Reader) error {
			const idleTimeout = 1 * time.Minute
			r.Configure(
				stream.WithConcurrency(config.Concurrency),
				stream.WithHandlerOptions(
					stream.WithIdleTimeout(idleTimeout),
					stream.WithMaxRetries(config.MaxRetries),
				))

			// register events
			_ = r.RegisterCreated(service.handleEventPullReqCreated)
			_ = r.RegisterReopened(service.handleEventPullReqReopened)
			_ = r.RegisterBranchUpdated(service.handleEventPullReqBranchUpdated)

			return nil
		})
	if err != nil {
		return nil, fmt.Errorf("failed to launch pr event reader for webhooks: %w", err)
	}

	return service, nil
}
