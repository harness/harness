// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package logs

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/harness/gitness/internal/api/controller/logs"
	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"

	"github.com/rs/zerolog/log"
)

var (
	pingInterval = 30 * time.Second
	tailMaxTime  = 1 * time.Hour
)

func HandleTail(logCtrl *logs.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)
		pipelineUID, err := request.GetPipelineUIDFromPath(r)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}
		executionNum, err := request.GetExecutionNumberFromPath(r)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}
		stageNum, err := request.GetStageNumberFromPath(r)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}
		stepNum, err := request.GetStepNumberFromPath(r)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}
		repoRef, err := request.GetRepoRefFromPath(r)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		f, ok := w.(http.Flusher)
		if !ok {
			log.Error().Msg("http writer type assertion failed")
			render.InternalError(w)
			return
		}

		h := w.Header()
		h.Set("Content-Type", "text/event-stream")
		h.Set("Cache-Control", "no-cache")
		h.Set("Connection", "keep-alive")
		h.Set("X-Accel-Buffering", "no")
		h.Set("Access-Control-Allow-Origin", "*")

		io.WriteString(w, ": ping\n\n")
		f.Flush()

		linec, errc, err := logCtrl.Tail(
			ctx, session, repoRef, pipelineUID,
			executionNum, int(stageNum), int(stepNum))
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		// could not get error channel
		if errc == nil {
			io.WriteString(w, "event: error\ndata: eof\n\n")
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), tailMaxTime)
		defer cancel()

		enc := json.NewEncoder(w)

		pingTimer := time.NewTimer(pingInterval)
		defer pingTimer.Stop()
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
				break L
			case err := <-errc:
				log.Err(err).Msg("received error in the tail channel")
				break L
			case <-pingTimer.C:
				// if time b/w messages takes longer, send a ping
				io.WriteString(w, ": ping\n\n")
				f.Flush()
			case line := <-linec:
				io.WriteString(w, "data: ")
				enc.Encode(line)
				io.WriteString(w, "\n\n")
				f.Flush()
			}
		}

		io.WriteString(w, "event: error\ndata: eof\n\n")
		f.Flush()
	}
}
