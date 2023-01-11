// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package webhook

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/harness/gitness/events"
	"github.com/harness/gitness/gitrpc"
	gitevents "github.com/harness/gitness/internal/events/git"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/internal/url"
)

const (
	eventsReaderGroupName = "webhook"
	processingTimeout     = 2 * time.Minute
)

type Config struct {
	EventReaderName     string `json:"event_reader_name"`
	Concurrency         int    `json:"concurrency"`
	MaxRetryCount       int64  `json:"max_retry_count"`
	AllowPrivateNetwork bool   `json:"allow_private_network"`
	AllowLoopback       bool   `json:"allow_loopback"`
}

// Server is responsible for processing webhook events.
type Server struct {
	webhookStore          store.WebhookStore
	webhookExecutionStore store.WebhookExecutionStore
	urlProvider           *url.Provider
	repoStore             store.RepoStore
	principalStore        store.PrincipalStore
	gitRPCClient          gitrpc.Interface

	readerCanceler     *events.ReaderCanceler
	secureHTTPClient   *http.Client
	insecureHTTPClient *http.Client
}

func NewServer(ctx context.Context, config Config,
	gitReaderFactory *events.ReaderFactory[*gitevents.Reader],
	webhookStore store.WebhookStore, webhookExecutionStore store.WebhookExecutionStore,
	repoStore store.RepoStore, urlProvider *url.Provider,
	principalStore store.PrincipalStore, gitRPCClient gitrpc.Interface) (*Server, error) {
	server := &Server{
		webhookStore:          webhookStore,
		webhookExecutionStore: webhookExecutionStore,
		repoStore:             repoStore,
		urlProvider:           urlProvider,
		principalStore:        principalStore,
		gitRPCClient:          gitRPCClient,

		// set after launching factory
		readerCanceler: nil,

		secureHTTPClient:   newHTTPClient(config.AllowLoopback, config.AllowPrivateNetwork, false),
		insecureHTTPClient: newHTTPClient(config.AllowLoopback, config.AllowPrivateNetwork, true),
	}
	canceler, err := gitReaderFactory.Launch(ctx, eventsReaderGroupName, config.EventReaderName,
		func(r *gitevents.Reader) error {
			// configure reader
			_ = r.SetConcurrency(config.Concurrency)
			_ = r.SetMaxRetryCount(config.MaxRetryCount)
			_ = r.SetProcessingTimeout(processingTimeout)

			// register events
			_ = r.RegisterBranchCreated(server.handleEventBranchCreated)
			_ = r.RegisterBranchUpdated(server.handleEventBranchUpdated)
			_ = r.RegisterBranchDeleted(server.handleEventBranchDeleted)

			return nil
		})
	if err != nil {
		return nil, fmt.Errorf("failed to launch event reader for webhooks: %w", err)
	}
	server.readerCanceler = canceler

	return server, nil
}

func (s *Server) Cancel() error {
	return s.readerCanceler.Cancel()
}
