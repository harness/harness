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

package openapi

import (
	"net/http"

	"github.com/harness/gitness/app/api/controller/pullreq"
	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/git"
	gittypes "github.com/harness/gitness/git/api"
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

type statePullReqRequest struct {
	pullReqRequest
	pullreq.StateInput
}

type listPullReqActivitiesRequest struct {
	pullReqRequest
}

type mergePullReq struct {
	pullReqRequest
	pullreq.MergeInput
}

type commentCreatePullReqRequest struct {
	pullReqRequest
	pullreq.CommentCreateInput
}

type commentApplySuggestionstRequest struct {
	pullReqRequest
	pullreq.CommentApplySuggestionsInput
}

type pullReqCommentRequest struct {
	pullReqRequest
	ID int64 `path:"pullreq_comment_id"`
}

type commentUpdatePullReqRequest struct {
	pullReqCommentRequest
	pullreq.CommentUpdateInput
}

type commentDeletePullReqRequest struct {
	pullReqCommentRequest
}

type commentStatusPullReqRequest struct {
	pullReqCommentRequest
	pullreq.CommentStatusInput
}

type reviewerListPullReqRequest struct {
	pullReqRequest
}

type reviewerDeletePullReqRequest struct {
	pullReqRequest
	PullReqReviewerID int64 `path:"pullreq_reviewer_id"`
}

type reviewerAddPullReqRequest struct {
	pullReqRequest
	pullreq.ReviewerAddInput
}

type reviewSubmitPullReqRequest struct {
	pullreq.ReviewSubmitInput
	pullReqRequest
}

type fileViewAddPullReqRequest struct {
	pullReqRequest
	pullreq.FileViewAddInput
}

type fileViewListPullReqRequest struct {
	pullReqRequest
}

type fileViewDeletePullReqRequest struct {
	pullReqRequest
	Path string `path:"file_path"`
}

type getRawPRDiffRequest struct {
	pullReqRequest
	Path []string `query:"path" description:"provide path for diff operation"`
}

type postRawPRDiffRequest struct {
	pullReqRequest
	gittypes.FileDiffRequests
}

type getPullReqChecksRequest struct {
	pullReqRequest
}

type pullReqAssignLabelInput struct {
	pullReqRequest
	types.PullReqCreateInput
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
		Description: ptr.String("List of principal IDs who created pull requests."),
		Required:    ptr.Bool(false),
		Schema: &openapi3.SchemaOrRef{
			Schema: &openapi3.Schema{
				Type: ptrSchemaType(openapi3.SchemaTypeArray),
				Items: &openapi3.SchemaOrRef{
					Schema: &openapi3.Schema{
						Type: ptrSchemaType(openapi3.SchemaTypeInteger),
					},
				},
			},
		},
		// making it look like created_by=1&created_by=2
		Style:   ptr.String(string(openapi3.EncodingStyleForm)),
		Explode: ptr.Bool(true),
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
						Enum:    enum.PullReqState("").Enum(),
					},
				},
			},
		},
		Style:   ptr.String(string(openapi3.EncodingStyleForm)),
		Explode: ptr.Bool(true),
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
				Default: ptrptr(enum.PullReqSortNumber),
				Enum:    enum.PullReqSort("").Enum(),
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
						Enum: enum.PullReqActivityKind("").Enum(),
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
						Enum: enum.PullReqActivityType("").Enum(),
					},
				},
			},
		},
	},
}

var queryParameterBeforePullRequestActivity = openapi3.ParameterOrRef{
	Parameter: &openapi3.Parameter{
		Name:        request.QueryParamBefore,
		In:          openapi3.ParameterInQuery,
		Description: ptr.String("The result should contain only entries created before this timestamp (unix millis)."),
		Required:    ptr.Bool(false),
		Schema: &openapi3.SchemaOrRef{
			Schema: &openapi3.Schema{
				Type:    ptrSchemaType(openapi3.SchemaTypeInteger),
				Minimum: ptr.Float64(0),
			},
		},
	},
}

