// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package webhook

import (
	"context"
	"fmt"
	"time"

	"github.com/harness/gitness/events"
	gitevents "github.com/harness/gitness/gitrpc/events"
)

const (
	eventsReaderGroupName = "webhook"
	processingTimeout     = 2 * time.Minute
)

// Server is responsible for processing webhook events.
type Server struct {
	readerCanceler *events.ReaderCanceler
}

func NewServer(ctx context.Context, gitReaderFactory *events.ReaderFactory[*gitevents.Reader],
	eventsReaderName string, concurrency int) (*Server, error) {
	canceler, err := gitReaderFactory.Launch(ctx, eventsReaderGroupName, eventsReaderName,
		func(r *gitevents.Reader) error {
			// configure reader
			_ = r.SetConcurrency(concurrency)
			_ = r.SetProcessingTimeout(processingTimeout)

			// register events
			_ = r.RegisterBranchCreated(branchCreated)
			_ = r.RegisterBranchDeleted(branchDeleted)

			return nil
		})
	if err != nil {
		return nil, fmt.Errorf("failed to launch event reader for webhooks: %w", err)
	}

	return &Server{
		readerCanceler: canceler,
	}, nil
}

func (p *Server) Cancel() error {
	return p.readerCanceler.Cancel()
}
