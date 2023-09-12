// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package space

import (
	"net/http"

	"github.com/harness/gitness/internal/api/controller/space"
	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
	"github.com/harness/gitness/internal/writer"

	"github.com/rs/zerolog/log"
)

// HandleEventsStream returns an http.HandlerFunc that watches for
// events on a space
func HandleEventsStream(spaceCtrl *space.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)

		spaceRef, err := request.GetSpaceRefFromPath(r)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		h := w.Header()
		h.Set("Content-Type", "text/event-stream")
		h.Set("Cache-Control", "no-cache")
		h.Set("Connection", "keep-alive")
		h.Set("X-Accel-Buffering", "no")
		h.Set("Access-Control-Allow-Origin", "*")

		f, ok := w.(http.Flusher)
		if !ok {
			log.Error().Msg("http writer type assertion failed")
			render.InternalError(w)
			return
		}

		writer := writer.NewWriterFlusher(w, f)

		err = spaceCtrl.Events(ctx, session, spaceRef, writer)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}
	}
}
