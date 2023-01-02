// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package openapi

import (
	"net/http"

	"github.com/harness/gitness/internal/api/controller/repo"
	"github.com/harness/gitness/internal/api/request"
	"github.com/harness/gitness/internal/api/usererror"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/gotidy/ptr"
	"github.com/swaggest/openapi-go/openapi3"
)

type createRepositoryRequest struct {
	repo.CreateInput
}

type gitignoreRequest struct {
}

type licenseRequest struct {
}

type repoRequest struct {
	Ref string `path:"repo_ref"`
}

type updateRepoRequest struct {
	repoRequest
	repo.UpdateInput
}

type moveRepoRequest struct {
	repoRequest
	repo.MoveInput
}

type createRepoPathRequest struct {
	repoRequest
	repo.CreatePathInput
}

type deleteRepoPathRequest struct {
	repoRequest
	PathID string `path:"path_id"`
}

type getContentRequest struct {
	repoRequest
	Path string `path:"path"`
}

type commitFilesRequest struct {
	repoRequest
	repo.CommitFilesOptions
}

// contentType is a plugin for repo.ContentType to allow using oneof.
type contentType string

func (contentType) Enum() []interface{} {
	return []interface{}{repo.ContentTypeFile, repo.ContentTypeDir, repo.ContentTypeSymlink, repo.ContentTypeSubmodule}
}

// contentInfo is used to overshadow the contentype of repo.ContentInfo.
type contentInfo struct {
	repo.ContentInfo
	Type contentType `json:"type"`
}

// dirContent is used to overshadow the Entries type of repo.DirContent.
type dirContent struct {
	repo.DirContent
	Entries []contentInfo `json:"entries"`
}

// content is a plugin for repo.content to allow using oneof.
type content struct{}

func (content) JSONSchemaOneOf() []interface{} {
	return []interface{}{repo.FileContent{}, dirContent{}, repo.SymlinkContent{}, repo.SubmoduleContent{}}
}

// getContentOutput is used to overshadow the content and contenttype of repo.GetContentOutput.
type getContentOutput struct {
	repo.GetContentOutput
	Type    contentType `json:"type"`
	Content content     `json:"content"`
}

type listCommitsRequest struct {
	repoRequest
}

type calculateCommitDivergenceRequest struct {
	repoRequest
	repo.GetCommitDivergencesInput
}

type listBranchesRequest struct {
	repoRequest
}
type createBranchRequest struct {
	repoRequest
	repo.CreateBranchInput
}

type deleteBranchRequest struct {
	repoRequest
	BranchName string `path:"branch_name"`
}

type listTagsRequest struct {
	repoRequest
}

type getRawDiffRequest struct {
	repoRequest
	Range string `path:"range" example:"main..dev"`
}

var queryParameterGitRef = openapi3.ParameterOrRef{
	Parameter: &openapi3.Parameter{
		Name: request.QueryParamGitRef,
		In:   openapi3.ParameterInQuery,
		Description: ptr.String("The git reference (branch / tag / commitID) that will be used to retrieve the data. " +
			"If no value is provided the default branch of the repository is used."),
		Required: ptr.Bool(false),
		Schema: &openapi3.SchemaOrRef{
			Schema: &openapi3.Schema{
				Type:    ptrSchemaType(openapi3.SchemaTypeString),
				Default: ptrptr("{Repository Default Branch}"),
			},
		},
	},
}

var queryParameterIncludeCommit = openapi3.ParameterOrRef{
	Parameter: &openapi3.Parameter{
		Name:        request.QueryParamIncludeCommit,
		In:          openapi3.ParameterInQuery,
		Description: ptr.String("Indicates whether optional commit information should be included in the response."),
		Required:    ptr.Bool(false),
		Schema: &openapi3.SchemaOrRef{
			Schema: &openapi3.Schema{
				Type:    ptrSchemaType(openapi3.SchemaTypeBoolean),
				Default: ptrptr(false),
			},
		},
	},
}

// TODO: this is technically coming from harness package, but we can't reference that.
var queryParameterSpacePath = openapi3.ParameterOrRef{
	Parameter: &openapi3.Parameter{
		Name:        "space_path",
		In:          openapi3.ParameterInQuery,
		Description: ptr.String("path of parent space (Not needed in standalone)."),
		Required:    ptr.Bool(false),
		Schema: &openapi3.SchemaOrRef{
			Schema: &openapi3.Schema{
				Type:    ptrSchemaType(openapi3.SchemaTypeString),
				Default: ptrptr(false),
			},
		},
	},
}

