package pipeline

import (
	"encoding/json"
	"net/http"

	"github.com/harness/gitness/internal/api/controller/pipeline"
	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
)

/*
 * Updates an existing pipeline.
 */
func HandleUpdate(pipelineCtrl *pipeline.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)

		in := new(pipeline.UpdateInput)
		err := json.NewDecoder(r.Body).Decode(in)
		if err != nil {
			render.BadRequestf(w, "Invalid Request Body: %s.", err)
			return
		}

		pipelineRef, err := request.GetPipelinePathRefFromPath(r)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}
		spaceRef, pipelineUID, err := SplitRef(pipelineRef)
		if err != nil {
			render.TranslatedUserError(w, err)
		}

		pipeline, err := pipelineCtrl.Update(ctx, session, spaceRef, pipelineUID, in)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		render.JSON(w, http.StatusOK, pipeline)
	}
}
