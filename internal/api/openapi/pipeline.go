// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package openapi

import (
	"net/http"

	"github.com/harness/scm/internal/api/render"
	"github.com/harness/scm/types"

	"github.com/swaggest/openapi-go/openapi3"
)

type (
	// request to find or delete a pipeline.
	pipelineRequest struct {
		Param string `path:"pipeline"`

		// include account parameters
		baseRequest
	}

	pipelineListRequest struct {
		// include account parameters
		baseRequest

		// include pagination parameters
		paginationRequest
	}

	// request to update a pipeline.
	pipelineUpdateRequest struct {
		Param string `path:"pipeline"`

		// include request body input.
		types.PipelineInput

		// include account parameters
		baseRequest
	}

	// request to create a pipeline.
	pipelineCreateRequest struct {
		// include account parameters
		baseRequest

		// include request body input.
		types.PipelineInput
	}
)

// helper function that constructs the openapi specification
// for pipeline resources.
func buildPipeline(reflector *openapi3.Reflector) {

	opFind := openapi3.Operation{}
	opFind.WithTags("pipeline")
	opFind.WithMapOfAnything(map[string]interface{}{"operationId": "getPipeline"})
	reflector.SetRequest(&opFind, new(pipelineRequest), http.MethodGet)
	reflector.SetJSONResponse(&opFind, new(types.Pipeline), http.StatusOK)
	reflector.SetJSONResponse(&opFind, new(render.Error), http.StatusInternalServerError)
	reflector.SetJSONResponse(&opFind, new(render.Error), http.StatusNotFound)
	reflector.Spec.AddOperation(http.MethodGet, "/pipelines/{pipeline}", opFind)

	onList := openapi3.Operation{}
	onList.WithTags("pipeline")
	onList.WithMapOfAnything(map[string]interface{}{"operationId": "listPipelines"})
	reflector.SetRequest(&onList, new(pipelineListRequest), http.MethodGet)
	reflector.SetJSONResponse(&onList, new([]*types.Pipeline), http.StatusOK)
	reflector.SetJSONResponse(&onList, new(render.Error), http.StatusInternalServerError)
	reflector.Spec.AddOperation(http.MethodGet, "/pipelines", onList)

	opCreate := openapi3.Operation{}
	opCreate.WithTags("pipeline")
	opCreate.WithMapOfAnything(map[string]interface{}{"operationId": "createPipeline"})
	reflector.SetRequest(&opCreate, new(pipelineCreateRequest), http.MethodPost)
	reflector.SetJSONResponse(&opCreate, new(types.Pipeline), http.StatusOK)
	reflector.SetJSONResponse(&opCreate, new(render.Error), http.StatusBadRequest)
	reflector.SetJSONResponse(&opCreate, new(render.Error), http.StatusInternalServerError)
	reflector.SetJSONResponse(&opCreate, new(render.Error), http.StatusNotFound)
	reflector.Spec.AddOperation(http.MethodPost, "/pipelines", opCreate)

	opUpdate := openapi3.Operation{}
	opUpdate.WithTags("pipeline")
	opUpdate.WithMapOfAnything(map[string]interface{}{"operationId": "updatePipeline"})
	reflector.SetRequest(&opUpdate, new(pipelineUpdateRequest), http.MethodPatch)
	reflector.SetJSONResponse(&opUpdate, new(types.Pipeline), http.StatusOK)
	reflector.SetJSONResponse(&opUpdate, new(render.Error), http.StatusBadRequest)
	reflector.SetJSONResponse(&opUpdate, new(render.Error), http.StatusInternalServerError)
	reflector.SetJSONResponse(&opUpdate, new(render.Error), http.StatusNotFound)
	reflector.Spec.AddOperation(http.MethodPatch, "/pipelines/{pipeline}", opUpdate)

	opDelete := openapi3.Operation{}
	opDelete.WithTags("pipeline")
	opDelete.WithMapOfAnything(map[string]interface{}{"operationId": "deletePipeline"})
	reflector.SetRequest(&opDelete, new(pipelineRequest), http.MethodDelete)
	reflector.SetJSONResponse(&opDelete, nil, http.StatusNoContent)
	reflector.SetJSONResponse(&opDelete, new(render.Error), http.StatusInternalServerError)
	reflector.SetJSONResponse(&opDelete, new(render.Error), http.StatusNotFound)
	reflector.Spec.AddOperation(http.MethodDelete, "/pipelines/{pipeline}", opDelete)
}