var queryParameterAssignable = openapi3.ParameterOrRef{
	Parameter: &openapi3.Parameter{
		Name:        request.QueryParamAssignable,
		In:          openapi3.ParameterInQuery,
		Description: ptr.String("The result should contain all labels assignable to the pullreq."),
		Required:    ptr.Bool(false),
		Schema: &openapi3.SchemaOrRef{
			Schema: &openapi3.Schema{
				Type:    ptrSchemaType(openapi3.SchemaTypeBoolean),
				Default: ptrptr(false),
			},
		},
	},
}

var queryParameterLabelID = openapi3.ParameterOrRef{
	Parameter: &openapi3.Parameter{
		Name:        request.QueryParamLabelID,
		In:          openapi3.ParameterInQuery,
		Description: ptr.String("List of label ids used to filter pull requests."),
		Required:    ptr.Bool(false),
		Schema: &openapi3.SchemaOrRef{
			Schema: &openapi3.Schema{
				Type: ptrSchemaType(openapi3.SchemaTypeArray),
				Items: &openapi3.SchemaOrRef{
					Schema: &openapi3.Schema{
						Type: ptrSchemaType(openapi3.SchemaTypeInteger),
					},
				},
			},
		},
		// making it look like label_id=1&label_id=2
		Style:   ptr.String(string(openapi3.EncodingStyleForm)),
		Explode: ptr.Bool(true),
	},
}

