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

package webhook

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	gitevents "github.com/harness/gitness/app/events/git"
	pullreqevents "github.com/harness/gitness/app/events/pullreq"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/encrypt"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/stream"
)

const (
	eventsReaderGroupName = "gitness:webhook"
)

type Config struct {
	// UserAgentIdentity specifies the identity used for the user agent header
	// IMPORTANT: do not include version.
	UserAgentIdentity string
	// HeaderIdentity specifies the identity used for headers in webhook calls (e.g. X-Gitness-Trigger, ...).
	// NOTE: If no value is provided, the UserAgentIdentity will be used.
	HeaderIdentity      string
	EventReaderName     string
	Concurrency         int
	MaxRetries          int
	AllowPrivateNetwork bool
	AllowLoopback       bool
}

func (c *Config) Prepare() error {
	if c == nil {
		return errors.New("config is required")
	}
	if c.EventReaderName == "" {
		return errors.New("config.EventReaderName is required")
	}
	if c.UserAgentIdentity == "" {
		return errors.New("config.UserAgentIdentity is required")
	}
	if c.Concurrency < 1 {
		return errors.New("config.Concurrency has to be a positive number")
	}
	if c.MaxRetries < 0 {
		return errors.New("config.MaxRetries can't be negative")
	}

	// Backfill data
	if c.HeaderIdentity == "" {
		c.HeaderIdentity = c.UserAgentIdentity
	}

	return nil
}

// Service is responsible for processing webhook events.
type Service struct {
	webhookStore          store.WebhookStore
	webhookExecutionStore store.WebhookExecutionStore
	urlProvider           url.Provider
	repoStore             store.RepoStore
	pullreqStore          store.PullReqStore
	principalStore        store.PrincipalStore
	git                   git.Interface
	activityStore         store.PullReqActivityStore
	encrypter             encrypt.Encrypter

	secureHTTPClient   *http.Client
	insecureHTTPClient *http.Client

	secureHTTPClientInternal   *http.Client
	insecureHTTPClientInternal *http.Client

	config Config
}

func NewService(
	ctx context.Context,
	config Config,
	gitReaderFactory *events.ReaderFactory[*gitevents.Reader],
	prReaderFactory *events.ReaderFactory[*pullreqevents.Reader],
	webhookStore store.WebhookStore,
	webhookExecutionStore store.WebhookExecutionStore,
	repoStore store.RepoStore,
	pullreqStore store.PullReqStore,
	activityStore store.PullReqActivityStore,
	urlProvider url.Provider,
	principalStore store.PrincipalStore,
	git git.Interface,
	encrypter encrypt.Encrypter,
) (*Service, error) {
	if err := config.Prepare(); err != nil {
		return nil, fmt.Errorf("provided webhook service config is invalid: %w", err)
	}
	service := &Service{
		webhookStore:          webhookStore,
		webhookExecutionStore: webhookExecutionStore,
		repoStore:             repoStore,
		pullreqStore:          pullreqStore,
		activityStore:         activityStore,
		urlProvider:           urlProvider,
		principalStore:        principalStore,
		git:                   git,
		encrypter:             encrypter,

		secureHTTPClient:   newHTTPClient(config.AllowLoopback, config.AllowPrivateNetwork, false),
		insecureHTTPClient: newHTTPClient(config.AllowLoopback, config.AllowPrivateNetwork, true),

		secureHTTPClientInternal:   newHTTPClient(config.AllowLoopback, true, false),
		insecureHTTPClientInternal: newHTTPClient(config.AllowLoopback, true, true),

		config: config,
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
			_ = r.RegisterClosed(service.handleEventPullReqClosed)
			_ = r.RegisterCommentCreated(service.handleEventPullReqComment)
			_ = r.RegisterMerged(service.handleEventPullReqMerged)
			_ = r.RegisterUpdated(service.handleEventPullReqUpdated)

			return nil
		})
	if err != nil {
		return nil, fmt.Errorf("failed to launch pr event reader for webhooks: %w", err)
	}

	return service, nil
}