var queryParameterSortBranch = openapi3.ParameterOrRef{
	Parameter: &openapi3.Parameter{
		Name:        request.QueryParamSort,
		In:          openapi3.ParameterInQuery,
		Description: ptr.String("The data by which the branches are sorted."),
		Required:    ptr.Bool(false),
		Schema: &openapi3.SchemaOrRef{
			Schema: &openapi3.Schema{
				Type:    ptrSchemaType(openapi3.SchemaTypeString),
				Default: ptrptr(enum.BranchSortOptionName.String()),
				Enum: []interface{}{
					ptr.String(enum.BranchSortOptionName.String()),
					ptr.String(enum.BranchSortOptionDate.String()),
				},
			},
		},
	},
}

var queryParameterQueryBranches = openapi3.ParameterOrRef{
	Parameter: &openapi3.Parameter{
		Name:        request.QueryParamQuery,
		In:          openapi3.ParameterInQuery,
		Description: ptr.String("The substring by which the branches are filtered."),
		Required:    ptr.Bool(false),
		Schema: &openapi3.SchemaOrRef{
			Schema: &openapi3.Schema{
				Type: ptrSchemaType(openapi3.SchemaTypeString),
			},
		},
	},
}

var queryParameterSortTags = openapi3.ParameterOrRef{
	Parameter: &openapi3.Parameter{
		Name:        request.QueryParamSort,
		In:          openapi3.ParameterInQuery,
		Description: ptr.String("The data by which the tags are sorted."),
		Required:    ptr.Bool(false),
		Schema: &openapi3.SchemaOrRef{
			Schema: &openapi3.Schema{
				Type:    ptrSchemaType(openapi3.SchemaTypeString),
				Default: ptrptr(enum.TagSortOptionName.String()),
				Enum: []interface{}{
					ptr.String(enum.TagSortOptionName.String()),
					ptr.String(enum.TagSortOptionDate.String()),
				},
			},
		},
	},
}

var queryParameterQueryTags = openapi3.ParameterOrRef{
	Parameter: &openapi3.Parameter{
		Name:        request.QueryParamQuery,
		In:          openapi3.ParameterInQuery,
		Description: ptr.String("The substring by which the tags are filtered."),
		Required:    ptr.Bool(false),
		Schema: &openapi3.SchemaOrRef{
			Schema: &openapi3.Schema{
				Type: ptrSchemaType(openapi3.SchemaTypeString),
			},
		},
	},
}

