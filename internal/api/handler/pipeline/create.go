package pipeline

import (
	"encoding/json"
	"net/http"

	"github.com/harness/gitness/internal/api/controller/pipeline"
	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
)

// HandleCreate returns a http.HandlerFunc that creates a new pipelinesitory.
func HandleCreate(pipelineCtrl *pipeline.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)

		in := new(pipeline.CreateInput)
		err := json.NewDecoder(r.Body).Decode(in)
		if err != nil {
			render.BadRequestf(w, "Invalid Request Body: %s.", err)
			return
		}

		pipeline, err := pipelineCtrl.Create(ctx, session, in)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		render.JSON(w, http.StatusCreated, pipeline)
	}
}
