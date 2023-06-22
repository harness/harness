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
						Enum:    enum.PullReqState("").Enum(),
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
		queryParameterPage, queryParameterLimit)
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
		queryParameterAfter, queryParameterBeforePullRequestActivity, queryParameterLimit)
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
	err := reflector.SetRequest(&reviewerDelete, new(reviewerDeletePullReqRequest), http.MethodDelete)
	err = reflector.SetJSONResponse(&reviewerDelete, nil, http.StatusNoContent)
	err = reflector.SetJSONResponse(&reviewerDelete, new(usererror.Error), http.StatusBadRequest)
	err = reflector.SetJSONResponse(&reviewerDelete, new(usererror.Error), http.StatusInternalServerError)
	err = reflector.SetJSONResponse(&reviewerDelete, new(usererror.Error), http.StatusUnauthorized)
	err = reflector.SetJSONResponse(&reviewerDelete, new(usererror.Error), http.StatusForbidden)
	err = reflector.Spec.AddOperation(http.MethodDelete,
		"/repos/{repo_ref}/pullreq/{pullreq_number}/reviewers/{pullreq_reviewer_id}", reviewerDelete)
	if err != nil {
		panic(err)
	}

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
	_ = reflector.SetJSONResponse(&mergePullReqOp, new(usererror.Error), http.StatusUnprocessableEntity)
	_ = reflector.Spec.AddOperation(http.MethodPost,
		"/repos/{repo_ref}/pullreq/{pullreq_number}/merge", mergePullReqOp)

	opListCommits := openapi3.Operation{}
	opListCommits.WithTags("pullreq")
	opListCommits.WithMapOfAnything(map[string]interface{}{"operationId": "listPullReqCommits"})
	opListCommits.WithParameters(queryParameterPage, queryParameterLimit)
	_ = reflector.SetRequest(&opListCommits, new(pullReqRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&opListCommits, []types.Commit{}, http.StatusOK)
	_ = reflector.SetJSONResponse(&opListCommits, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opListCommits, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opListCommits, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opListCommits, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/repos/{repo_ref}/pullreq/{pullreq_number}/commits", opListCommits)

	opRawDiff := openapi3.Operation{}
	opRawDiff.WithTags("pullreq")
	opRawDiff.WithMapOfAnything(map[string]interface{}{"operationId": "rawPullReqDiff"})
	_ = reflector.SetRequest(&opRawDiff, new(pullReqRequest), http.MethodGet)
	_ = reflector.SetStringResponse(&opRawDiff, http.StatusOK, "text/plain")
	_ = reflector.SetJSONResponse(&opRawDiff, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opRawDiff, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opRawDiff, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opRawDiff, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/repos/{repo_ref}/pullreq/{pullreq_number}/diff", opRawDiff)

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
}
