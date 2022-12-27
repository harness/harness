// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package openapi

import (
	"net/http"

	"github.com/harness/gitness/internal/api/controller/pullreq"
	"github.com/harness/gitness/internal/api/request"
	"github.com/harness/gitness/internal/api/usererror"
	"github.com/harness/gitness/types"
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
	ID int64 `path:"pullreq_number"`
}

type getPullReqRequest struct {
	pullReqRequest
}

type updatePullReqRequest struct {
	pullReqRequest
	pullreq.UpdateInput
}

type listPullReqActivitiesRequest struct {
	pullReqRequest
}

type commentPullReqRequest struct {
	pullReqRequest
	pullreq.CommentCreateInput
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

var queryParameterSourceRepoRefPullRequest = openapi3.ParameterOrRef{
	Parameter: &openapi3.Parameter{
		Name:        "source_repo_ref",
		In:          openapi3.ParameterInQuery,
		Description: ptr.String("Source repository ref of the pull requests."),
		Required:    ptr.Bool(false),
		Schema: &openapi3.SchemaOrRef{
			Schema: &openapi3.Schema{
				Type: ptrSchemaType(openapi3.SchemaTypeString),
			},
		},
	},
}

var queryParameterSourceBranchPullRequest = openapi3.ParameterOrRef{
	Parameter: &openapi3.Parameter{
		Name:        "source_branch",
		In:          openapi3.ParameterInQuery,
		Description: ptr.String("Source branch of the pull requests."),
		Required:    ptr.Bool(false),
		Schema: &openapi3.SchemaOrRef{
			Schema: &openapi3.Schema{
				Type: ptrSchemaType(openapi3.SchemaTypeString),
			},
		},
	},
}

var queryParameterTargetBranchPullRequest = openapi3.ParameterOrRef{
	Parameter: &openapi3.Parameter{
		Name:        "target_branch",
		In:          openapi3.ParameterInQuery,
		Description: ptr.String("Target branch of the pull requests."),
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

var queryParameterKindPullRequestActivity = openapi3.ParameterOrRef{
	Parameter: &openapi3.Parameter{
		Name:        request.QueryParamKind,
		In:          openapi3.ParameterInQuery,
		Description: ptr.String("The kind of the pull request activity to include in the result."),
		Required:    ptr.Bool(false),
		Schema: &openapi3.SchemaOrRef{
			Schema: &openapi3.Schema{
				Type: ptrSchemaType(openapi3.SchemaTypeArray),
				Items: &openapi3.SchemaOrRef{
					Schema: &openapi3.Schema{
						Type: ptrSchemaType(openapi3.SchemaTypeString),
						Enum: []interface{}{
							ptr.String(string(enum.PullReqActivityKindSystem)),
							ptr.String(string(enum.PullReqActivityKindComment)),
							ptr.String(string(enum.PullReqActivityKindCodeComment)),
						},
					},
				},
			},
		},
	},
}

var queryParameterTypePullRequestActivity = openapi3.ParameterOrRef{
	Parameter: &openapi3.Parameter{
		Name:        request.QueryParamType,
		In:          openapi3.ParameterInQuery,
		Description: ptr.String("The type of the pull request activity to include in the result."),
		Required:    ptr.Bool(false),
		Schema: &openapi3.SchemaOrRef{
			Schema: &openapi3.Schema{
				Type: ptrSchemaType(openapi3.SchemaTypeArray),
				Items: &openapi3.SchemaOrRef{
					Schema: &openapi3.Schema{
						Type: ptrSchemaType(openapi3.SchemaTypeString),
						Enum: []interface{}{
							ptr.String(string(enum.PullReqActivityTypeComment)),
							ptr.String(string(enum.PullReqActivityTypeCodeComment)),
							ptr.String(string(enum.PullReqActivityTypeTitleChange)),
						},
					},
				},
			},
		},
	},
}

//nolint:funlen
func pullReqOperations(reflector *openapi3.Reflector) {
	createPullReq := openapi3.Operation{}
	createPullReq.WithTags("pullreq")
	createPullReq.WithMapOfAnything(map[string]interface{}{"operationId": "createPullReq"})
	_ = reflector.SetRequest(&createPullReq, new(createPullReqRequest), http.MethodPost)
	_ = reflector.SetJSONResponse(&createPullReq, new(types.PullReq), http.StatusCreated)
	_ = reflector.SetJSONResponse(&createPullReq, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&createPullReq, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&createPullReq, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&createPullReq, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodPost, "/repos/{repoRef}/pullreq", createPullReq)

	listPullReq := openapi3.Operation{}
	listPullReq.WithTags("pullreq")
	listPullReq.WithMapOfAnything(map[string]interface{}{"operationId": "listPullReq"})
	listPullReq.WithParameters(
		queryParameterStatePullRequest, queryParameterSourceRepoRefPullRequest,
		queryParameterSourceBranchPullRequest, queryParameterTargetBranchPullRequest,
		queryParameterQueryPullRequest, queryParameterCreatedByPullRequest,
		queryParameterDirection, queryParameterSortPullRequest,
		queryParameterPage, queryParameterPerPage)
	_ = reflector.SetRequest(&listPullReq, new(listPullReqRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&listPullReq, new([]types.PullReq), http.StatusOK)
	_ = reflector.SetJSONResponse(&listPullReq, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&listPullReq, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&listPullReq, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&listPullReq, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/repos/{repoRef}/pullreq", listPullReq)

	getPullReq := openapi3.Operation{}
	getPullReq.WithTags("pullreq")
	getPullReq.WithMapOfAnything(map[string]interface{}{"operationId": "getPullReq"})
	_ = reflector.SetRequest(&getPullReq, new(getPullReqRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&getPullReq, new(types.PullReq), http.StatusOK)
	_ = reflector.SetJSONResponse(&getPullReq, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&getPullReq, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&getPullReq, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&getPullReq, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/repos/{repoRef}/pullreq/{pullreq_number}", getPullReq)

	putPullReq := openapi3.Operation{}
	putPullReq.WithTags("pullreq")
	putPullReq.WithMapOfAnything(map[string]interface{}{"operationId": "updatePullReq"})
	_ = reflector.SetRequest(&putPullReq, new(updatePullReqRequest), http.MethodPut)
	_ = reflector.SetJSONResponse(&putPullReq, new(types.PullReq), http.StatusOK)
	_ = reflector.SetJSONResponse(&putPullReq, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&putPullReq, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&putPullReq, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&putPullReq, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodPut, "/repos/{repoRef}/pullreq/{pullreq_number}", putPullReq)

	listPullReqActivities := openapi3.Operation{}
	listPullReqActivities.WithTags("pullreq")
	listPullReqActivities.WithMapOfAnything(map[string]interface{}{"operationId": "listPullReqActivities"})
	listPullReqActivities.WithParameters(
		queryParameterKindPullRequestActivity, queryParameterTypePullRequestActivity,
		queryParameterSince, queryParameterUntil, queryParameterLimit)
	_ = reflector.SetRequest(&listPullReqActivities, new(listPullReqActivitiesRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&listPullReqActivities, new([]types.PullReqActivity), http.StatusOK)
	_ = reflector.SetJSONResponse(&listPullReqActivities, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&listPullReqActivities, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&listPullReqActivities, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&listPullReqActivities, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodGet,
		"/repos/{repoRef}/pullreq/{pullreq_number}/activities", listPullReqActivities)

	commentPullReq := openapi3.Operation{}
	commentPullReq.WithTags("pullreq")
	commentPullReq.WithMapOfAnything(map[string]interface{}{"operationId": "commentPullReq"})
	_ = reflector.SetRequest(&commentPullReq, new(commentPullReqRequest), http.MethodPost)
	_ = reflector.SetJSONResponse(&commentPullReq, new(types.PullReqActivity), http.StatusOK)
	_ = reflector.SetJSONResponse(&commentPullReq, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&commentPullReq, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&commentPullReq, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&commentPullReq, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodPost,
		"/repos/{repoRef}/pullreq/{pullreq_number}/comment", commentPullReq)
}
