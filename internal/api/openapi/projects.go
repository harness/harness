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
	projectRequest struct {
		Project string `path:"project"`

		// include base request
		baseRequest
	}

	projectListRequest struct {
		// include pagination request
		paginationRequest

		// include base request
		baseRequest
	}

	projectResponse struct {
		Data   types.Project `json:"data"`
		Status string        `json:"status" enum:"SUCCESS,FAILURE,ERROR"`
	}

	projectListResponse struct {
		Data   types.ProjectList `json:"data"`
		Status string            `json:"status" enum:"SUCCESS,FAILURE,ERROR"`
	}
)

// helper function that constructs the openapi specification
// for project resources.
func buildProjects(reflector *openapi3.Reflector) {

	opFind := openapi3.Operation{}
	opFind.WithTags("projects")
	opFind.WithMapOfAnything(map[string]interface{}{"operationId": "getProject"})
	reflector.SetRequest(&opFind, new(projectRequest), http.MethodGet)
	reflector.SetJSONResponse(&opFind, new(projectResponse), http.StatusOK)
	reflector.SetJSONResponse(&opFind, new(render.Error), http.StatusInternalServerError)
	reflector.Spec.AddOperation(http.MethodGet, "/api/projects/{project}", opFind)

	opList := openapi3.Operation{}
	opList.WithTags("projects", "user")
	opList.WithMapOfAnything(map[string]interface{}{"operationId": "listProjects"})
	reflector.SetRequest(&opList, new(projectListRequest), http.MethodGet)
	reflector.SetJSONResponse(&opList, new(projectListResponse), http.StatusOK)
	reflector.SetJSONResponse(&opList, new(render.Error), http.StatusInternalServerError)
	reflector.Spec.AddOperation(http.MethodGet, "/api/user/projects", opList)

}
