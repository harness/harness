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

package render

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/sse"

	"github.com/rs/zerolog/log"
)

func StreamSSE(
	ctx context.Context,
	w http.ResponseWriter,
	chStop <-chan struct{},
	chEvents <-chan *sse.Event,
	chErr <-chan error,
) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		UserError(ctx, w, usererror.ErrResponseNotFlushable)
		log.Ctx(ctx).Warn().Err(usererror.ErrResponseNotFlushable).Msg("failed to build SSE stream")
		return
	}

	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("Connection", "keep-alive")
	h.Set("X-Accel-Buffering", "no")
	h.Set("Access-Control-Allow-Origin", "*")

	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)

	stream := sseStream{
		enc:     enc,
		writer:  w,
		flusher: flusher,
	}

	const (
		pingInterval = 30 * time.Second
		tailMaxTime  = 2 * time.Hour
	)

	ctx, ctxCancel := context.WithTimeout(ctx, tailMaxTime)
	defer ctxCancel()

	if err := stream.ping(); err != nil {
		log.Ctx(ctx).Err(err).Msg("failed to send initial ping")
		return
	}

	defer func() {
		if err := stream.close(); err != nil {
			log.Ctx(ctx).Err(err).Msg("failed to close SSE stream")
		}
	}()

	pingTimer := time.NewTimer(pingInterval)
	defer pingTimer.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Ctx(ctx).Debug().Err(ctx.Err()).Msg("stream SSE request context done")
			return

		case <-chStop:
			log.Ctx(ctx).Debug().Msg("app shutdown")
			return

		case err := <-chErr:
			log.Ctx(ctx).Debug().Err(err).Msg("received error from SSE stream")
			return

		case <-pingTimer.C:
			if err := stream.ping(); err != nil {
				log.Ctx(ctx).Err(err).Msg("failed to send SSE ping")
				return
			}

		case event, canProduce := <-chEvents:
			if !canProduce {
				log.Ctx(ctx).Debug().Msg("events channel is drained and closed.")
				return
			}
			if err := stream.event(event); err != nil {
				log.Ctx(ctx).Err(err).Msgf("failed to send SSE event: %s", event.Type)
				return
			}
		}

		pingTimer.Stop() // stop timer

		select {
		case <-pingTimer.C: // drain channel
		default:
		}

		pingTimer.Reset(pingInterval) // reset timer
	}
}

type sseStream struct {
	enc     *json.Encoder
	writer  io.Writer
	flusher http.Flusher
}

func (r sseStream) event(event *sse.Event) error {
	_, err := io.WriteString(r.writer, fmt.Sprintf("event: %s\n", event.Type))
	if err != nil {
		return fmt.Errorf("failed to send event header: %w", err)
	}

	_, err = io.WriteString(r.writer, "data: ")
	if err != nil {
		return fmt.Errorf("failed to send data header: %w", err)
	}

	err = r.enc.Encode(event.Data)
	if err != nil {
		return fmt.Errorf("failed to send data: %w", err)
	}

	// NOTE: enc.Encode is ending the data with a new line, only add one more
	// Source: https://cs.opensource.google/go/go/+/refs/tags/go1.21.1:src/encoding/json/stream.go;l=220
	_, err = r.writer.Write([]byte{'\n'})
	if err != nil {
		return fmt.Errorf("failed to send end of message: %w", err)
	}

	r.flusher.Flush()
	return nil
}

func (r sseStream) close() error {
	_, err := io.WriteString(r.writer, "event: error\ndata: eof\n\n")
	if err != nil {
		return fmt.Errorf("failed to send EOF: %w", err)
	}
	r.flusher.Flush()
	return nil
}

func (r sseStream) ping() error {
	_, err := io.WriteString(r.writer, ": ping\n\n")
	if err != nil {
		return fmt.Errorf("failed to send ping: %w", err)
	}
	r.flusher.Flush()
	return nil
}
