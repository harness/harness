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

//nolint:cyclop
package logs

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/harness/gitness/app/api/controller/logs"
	"github.com/harness/gitness/app/api/render"
	"github.com/harness/gitness/app/api/request"

	"github.com/rs/zerolog/log"
)

var (
	pingInterval = 30 * time.Second
	tailMaxTime  = 1 * time.Hour
)

// TODO: Move to controller and do error handling (see space events)
//
//nolint:gocognit,errcheck,cyclop // refactor if needed.
func HandleTail(logCtrl *logs.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)
		pipelineIdentifier, err := request.GetPipelineIdentifierFromPath(r)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}
		executionNum, err := request.GetExecutionNumberFromPath(r)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}
		stageNum, err := request.GetStageNumberFromPath(r)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}
		stepNum, err := request.GetStepNumberFromPath(r)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}
		repoRef, err := request.GetRepoRefFromPath(r)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		f, ok := w.(http.Flusher)
		if !ok {
			log.Error().Msg("http writer type assertion failed")
			render.InternalError(ctx, w)
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
			ctx, session, repoRef, pipelineIdentifier,
			executionNum, int(stageNum), int(stepNum))
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
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
