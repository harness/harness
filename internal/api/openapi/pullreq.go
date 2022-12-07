// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package openapi

import (
	"net/http"

	"github.com/harness/gitness/internal/api/controller/pullreq"
	handler "github.com/harness/gitness/internal/api/handler/pullreq"
	"github.com/harness/gitness/internal/api/request"
	"github.com/harness/gitness/internal/api/usererror"
	"github.com/harness/gitness/types/enum"

	"github.com/gotidy/ptr"
	"github.com/swaggest/openapi-go/openapi3"
)

type createPullReqRequest struct {
	repoRequest
	pullreq.CreateInput
}

type listPullReqRequest struct {
	repoRequest
}

type pullReqRequest struct {
	repoRequest
	ID int64 `path:"pullreqNumber"`
}

type getPullReqRequest struct {
	pullReqRequest
}

var queryParameterQueryPullRequest = openapi3.ParameterOrRef{
	Parameter: &openapi3.Parameter{
		Name:        request.QueryParamQuery,
		In:          openapi3.ParameterInQuery,
		Description: ptr.String("The substring by which the pull requests are filtered."),
		Required:    ptr.Bool(false),
		Schema: &openapi3.SchemaOrRef{
			Schema: &openapi3.Schema{
				Type: ptrSchemaType(openapi3.SchemaTypeString),
			},
		},
	},
}

var queryParameterCreatedByPullRequest = openapi3.ParameterOrRef{
	Parameter: &openapi3.Parameter{
		Name:        request.QueryParamCreatedBy,
		In:          openapi3.ParameterInQuery,
		Description: ptr.String("The principal ID who created pull requests."),
		Required:    ptr.Bool(false),
		Schema: &openapi3.SchemaOrRef{
			Schema: &openapi3.Schema{
				Type: ptrSchemaType(openapi3.SchemaTypeInteger),
			},
		},
	},
}

var queryParameterStatePullRequest = openapi3.ParameterOrRef{
	Parameter: &openapi3.Parameter{
		Name:        request.QueryParamState,
		In:          openapi3.ParameterInQuery,
		Description: ptr.String("The state of the pull requests to include in the result."),
		Required:    ptr.Bool(false),
		Schema: &openapi3.SchemaOrRef{
			Schema: &openapi3.Schema{
				Type: ptrSchemaType(openapi3.SchemaTypeArray),
				Items: &openapi3.SchemaOrRef{
					Schema: &openapi3.Schema{
						Type:    ptrSchemaType(openapi3.SchemaTypeString),
						Default: ptrptr(string(enum.PullReqStateOpen)),
						Enum: []interface{}{
							ptr.String(string(enum.PullReqStateOpen)),
							ptr.String(string(enum.PullReqStateClosed)),
							ptr.String(string(enum.PullReqStateMerged)),
							ptr.String(string(enum.PullReqStateRejected)),
						},
					},
				},
			},
		},
	},
}

var queryParameterSortPullRequest = openapi3.ParameterOrRef{
	Parameter: &openapi3.Parameter{
		Name:        request.QueryParamSort,
		In:          openapi3.ParameterInQuery,
		Description: ptr.String("The data by which the pull requests are sorted."),
		Required:    ptr.Bool(false),
		Schema: &openapi3.SchemaOrRef{
			Schema: &openapi3.Schema{
				Type:    ptrSchemaType(openapi3.SchemaTypeString),
				Default: ptrptr(enum.PullReqSortNumber.String()),
				Enum: []interface{}{
					ptr.String(enum.PullReqSortNumber.String()),
					ptr.String(enum.PullReqSortCreated.String()),
					ptr.String(enum.PullReqSortUpdated.String()),
				},
			},
		},
	},
}

func pullReqOperations(reflector *openapi3.Reflector) {
	createPullReq := openapi3.Operation{}
	createPullReq.WithTags("pullreq")
	createPullReq.WithMapOfAnything(map[string]interface{}{"operationId": "createPullReq"})
	_ = reflector.SetRequest(&createPullReq, new(createPullReqRequest), http.MethodPost)
	_ = reflector.SetJSONResponse(&createPullReq, new(handler.PullReq), http.StatusCreated)
	_ = reflector.SetJSONResponse(&createPullReq, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&createPullReq, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&createPullReq, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&createPullReq, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodPost, "/repos/{repoRef}/pullreq", createPullReq)

	listPullReq := openapi3.Operation{}
	listPullReq.WithTags("pullreq")
	listPullReq.WithMapOfAnything(map[string]interface{}{"operationId": "listPullReq"})
	listPullReq.WithParameters(
		queryParameterStatePullRequest,
		queryParameterQueryPullRequest, queryParameterCreatedByPullRequest,
		queryParameterDirection, queryParameterSortPullRequest,
		queryParameterPage, queryParameterPerPage)
	_ = reflector.SetRequest(&listPullReq, new(listPullReqRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&listPullReq, new([]handler.PullReq), http.StatusOK)
	_ = reflector.SetJSONResponse(&listPullReq, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&listPullReq, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&listPullReq, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&listPullReq, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/repos/{repoRef}/pullreq", listPullReq)

	getPullReq := openapi3.Operation{}
	getPullReq.WithTags("pullreq")
	getPullReq.WithMapOfAnything(map[string]interface{}{"operationId": "getPullReq"})
	_ = reflector.SetRequest(&getPullReq, new(getPullReqRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&getPullReq, new(handler.PullReq), http.StatusOK)
	_ = reflector.SetJSONResponse(&getPullReq, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&getPullReq, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&getPullReq, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&getPullReq, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/repos/{repoRef}/pullreq/{pullreqNumber}", getPullReq)
}
