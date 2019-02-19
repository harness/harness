// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

package events

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/drone/drone/handler/api/request"
	"github.com/drone/drone/logger"
	"github.com/drone/drone/core"
)

// HandleGlobal creates an http.HandlerFunc that streams builds events
// to the http.Response in an event stream format.
func HandleGlobal(
	repos core.RepositoryStore,
	events core.Pubsub,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := logger.FromRequest(r)

		h := w.Header()
		h.Set("Content-Type", "text/event-stream")
		h.Set("Cache-Control", "no-cache")
		h.Set("Connection", "keep-alive")
		h.Set("X-Accel-Buffering", "no")

		f, ok := w.(http.Flusher)
		if !ok {
			return
		}

		access := map[string]struct{}{}
		user, authenticated := request.UserFrom(r.Context())
		if authenticated {
			list, _ := repos.List(r.Context(), user.ID)
			for _, repo := range list {
				access[repo.Slug] = struct{}{}
			}
		}

		io.WriteString(w, ": ping\n\n")
		f.Flush()

		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()

		events, errc := events.Subscribe(ctx)
		logger.Debugln("events: stream opened")

	L:
		for {
			select {
			case <-ctx.Done():
				logger.Debugln("events: stream cancelled")
				break L
			case <-errc:
				logger.Debugln("events: stream eror")
				break L
			case <-time.After(time.Hour):
				logger.Debugln("events: stream timeout")
				break L
			case <-time.After(pingInterval):
				io.WriteString(w, ": ping\n\n")
				f.Flush()
			case event := <-events:
				_, authorized := access[event.Repository]
				if event.Visibility == core.VisibilityPublic {
					authorized = true
				}
				if event.Visibility == core.VisibilityInternal && authenticated {
					authorized = true
				}
				if authorized {
					io.WriteString(w, "data: ")
					w.Write(event.Data)
					io.WriteString(w, "\n\n")
					f.Flush()
				}
			}
		}

		io.WriteString(w, "event: error\ndata: eof\n\n")
		f.Flush()

		logger.Debugln("events: stream closed")
	}
}