//nolint:funlen
func repoOperations(reflector *openapi3.Reflector) {
	createRepository := openapi3.Operation{}
	createRepository.WithTags("repository")
	createRepository.WithMapOfAnything(map[string]interface{}{"operationId": "createRepository"})
	createRepository.WithParameters(queryParameterSpacePath)
	_ = reflector.SetRequest(&createRepository, new(createRepositoryRequest), http.MethodPost)
	_ = reflector.SetJSONResponse(&createRepository, new(types.Repository), http.StatusCreated)
	_ = reflector.SetJSONResponse(&createRepository, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&createRepository, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&createRepository, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&createRepository, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodPost, "/repos", createRepository)

	opFind := openapi3.Operation{}
	opFind.WithTags("repository")
	opFind.WithMapOfAnything(map[string]interface{}{"operationId": "findRepository"})
	_ = reflector.SetRequest(&opFind, new(repoRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&opFind, new(types.Repository), http.StatusOK)
	_ = reflector.SetJSONResponse(&opFind, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opFind, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opFind, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opFind, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/repos/{repo_ref}", opFind)

	opUpdate := openapi3.Operation{}
	opUpdate.WithTags("repository")
	opUpdate.WithMapOfAnything(map[string]interface{}{"operationId": "updateRepository"})
	_ = reflector.SetRequest(&opUpdate, new(updateRepoRequest), http.MethodPatch)
	_ = reflector.SetJSONResponse(&opUpdate, new(types.Repository), http.StatusOK)
	_ = reflector.SetJSONResponse(&opUpdate, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opUpdate, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opUpdate, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opUpdate, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opUpdate, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodPatch, "/repos/{repo_ref}", opUpdate)

	opDelete := openapi3.Operation{}
	opDelete.WithTags("repository")
	opDelete.WithMapOfAnything(map[string]interface{}{"operationId": "deleteRepository"})
	_ = reflector.SetRequest(&opDelete, new(repoRequest), http.MethodDelete)
	_ = reflector.SetJSONResponse(&opDelete, nil, http.StatusNoContent)
	_ = reflector.SetJSONResponse(&opDelete, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opDelete, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opDelete, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opDelete, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodDelete, "/repos/{repo_ref}", opDelete)

	opMove := openapi3.Operation{}
	opMove.WithTags("repository")
	opMove.WithMapOfAnything(map[string]interface{}{"operationId": "moveRepository"})
	_ = reflector.SetRequest(&opMove, new(moveRepoRequest), http.MethodPost)
	_ = reflector.SetJSONResponse(&opMove, new(types.Repository), http.StatusOK)
	_ = reflector.SetJSONResponse(&opMove, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opMove, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opMove, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opMove, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodPost, "/repos/{repo_ref}/move", opMove)

	opServiceAccounts := openapi3.Operation{}
	opServiceAccounts.WithTags("repository")
	opServiceAccounts.WithMapOfAnything(map[string]interface{}{"operationId": "listRepositoryServiceAccounts"})
	_ = reflector.SetRequest(&opServiceAccounts, new(repoRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&opServiceAccounts, []types.ServiceAccount{}, http.StatusOK)
	_ = reflector.SetJSONResponse(&opServiceAccounts, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opServiceAccounts, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opServiceAccounts, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opServiceAccounts, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/repos/{repo_ref}/service-accounts", opServiceAccounts)

	opListPaths := openapi3.Operation{}
	opListPaths.WithTags("repository")
	opListPaths.WithMapOfAnything(map[string]interface{}{"operationId": "listRepositoryPaths"})
	opListPaths.WithParameters(queryParameterPage, queryParameterLimit)
	_ = reflector.SetRequest(&opListPaths, new(repoRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&opListPaths, []types.Path{}, http.StatusOK)
	_ = reflector.SetJSONResponse(&opListPaths, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opListPaths, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opListPaths, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opListPaths, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/repos/{repo_ref}/paths", opListPaths)

	opCreatePath := openapi3.Operation{}
	opCreatePath.WithTags("repository")
	opCreatePath.WithMapOfAnything(map[string]interface{}{"operationId": "createRepositoryPath"})
	_ = reflector.SetRequest(&opCreatePath, new(createRepoPathRequest), http.MethodPost)
	_ = reflector.SetJSONResponse(&opCreatePath, new(types.Path), http.StatusCreated)
	_ = reflector.SetJSONResponse(&opCreatePath, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opCreatePath, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opCreatePath, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opCreatePath, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodPost, "/repos/{repo_ref}/paths", opCreatePath)

	onDeletePath := openapi3.Operation{}
	onDeletePath.WithTags("repository")
	onDeletePath.WithMapOfAnything(map[string]interface{}{"operationId": "deleteRepositoryPath"})
	_ = reflector.SetRequest(&onDeletePath, new(deleteRepoPathRequest), http.MethodDelete)
	_ = reflector.SetJSONResponse(&onDeletePath, nil, http.StatusNoContent)
	_ = reflector.SetJSONResponse(&onDeletePath, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&onDeletePath, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&onDeletePath, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&onDeletePath, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodDelete, "/repos/{repo_ref}/paths/{path_id}", onDeletePath)

	opGetContent := openapi3.Operation{}
	opGetContent.WithTags("repository")
	opGetContent.WithMapOfAnything(map[string]interface{}{"operationId": "getContent"})
	opGetContent.WithParameters(queryParameterGitRef, queryParameterIncludeCommit)
	_ = reflector.SetRequest(&opGetContent, new(getContentRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&opGetContent, new(getContentOutput), http.StatusOK)
	_ = reflector.SetJSONResponse(&opGetContent, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opGetContent, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opGetContent, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opGetContent, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/repos/{repo_ref}/content/{path}", opGetContent)

	opListCommits := openapi3.Operation{}
	opListCommits.WithTags("repository")
	opListCommits.WithMapOfAnything(map[string]interface{}{"operationId": "listCommits"})
	opListCommits.WithParameters(queryParameterGitRef, queryParameterPage, queryParameterLimit)
	_ = reflector.SetRequest(&opListCommits, new(listCommitsRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&opListCommits, []repo.Commit{}, http.StatusOK)
	_ = reflector.SetJSONResponse(&opListCommits, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opListCommits, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opListCommits, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opListCommits, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/repos/{repo_ref}/commits", opListCommits)

	opCalulateCommitDivergence := openapi3.Operation{}
	opCalulateCommitDivergence.WithTags("repository")
	opCalulateCommitDivergence.WithMapOfAnything(map[string]interface{}{"operationId": "calculateCommitDivergence"})
	_ = reflector.SetRequest(&opCalulateCommitDivergence, new(calculateCommitDivergenceRequest), http.MethodPost)
	_ = reflector.SetJSONResponse(&opCalulateCommitDivergence, []repo.CommitDivergence{}, http.StatusOK)
	_ = reflector.SetJSONResponse(&opCalulateCommitDivergence, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opCalulateCommitDivergence, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opCalulateCommitDivergence, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opCalulateCommitDivergence, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodPost, "/repos/{repo_ref}/commits/calculate-divergence",
		opCalulateCommitDivergence)

	opCreateBranch := openapi3.Operation{}
	opCreateBranch.WithTags("repository")
	opCreateBranch.WithMapOfAnything(map[string]interface{}{"operationId": "createBranch"})
	_ = reflector.SetRequest(&opCreateBranch, new(createBranchRequest), http.MethodPost)
	_ = reflector.SetJSONResponse(&opCreateBranch, new(repo.Branch), http.StatusCreated)
	_ = reflector.SetJSONResponse(&opCreateBranch, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opCreateBranch, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opCreateBranch, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opCreateBranch, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodPost, "/repos/{repo_ref}/branches", opCreateBranch)

	onDeleteBranch := openapi3.Operation{}
	onDeleteBranch.WithTags("repository")
	onDeleteBranch.WithMapOfAnything(map[string]interface{}{"operationId": "deleteBranch"})
	_ = reflector.SetRequest(&onDeleteBranch, new(deleteBranchRequest), http.MethodDelete)
	_ = reflector.SetJSONResponse(&onDeleteBranch, nil, http.StatusNoContent)
	_ = reflector.SetJSONResponse(&onDeleteBranch, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&onDeleteBranch, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&onDeleteBranch, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&onDeleteBranch, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodDelete, "/repos/{repo_ref}/branches/{branch_name}", onDeleteBranch)

	opListBranches := openapi3.Operation{}
	opListBranches.WithTags("repository")
	opListBranches.WithMapOfAnything(map[string]interface{}{"operationId": "listBranches"})
	opListBranches.WithParameters(queryParameterIncludeCommit,
		queryParameterQueryBranches, queryParameterOrder, queryParameterSortBranch,
		queryParameterPage, queryParameterLimit)
	_ = reflector.SetRequest(&opListBranches, new(listBranchesRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&opListBranches, []repo.Branch{}, http.StatusOK)
	_ = reflector.SetJSONResponse(&opListBranches, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opListBranches, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opListBranches, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opListBranches, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/repos/{repo_ref}/branches", opListBranches)

	opListTags := openapi3.Operation{}
	opListTags.WithTags("repository")
	opListTags.WithMapOfAnything(map[string]interface{}{"operationId": "listTags"})
	opListTags.WithParameters(queryParameterIncludeCommit,
		queryParameterQueryTags, queryParameterOrder, queryParameterSortTags,
		queryParameterPage, queryParameterLimit)
	_ = reflector.SetRequest(&opListTags, new(listTagsRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&opListTags, []repo.CommitTag{}, http.StatusOK)
	_ = reflector.SetJSONResponse(&opListTags, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opListTags, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opListTags, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opListTags, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/repos/{repo_ref}/tags", opListTags)

	opCommitFiles := openapi3.Operation{}
	opCommitFiles.WithTags("repository")
	opCommitFiles.WithMapOfAnything(map[string]interface{}{"operationId": "commitFiles"})
	_ = reflector.SetRequest(&opCommitFiles, new(commitFilesRequest), http.MethodPost)
	_ = reflector.SetJSONResponse(&opCommitFiles, repo.CommitFilesResponse{}, http.StatusOK)
	_ = reflector.SetJSONResponse(&opCommitFiles, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opCommitFiles, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opCommitFiles, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opCommitFiles, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opCommitFiles, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodPost, "/repos/{repo_ref}/commits", opCommitFiles)

	opRawDiff := openapi3.Operation{}
	opRawDiff.WithTags("repository")
	opRawDiff.WithMapOfAnything(map[string]interface{}{"operationId": "rawDiff"})
	_ = reflector.SetRequest(&opRawDiff, new(getRawDiffRequest), http.MethodGet)
	_ = reflector.SetStringResponse(&opRawDiff, http.StatusOK, "text/plain")
	_ = reflector.SetJSONResponse(&opRawDiff, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opRawDiff, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opRawDiff, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/repos/{repo_ref}/compare/{range}", opRawDiff)
}