var queryParameterValueID = openapi3.ParameterOrRef{
	Parameter: &openapi3.Parameter{
		Name:        request.QueryParamValueID,
		In:          openapi3.ParameterInQuery,
		Description: ptr.String("List of label value ids used to filter pull requests."),
		Required:    ptr.Bool(false),
		Schema: &openapi3.SchemaOrRef{
			Schema: &openapi3.Schema{
				Type: ptrSchemaType(openapi3.SchemaTypeArray),
				Items: &openapi3.SchemaOrRef{
					Schema: &openapi3.Schema{
						Type: ptrSchemaType(openapi3.SchemaTypeInteger),
					},
				},
			},
		},
		// making it look like value_id=1&value_id=2
		Style:   ptr.String(string(openapi3.EncodingStyleForm)),
		Explode: ptr.Bool(true),
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
	_ = reflector.Spec.AddOperation(http.MethodPost, "/repos/{repo_ref}/pullreq", createPullReq)

	listPullReq := openapi3.Operation{}
	listPullReq.WithTags("pullreq")
	listPullReq.WithMapOfAnything(map[string]interface{}{"operationId": "listPullReq"})
	listPullReq.WithParameters(
		queryParameterStatePullRequest, queryParameterSourceRepoRefPullRequest,
		queryParameterSourceBranchPullRequest, queryParameterTargetBranchPullRequest,
		queryParameterQueryPullRequest, queryParameterCreatedByPullRequest,
		queryParameterOrder, queryParameterSortPullRequest,
		queryParameterCreatedLt, queryParameterCreatedGt,
		QueryParameterPage, QueryParameterLimit,
		queryParameterLabelID, queryParameterValueID)
	_ = reflector.SetRequest(&listPullReq, new(listPullReqRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&listPullReq, new([]types.PullReq), http.StatusOK)
	_ = reflector.SetJSONResponse(&listPullReq, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&listPullReq, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&listPullReq, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&listPullReq, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/repos/{repo_ref}/pullreq", listPullReq)

	getPullReq := openapi3.Operation{}
	getPullReq.WithTags("pullreq")
	getPullReq.WithMapOfAnything(map[string]interface{}{"operationId": "getPullReq"})
	_ = reflector.SetRequest(&getPullReq, new(getPullReqRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&getPullReq, new(types.PullReq), http.StatusOK)
	_ = reflector.SetJSONResponse(&getPullReq, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&getPullReq, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&getPullReq, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&getPullReq, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/repos/{repo_ref}/pullreq/{pullreq_number}", getPullReq)

	putPullReq := openapi3.Operation{}
	putPullReq.WithTags("pullreq")
	putPullReq.WithMapOfAnything(map[string]interface{}{"operationId": "updatePullReq"})
	_ = reflector.SetRequest(&putPullReq, new(updatePullReqRequest), http.MethodPatch)
	_ = reflector.SetJSONResponse(&putPullReq, new(types.PullReq), http.StatusOK)
	_ = reflector.SetJSONResponse(&putPullReq, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&putPullReq, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&putPullReq, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&putPullReq, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodPatch, "/repos/{repo_ref}/pullreq/{pullreq_number}", putPullReq)

	statePullReq := openapi3.Operation{}
	statePullReq.WithTags("pullreq")
	statePullReq.WithMapOfAnything(map[string]interface{}{"operationId": "statePullReq"})
	_ = reflector.SetRequest(&statePullReq, new(statePullReqRequest), http.MethodPatch)
	_ = reflector.SetJSONResponse(&statePullReq, new(types.PullReq), http.StatusOK)
	_ = reflector.SetJSONResponse(&statePullReq, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&statePullReq, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&statePullReq, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&statePullReq, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodPost, "/repos/{repo_ref}/pullreq/{pullreq_number}/state", statePullReq)

	listPullReqActivities := openapi3.Operation{}
	listPullReqActivities.WithTags("pullreq")
	listPullReqActivities.WithMapOfAnything(map[string]interface{}{"operationId": "listPullReqActivities"})
	listPullReqActivities.WithParameters(
		queryParameterKindPullRequestActivity, queryParameterTypePullRequestActivity,
		queryParameterAfter, queryParameterBeforePullRequestActivity, QueryParameterLimit)
	_ = reflector.SetRequest(&listPullReqActivities, new(listPullReqActivitiesRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&listPullReqActivities, new([]types.PullReqActivity), http.StatusOK)
	_ = reflector.SetJSONResponse(&listPullReqActivities, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&listPullReqActivities, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&listPullReqActivities, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&listPullReqActivities, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodGet,
		"/repos/{repo_ref}/pullreq/{pullreq_number}/activities", listPullReqActivities)

	commentCreatePullReq := openapi3.Operation{}
	commentCreatePullReq.WithTags("pullreq")
	commentCreatePullReq.WithMapOfAnything(map[string]interface{}{"operationId": "commentCreatePullReq"})
	_ = reflector.SetRequest(&commentCreatePullReq, new(commentCreatePullReqRequest), http.MethodPost)
	_ = reflector.SetJSONResponse(&commentCreatePullReq, new(types.PullReqActivity), http.StatusOK)
	_ = reflector.SetJSONResponse(&commentCreatePullReq, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&commentCreatePullReq, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&commentCreatePullReq, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&commentCreatePullReq, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodPost,
		"/repos/{repo_ref}/pullreq/{pullreq_number}/comments", commentCreatePullReq)

	commentUpdatePullReq := openapi3.Operation{}
	commentUpdatePullReq.WithTags("pullreq")
	commentUpdatePullReq.WithMapOfAnything(map[string]interface{}{"operationId": "commentUpdatePullReq"})
	_ = reflector.SetRequest(&commentUpdatePullReq, new(commentUpdatePullReqRequest), http.MethodPatch)
	_ = reflector.SetJSONResponse(&commentUpdatePullReq, new(types.PullReqActivity), http.StatusOK)
	_ = reflector.SetJSONResponse(&commentUpdatePullReq, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&commentUpdatePullReq, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&commentUpdatePullReq, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&commentUpdatePullReq, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodPatch,
		"/repos/{repo_ref}/pullreq/{pullreq_number}/comments/{pullreq_comment_id}", commentUpdatePullReq)

	commentDeletePullReq := openapi3.Operation{}
	commentDeletePullReq.WithTags("pullreq")
	commentDeletePullReq.WithMapOfAnything(map[string]interface{}{"operationId": "commentDeletePullReq"})
	_ = reflector.SetRequest(&commentDeletePullReq, new(commentDeletePullReqRequest), http.MethodDelete)
	_ = reflector.SetJSONResponse(&commentDeletePullReq, nil, http.StatusNoContent)
	_ = reflector.SetJSONResponse(&commentDeletePullReq, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&commentDeletePullReq, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&commentDeletePullReq, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&commentDeletePullReq, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodDelete,
		"/repos/{repo_ref}/pullreq/{pullreq_number}/comments/{pullreq_comment_id}", commentDeletePullReq)

	commentStatusPullReq := openapi3.Operation{}
	commentStatusPullReq.WithTags("pullreq")
	commentStatusPullReq.WithMapOfAnything(map[string]interface{}{"operationId": "commentStatusPullReq"})
	_ = reflector.SetRequest(&commentStatusPullReq, new(commentStatusPullReqRequest), http.MethodPut)
	_ = reflector.SetJSONResponse(&commentStatusPullReq, new(types.PullReqActivity), http.StatusOK)
	_ = reflector.SetJSONResponse(&commentStatusPullReq, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&commentStatusPullReq, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&commentStatusPullReq, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&commentStatusPullReq, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodPut,
		"/repos/{repo_ref}/pullreq/{pullreq_number}/comments/{pullreq_comment_id}/status", commentStatusPullReq)

	commentApplySuggestions := openapi3.Operation{}
	commentApplySuggestions.WithTags("pullreq")
	commentApplySuggestions.WithMapOfAnything(map[string]interface{}{"operationId": "commentApplySuggestions"})
	_ = reflector.SetRequest(&commentApplySuggestions, new(commentApplySuggestionstRequest), http.MethodPost)
	_ = reflector.SetJSONResponse(&commentApplySuggestions, new(pullreq.CommentApplySuggestionsOutput), http.StatusOK)
	_ = reflector.SetJSONResponse(&commentApplySuggestions, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&commentApplySuggestions, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&commentApplySuggestions, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&commentApplySuggestions, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&commentApplySuggestions, new(types.RulesViolations), http.StatusUnprocessableEntity)
	_ = reflector.Spec.AddOperation(http.MethodPost,
		"/repos/{repo_ref}/pullreq/{pullreq_number}/comments/apply-suggestions", commentApplySuggestions)

	reviewerAdd := openapi3.Operation{}
	reviewerAdd.WithTags("pullreq")
	reviewerAdd.WithMapOfAnything(map[string]interface{}{"operationId": "reviewerAddPullReq"})
	_ = reflector.SetRequest(&reviewerAdd, new(reviewerAddPullReqRequest), http.MethodPut)
	_ = reflector.SetJSONResponse(&reviewerAdd, new(types.PullReqReviewer), http.StatusOK)
	_ = reflector.SetJSONResponse(&reviewerAdd, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&reviewerAdd, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&reviewerAdd, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&reviewerAdd, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodPut,
		"/repos/{repo_ref}/pullreq/{pullreq_number}/reviewers", reviewerAdd)

	reviewerList := openapi3.Operation{}
	reviewerList.WithTags("pullreq")
	reviewerList.WithMapOfAnything(map[string]interface{}{"operationId": "reviewerListPullReq"})
	_ = reflector.SetRequest(&reviewerList, new(reviewerListPullReqRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&reviewerList, new([]*types.PullReqReviewer), http.StatusOK)
	_ = reflector.SetJSONResponse(&reviewerList, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&reviewerList, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&reviewerList, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&reviewerList, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodGet,
		"/repos/{repo_ref}/pullreq/{pullreq_number}/reviewers", reviewerList)

	reviewerDelete := openapi3.Operation{}
	reviewerDelete.WithTags("pullreq")
	reviewerDelete.WithMapOfAnything(map[string]interface{}{"operationId": "reviewerDeletePullReq"})
	_ = reflector.SetRequest(&reviewerDelete, new(reviewerDeletePullReqRequest), http.MethodDelete)
	_ = reflector.SetJSONResponse(&reviewerDelete, nil, http.StatusNoContent)
	_ = reflector.SetJSONResponse(&reviewerDelete, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&reviewerDelete, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&reviewerDelete, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&reviewerDelete, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodDelete,
		"/repos/{repo_ref}/pullreq/{pullreq_number}/reviewers/{pullreq_reviewer_id}", reviewerDelete)

	reviewSubmit := openapi3.Operation{}
	reviewSubmit.WithTags("pullreq")
	reviewSubmit.WithMapOfAnything(map[string]interface{}{"operationId": "reviewSubmitPullReq"})
	_ = reflector.SetRequest(&reviewSubmit, new(reviewSubmitPullReqRequest), http.MethodPost)
	_ = reflector.SetJSONResponse(&reviewSubmit, nil, http.StatusNoContent)
	_ = reflector.SetJSONResponse(&reviewSubmit, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&reviewSubmit, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&reviewSubmit, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&reviewSubmit, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodPost,
		"/repos/{repo_ref}/pullreq/{pullreq_number}/reviews", reviewSubmit)

	mergePullReqOp := openapi3.Operation{}
	mergePullReqOp.WithTags("pullreq")
	mergePullReqOp.WithMapOfAnything(map[string]interface{}{"operationId": "mergePullReqOp"})
	_ = reflector.SetRequest(&mergePullReqOp, new(mergePullReq), http.MethodPost)
	_ = reflector.SetJSONResponse(&mergePullReqOp, new(types.MergeResponse), http.StatusOK)
	_ = reflector.SetJSONResponse(&mergePullReqOp, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&mergePullReqOp, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&mergePullReqOp, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&mergePullReqOp, new(usererror.Error), http.StatusNotFound)
	_ = reflector.SetJSONResponse(&mergePullReqOp, new(usererror.Error), http.StatusMethodNotAllowed)
	_ = reflector.SetJSONResponse(&mergePullReqOp, new(usererror.Error), http.StatusConflict)
	_ = reflector.SetJSONResponse(&mergePullReqOp, new(types.MergeViolations), http.StatusUnprocessableEntity)
	_ = reflector.Spec.AddOperation(http.MethodPost,
		"/repos/{repo_ref}/pullreq/{pullreq_number}/merge", mergePullReqOp)

	opListCommits := openapi3.Operation{}
	opListCommits.WithTags("pullreq")
	opListCommits.WithMapOfAnything(map[string]interface{}{"operationId": "listPullReqCommits"})
	opListCommits.WithParameters(QueryParameterPage, QueryParameterLimit)
	_ = reflector.SetRequest(&opListCommits, new(pullReqRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&opListCommits, []types.Commit{}, http.StatusOK)
	_ = reflector.SetJSONResponse(&opListCommits, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opListCommits, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opListCommits, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opListCommits, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/repos/{repo_ref}/pullreq/{pullreq_number}/commits", opListCommits)

	opMetaData := openapi3.Operation{}
	opMetaData.WithTags("pullreq")
	opMetaData.WithMapOfAnything(map[string]interface{}{"operationId": "pullReqMetaData"})
	_ = reflector.SetRequest(&opMetaData, new(pullReqRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&opMetaData, new(types.PullReqStats), http.StatusOK)
	_ = reflector.SetJSONResponse(&opMetaData, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opMetaData, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opMetaData, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opMetaData, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/repos/{repo_ref}/pullreq/{pullreq_number}/metadata", opMetaData)

	fileViewAdd := openapi3.Operation{}
	fileViewAdd.WithTags("pullreq")
	fileViewAdd.WithMapOfAnything(map[string]interface{}{"operationId": "fileViewAddPullReq"})
	_ = reflector.SetRequest(&fileViewAdd, new(fileViewAddPullReqRequest), http.MethodPut)
	_ = reflector.SetJSONResponse(&fileViewAdd, new(types.PullReqFileView), http.StatusOK)
	_ = reflector.SetJSONResponse(&fileViewAdd, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&fileViewAdd, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&fileViewAdd, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&fileViewAdd, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodPut,
		"/repos/{repo_ref}/pullreq/{pullreq_number}/file-views", fileViewAdd)

	fileViewList := openapi3.Operation{}
	fileViewList.WithTags("pullreq")
	fileViewList.WithMapOfAnything(map[string]interface{}{"operationId": "fileViewListPullReq"})
	_ = reflector.SetRequest(&fileViewList, new(fileViewListPullReqRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&fileViewList, []types.PullReqFileView{}, http.StatusOK)
	_ = reflector.SetJSONResponse(&fileViewList, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&fileViewList, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&fileViewList, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&fileViewList, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodGet,
		"/repos/{repo_ref}/pullreq/{pullreq_number}/file-views", fileViewList)

	fileViewDelete := openapi3.Operation{}
	fileViewDelete.WithTags("pullreq")
	fileViewDelete.WithMapOfAnything(map[string]interface{}{"operationId": "fileViewDeletePullReq"})
	_ = reflector.SetRequest(&fileViewDelete, new(fileViewDeletePullReqRequest), http.MethodDelete)
	_ = reflector.SetJSONResponse(&fileViewDelete, nil, http.StatusNoContent)
	_ = reflector.SetJSONResponse(&fileViewDelete, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&fileViewDelete, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&fileViewDelete, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&fileViewDelete, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodDelete,
		"/repos/{repo_ref}/pullreq/{pullreq_number}/file-views/{file_path}", fileViewDelete)

	codeOwners := openapi3.Operation{}
	codeOwners.WithTags("pullreq")
	codeOwners.WithMapOfAnything(map[string]interface{}{"operationId": "codeownersPullReq"})
	_ = reflector.SetRequest(&codeOwners, new(pullReqRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&codeOwners, types.CodeOwnerEvaluation{}, http.StatusOK)
	_ = reflector.SetJSONResponse(&codeOwners, new(usererror.Error), http.StatusUnprocessableEntity)
	_ = reflector.SetJSONResponse(&codeOwners, new(usererror.Error), http.StatusNotFound)
	_ = reflector.SetJSONResponse(&codeOwners, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&codeOwners, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&codeOwners, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&codeOwners, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodGet,
		"/repos/{repo_ref}/pullreq/{pullreq_number}/codeowners", codeOwners)

	opDiff := openapi3.Operation{}
	opDiff.WithTags("pullreq")
	opDiff.WithMapOfAnything(map[string]interface{}{"operationId": "diffPullReq"})
	panicOnErr(reflector.SetRequest(&opDiff, new(getRawPRDiffRequest), http.MethodGet))
	panicOnErr(reflector.SetStringResponse(&opDiff, http.StatusOK, "text/plain"))
	panicOnErr(reflector.SetJSONResponse(&opDiff, new([]git.FileDiff), http.StatusOK))
	panicOnErr(reflector.SetJSONResponse(&opDiff, new(usererror.Error), http.StatusInternalServerError))
	panicOnErr(reflector.SetJSONResponse(&opDiff, new(usererror.Error), http.StatusUnauthorized))
	panicOnErr(reflector.SetJSONResponse(&opDiff, new(usererror.Error), http.StatusForbidden))
	panicOnErr(reflector.SetJSONResponse(&opDiff, new(usererror.Error), http.StatusNotFound))
	panicOnErr(reflector.Spec.AddOperation(http.MethodGet, "/repos/{repo_ref}/pullreq/{pullreq_number}/diff", opDiff))

	opPostDiff := openapi3.Operation{}
	opPostDiff.WithTags("pullreq")
	opPostDiff.WithMapOfAnything(map[string]interface{}{"operationId": "diffPullReqPost"})
	panicOnErr(reflector.SetRequest(&opPostDiff, new(postRawPRDiffRequest), http.MethodPost))
	panicOnErr(reflector.SetStringResponse(&opPostDiff, http.StatusOK, "text/plain"))
	panicOnErr(reflector.SetJSONResponse(&opPostDiff, new([]git.FileDiff), http.StatusOK))
	panicOnErr(reflector.SetJSONResponse(&opPostDiff, new(usererror.Error), http.StatusInternalServerError))
	panicOnErr(reflector.SetJSONResponse(&opPostDiff, new(usererror.Error), http.StatusUnauthorized))
	panicOnErr(reflector.SetJSONResponse(&opPostDiff, new(usererror.Error), http.StatusForbidden))
	panicOnErr(reflector.SetJSONResponse(&opPostDiff, new(usererror.Error), http.StatusNotFound))
	panicOnErr(reflector.Spec.AddOperation(http.MethodPost, "/repos/{repo_ref}/pullreq/{pullreq_number}/diff", opPostDiff))

	opChecks := openapi3.Operation{}
	opChecks.WithTags("pullreq")
	opChecks.WithMapOfAnything(map[string]interface{}{"operationId": "checksPullReq"})
	_ = reflector.SetRequest(&opChecks, new(getPullReqChecksRequest), http.MethodGet)
	panicOnErr(reflector.SetJSONResponse(&opChecks, new([]types.PullReqChecks), http.StatusOK))
	panicOnErr(reflector.SetJSONResponse(&opChecks, new(usererror.Error), http.StatusInternalServerError))
	panicOnErr(reflector.SetJSONResponse(&opChecks, new(usererror.Error), http.StatusUnauthorized))
	panicOnErr(reflector.SetJSONResponse(&opChecks, new(usererror.Error), http.StatusForbidden))
	panicOnErr(reflector.SetJSONResponse(&opChecks, new(usererror.Error), http.StatusNotFound))
	panicOnErr(reflector.Spec.AddOperation(http.MethodGet, "/repos/{repo_ref}/pullreq/{pullreq_number}/checks", opChecks))

	opAssignLabel := openapi3.Operation{}
	opAssignLabel.WithTags("pullreq")
	opAssignLabel.WithMapOfAnything(map[string]interface{}{"operationId": "assignLabel"})
	_ = reflector.SetRequest(&opAssignLabel, new(pullReqAssignLabelInput), http.MethodPut)
	_ = reflector.SetJSONResponse(&opAssignLabel, new(types.PullReqLabel), http.StatusOK)
	_ = reflector.SetJSONResponse(&opAssignLabel, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opAssignLabel, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opAssignLabel, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opAssignLabel, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodPut,
		"/repos/{repo_ref}/pullreq/{pullreq_number}/labels", opAssignLabel)

	opListLabels := openapi3.Operation{}
	opListLabels.WithTags("pullreq")
	opListLabels.WithMapOfAnything(map[string]interface{}{"operationId": "listLabels"})
	opListLabels.WithParameters(
		QueryParameterPage, QueryParameterLimit, queryParameterAssignable, queryParameterQueryLabel)
	_ = reflector.SetRequest(&opListLabels, new(pullReqRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&opListLabels, new(types.ScopesLabels), http.StatusOK)
	_ = reflector.SetJSONResponse(&opListLabels, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opListLabels, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opListLabels, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opListLabels, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodGet,
		"/repos/{repo_ref}/pullreq/{pullreq_number}/labels", opListLabels)

	opUnassignLabel := openapi3.Operation{}
	opUnassignLabel.WithTags("pullreq")
	opUnassignLabel.WithMapOfAnything(map[string]interface{}{"operationId": "unassignLabel"})
	_ = reflector.SetRequest(&opUnassignLabel, struct {
		pullReqRequest
		LabelID int64 `path:"label_id"`
	}{}, http.MethodDelete)
	_ = reflector.SetJSONResponse(&opUnassignLabel, nil, http.StatusNoContent)
	_ = reflector.SetJSONResponse(&opUnassignLabel, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opUnassignLabel, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opUnassignLabel, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opUnassignLabel, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodDelete,
		"/repos/{repo_ref}/pullreq/{pullreq_number}/labels/{label_id}", opUnassignLabel)
}
