// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package space

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/internal/writer"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

var (
	pingInterval = 30 * time.Second
	tailMaxTime  = 2 * time.Hour
)

func (c *Controller) Events(
	ctx context.Context,
	session *auth.Session,
	spaceRef string,
	w writer.WriterFlusher,
) error {
	space, err := c.spaceStore.FindByRef(ctx, spaceRef)
	if err != nil {
		return fmt.Errorf("failed to find space ref: %w", err)
	}

	if err = apiauth.CheckSpace(ctx, c.authorizer, session, space, enum.PermissionSpaceView, true); err != nil {
		return fmt.Errorf("failed to authorize stream: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, tailMaxTime)
	defer cancel()

	io.WriteString(w, ": ping\n\n")
	w.Flush()

	events, errc, consumer := c.eventsStream.Subscribe(ctx, space.ID)
	defer c.eventsStream.Unsubscribe(ctx, consumer)
	// could not get error channel
	if errc == nil {
		io.WriteString(w, "event: error\ndata: eof\n\n")
		w.Flush()
		return fmt.Errorf("could not get error channel")
	}
	pingTimer := time.NewTimer(pingInterval)
	defer pingTimer.Stop()

	enc := json.NewEncoder(w)
L:
	for {
		// ensure timer is stopped before resetting (see documentation)
		if !pingTimer.Stop() {
			// in this specific case the timer's channel could be both, empty or full
			select {
			case <-pingTimer.C:
			default:
			}
		}
		pingTimer.Reset(pingInterval)
		select {
		case <-ctx.Done():
			log.Debug().Msg("events: stream cancelled")
			break L
		case err := <-errc:
			log.Err(err).Msg("events: received error in the tail channel")
			break L
		case <-pingTimer.C:
			// if time b/w messages takes longer, send a ping
			io.WriteString(w, ": ping\n\n")
			w.Flush()
		case event := <-events:
			io.WriteString(w, "data: ")
			enc.Encode(event)
			io.WriteString(w, "\n\n")
			w.Flush()
		}
	}

	io.WriteString(w, "event: error\ndata: eof\n\n")
	w.Flush()
	log.Debug().Msg("events: stream closed")
	return nil
}
