// Copyright 2019 Drone IO, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package stages

import (
	"net/http"
	"strconv"

	"github.com/drone/drone/core"
	"github.com/drone/drone/handler/api/render"
	"github.com/drone/drone/trigger/dag"

	"github.com/go-chi/chi"
)

// HandleRestart returns an http.HandlerFunc that processes http
// requests to restart a single stage of a build (re-queue the stage
// and re-run its steps without creating a new build).
//
// Build status is not updated here. When the restarted stage completes,
// the operator teardown runs, isBuildComplete(stages) becomes true, and
// build status is recomputed from all stages (see operator/manager/teardown.go),
// so the build can turn green once the restarted stage passes.
func HandleRestart(
	repos core.RepositoryStore,
	builds core.BuildStore,
	stages core.StageStore,
	steps core.StepStore,
	sched core.Scheduler,
	logs core.LogStore,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var (
			namespace = chi.URLParam(r, "owner")
			name      = chi.URLParam(r, "name")
		)
		number, err := strconv.ParseInt(chi.URLParam(r, "number"), 10, 64)
		if err != nil {
			render.BadRequest(w, err)
			return
		}
		repo, err := repos.FindName(r.Context(), namespace, name)
		if err != nil {
			render.NotFound(w, err)
			return
		}
		prev, err := builds.FindNumber(r.Context(), repo.ID, number)
		if err != nil {
			render.NotFound(w, err)
			return
		}

		switch prev.Status {
		case core.StatusBlocked:
			render.BadRequestf(w, "cannot start a blocked build")
			return
		case core.StatusDeclined:
			render.BadRequestf(w, "cannot start a declined build")
			return
		}

		stageNumber, err := strconv.Atoi(chi.URLParam(r, "stage"))
		if err != nil {
			render.BadRequest(w, err)
			return
		}
		stage, err := stages.FindNumber(r.Context(), prev.ID, stageNumber)
		if err != nil {
			render.NotFound(w, err)
			return
		}
		if !isRestartableStageStatus(stage.Status) {
			render.BadRequestf(w, "cannot restart a pipeline with status %q; only completed pipelines (success, failure, error, killed) can be restarted", stage.Status)
			return
		}

		stepList, err := steps.List(r.Context(), stage.ID)
		if err != nil {
			render.InternalError(w, err)
			return
		}
		for _, step := range stepList {
			step.Status = core.StatusPending
			step.Error = ""
			step.ExitCode = 0
			step.Started = 0
			step.Stopped = 0
			err = steps.Update(r.Context(), step)
			if err != nil {
				render.InternalError(w, err)
				return
			}
			_ = logs.Delete(r.Context(), step.ID)
		}

		stage.Status = core.StatusPending
		stage.Machine = ""
		stage.Error = ""
		stage.ExitCode = 0
		stage.Started = 0
		stage.Stopped = 0
		err = stages.Update(r.Context(), stage)
		if err != nil {
			render.InternalError(w, err)
			return
		}
		err = sched.Schedule(noContext, stage)
		if err != nil {
			render.InternalError(w, err)
			return
		}

		// Reset all stages downstream of the restarted stage to Waiting so that
		// when this stage completes, teardown's scheduleDownstream will schedule
		// them (same outcome as full retry: A->B->C all run in order).
		allStages, err := stages.List(r.Context(), prev.ID)
		if err != nil {
			render.InternalError(w, err)
			return
		}
		d := dag.New()
		for _, s := range allStages {
			d.Add(s.Name, s.DependsOn...)
		}
		downstreamNames := make(map[string]struct{})
		for _, n := range d.Descendants(stage.Name) {
			downstreamNames[n] = struct{}{}
		}
		for _, s := range allStages {
			if _, isDownstream := downstreamNames[s.Name]; isDownstream && s.Number != stage.Number {
				s.Status = core.StatusWaiting
				s.Machine = ""
				s.Error = ""
				s.ExitCode = 0
				s.Started = 0
				s.Stopped = 0
				err = stages.Update(r.Context(), s)
				if err != nil {
					render.InternalError(w, err)
					return
				}
			}
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// isRestartableStageStatus returns true if the stage has a terminal status
// that can be restarted (re-run). Pending, running, waiting, blocked, and
// declined stages cannot be restarted via this endpoint.
func isRestartableStageStatus(status string) bool {
	switch status {
	case core.StatusPassing, core.StatusFailing, core.StatusKilled, core.StatusError, core.StatusSkipped:
		return true
	default:
		return false
	}
}
