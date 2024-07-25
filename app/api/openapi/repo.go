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

	"github.com/harness/gitness/app/api/controller/repo"
	"github.com/harness/gitness/app/api/controller/reposettings"
	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/git"
	gittypes "github.com/harness/gitness/git/api"
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

type getContentRequest struct {
	repoRequest
	Path string `path:"path"`
}

type pathsDetailsRequest struct {
	repoRequest
	repo.PathsDetailsInput
}

type getBlameRequest struct {
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

type GetCommitRequest struct {
	repoRequest
	CommitSHA string `path:"commit_sha"`
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

type getBranchRequest struct {
	repoRequest
	BranchName string `path:"branch_name"`
}

type deleteBranchRequest struct {
	repoRequest
	BranchName string `path:"branch_name"`
}

type createTagRequest struct {
	repoRequest
	repo.CreateCommitTagInput
}

type listTagsRequest struct {
	repoRequest
}

type deleteTagRequest struct {
	repoRequest
	TagName string `path:"tag_name"`
}

type getRawDiffRequest struct {
	repoRequest
	Range string   `path:"range" example:"main..dev"`
	Path  []string `query:"path" description:"provide path for diff operation"`
}

type postRawDiffRequest struct {
	repoRequest
	gittypes.FileDiffRequests
	Range string `path:"range" example:"main..dev"`
}

type codeOwnersValidate struct {
	repoRequest
}

// ruleType is a plugin for types.RuleType to allow using oneof.
type ruleType string

func (ruleType) Enum() []interface{} {
	return []interface{}{protection.TypeBranch}
}

// ruleDefinition is a plugin for types.Rule Definition to allow using oneof.
type ruleDefinition struct{}

func (ruleDefinition) JSONSchemaOneOf() []interface{} {
	return []interface{}{protection.Branch{}}
}

type rule struct {
	types.Rule

	// overshadow Type and Definition to enable oneof.
	Type       ruleType       `json:"type"`
	Definition ruleDefinition `json:"definition"`

	// overshadow Pattern to correct the type
	Pattern protection.Pattern `json:"pattern"`
}

type restoreRequest struct {
	repoRequest
	repo.RestoreInput
}

type updateRepoPublicAccessRequest struct {
	repoRequest
	repo.UpdatePublicAccessInput
}

type securitySettingsRequest struct {
	repoRequest
	reposettings.SecuritySettings
}

type generalSettingsRequest struct {
	repoRequest
	reposettings.GeneralSettings
}

type archiveRequest struct {
	repoRequest
	GitRef string `path:"git_ref" required:"true"`
	Format string `path:"format" required:"true"`
}

type labelRequest struct {
	Key         string          `json:"key"`
	Description string          `json:"description"`
	Type        enum.LabelType  `json:"type"`
	Color       enum.LabelColor `json:"color"`
}

type labelValueRequest struct {
	Value string          `json:"value"`
	Color enum.LabelColor `json:"color"`
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

var queryParameterPath = openapi3.ParameterOrRef{
	Parameter: &openapi3.Parameter{
		Name:        request.QueryParamPath,
		In:          openapi3.ParameterInQuery,
		Description: ptr.String("Path for which commit information should be retrieved"),
		Required:    ptr.Bool(false),
		Schema: &openapi3.SchemaOrRef{
			Schema: &openapi3.Schema{
				Type:    ptrSchemaType(openapi3.SchemaTypeString),
				Default: ptrptr(""),
			},
		},
	},
}

var queryParameterSince = openapi3.ParameterOrRef{
	Parameter: &openapi3.Parameter{
		Name:        request.QueryParamSince,
		In:          openapi3.ParameterInQuery,
		Description: ptr.String("Epoch since when commit information should be retrieved."),
		Required:    ptr.Bool(false),
		Schema: &openapi3.SchemaOrRef{
			Schema: &openapi3.Schema{
				Type: ptrSchemaType(openapi3.SchemaTypeInteger),
			},
		},
	},
}

var queryParameterUntil = openapi3.ParameterOrRef{
	Parameter: &openapi3.Parameter{
		Name:        request.QueryParamUntil,
		In:          openapi3.ParameterInQuery,
		Description: ptr.String("Epoch until when commit information should be retrieved."),
		Required:    ptr.Bool(false),
		Schema: &openapi3.SchemaOrRef{
			Schema: &openapi3.Schema{
				Type: ptrSchemaType(openapi3.SchemaTypeInteger),
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

var queryParameterIncludeDirectories = openapi3.ParameterOrRef{
	Parameter: &openapi3.Parameter{
		Name:        request.QueryParamIncludeDirectories,
		In:          openapi3.ParameterInQuery,
		Description: ptr.String("Indicates whether directories should be included in the response."),
		Required:    ptr.Bool(false),
		Schema: &openapi3.SchemaOrRef{
			Schema: &openapi3.Schema{
				Type:    ptrSchemaType(openapi3.SchemaTypeBoolean),
				Default: ptrptr(false),
			},
		},
	},
}

var QueryParamIncludeStats = openapi3.ParameterOrRef{
	Parameter: &openapi3.Parameter{
		Name:        request.QueryParamIncludeStats,
		In:          openapi3.ParameterInQuery,
		Description: ptr.String("Indicates whether optional stats should be included in the response."),
		Required:    ptr.Bool(false),
		Schema: &openapi3.SchemaOrRef{
			Schema: &openapi3.Schema{
				Type:    ptrSchemaType(openapi3.SchemaTypeBoolean),
				Default: ptrptr(false),
			},
		},
	},
}

var queryParameterLineFrom = openapi3.ParameterOrRef{
	Parameter: &openapi3.Parameter{
		Name:        request.QueryParamLineFrom,
		In:          openapi3.ParameterInQuery,
		Description: ptr.String("Line number from which the file data is considered"),
		Required:    ptr.Bool(false),
		Schema: &openapi3.SchemaOrRef{
			Schema: &openapi3.Schema{
				Type:    ptrSchemaType(openapi3.SchemaTypeInteger),
				Default: ptrptr(0),
			},
		},
	},
}

var queryParameterLineTo = openapi3.ParameterOrRef{
	Parameter: &openapi3.Parameter{
		Name:        request.QueryParamLineTo,
		In:          openapi3.ParameterInQuery,
		Description: ptr.String("Line number to which the file data is considered"),
		Required:    ptr.Bool(false),
		Schema: &openapi3.SchemaOrRef{
			Schema: &openapi3.Schema{
				Type:    ptrSchemaType(openapi3.SchemaTypeInteger),
				Default: ptrptr(0),
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

var queryParameterAfterCommits = openapi3.ParameterOrRef{
	Parameter: &openapi3.Parameter{
		Name:        request.QueryParamAfter,
		In:          openapi3.ParameterInQuery,
		Description: ptr.String("The result should only contain commits that occurred after the provided reference."),
		Required:    ptr.Bool(false),
		Schema: &openapi3.SchemaOrRef{
			Schema: &openapi3.Schema{
				Type: ptrSchemaType(openapi3.SchemaTypeString),
			},
		},
	},
}

var queryParameterCommitter = openapi3.ParameterOrRef{
	Parameter: &openapi3.Parameter{
		Name:        request.QueryParamCommitter,
		In:          openapi3.ParameterInQuery,
		Description: ptr.String("Committer pattern for which commit information should be retrieved."),
		Required:    ptr.Bool(false),
		Schema: &openapi3.SchemaOrRef{
			Schema: &openapi3.Schema{
				Type: ptrSchemaType(openapi3.SchemaTypeString),
			},
		},
	},
}

var queryParameterQueryRuleList = openapi3.ParameterOrRef{
	Parameter: &openapi3.Parameter{
		Name:        request.QueryParamQuery,
		In:          openapi3.ParameterInQuery,
		Description: ptr.String("The substring by which the repository protection rules are filtered."),
		Required:    ptr.Bool(false),
		Schema: &openapi3.SchemaOrRef{
			Schema: &openapi3.Schema{
				Type: ptrSchemaType(openapi3.SchemaTypeString),
			},
		},
	},
}

var queryParameterSortRuleList = openapi3.ParameterOrRef{
	Parameter: &openapi3.Parameter{
		Name:        request.QueryParamSort,
		In:          openapi3.ParameterInQuery,
		Description: ptr.String("The field by which the protection rules are sorted."),
		Required:    ptr.Bool(false),
		Schema: &openapi3.SchemaOrRef{
			Schema: &openapi3.Schema{
				Type:    ptrSchemaType(openapi3.SchemaTypeString),
				Default: ptrptr(enum.RuleSortCreated),
				Enum:    enum.RuleSort("").Enum(),
			},
		},
	},
}

var queryParameterBypassRules = openapi3.ParameterOrRef{
	Parameter: &openapi3.Parameter{
		Name:        request.QueryParamBypassRules,
		In:          openapi3.ParameterInQuery,
		Description: ptr.String("Bypass rule violations if possible."),
		Required:    ptr.Bool(false),
		Schema: &openapi3.SchemaOrRef{
			Schema: &openapi3.Schema{
				Type:    ptrSchemaType(openapi3.SchemaTypeBoolean),
				Default: ptrptr(false),
			},
		},
	},
}

var queryParameterDeletedAt = openapi3.ParameterOrRef{
	Parameter: &openapi3.Parameter{
		Name:        request.QueryParamDeletedAt,
		In:          openapi3.ParameterInQuery,
		Description: ptr.String("The exact time the resource was delete at in epoch format."),
		Required:    ptr.Bool(true),
		Schema: &openapi3.SchemaOrRef{
			Schema: &openapi3.Schema{
				Type: ptrSchemaType(openapi3.SchemaTypeInteger),
			},
		},
	},
}

var queryParamArchivePaths = openapi3.ParameterOrRef{
	Parameter: &openapi3.Parameter{
		Name: request.QueryParamArchivePaths,
		In:   openapi3.ParameterInQuery,
		Description: ptr.String("Without an optional path parameter, all files and subdirectories of the " +
			"current working directory are included in the archive. If one or more paths are specified," +
			" only these are included."),
		Required: ptr.Bool(false),
		Schema: &openapi3.SchemaOrRef{
			Schema: &openapi3.Schema{
				Type: ptrSchemaType(openapi3.SchemaTypeArray),
				Items: &openapi3.SchemaOrRef{
					Schema: &openapi3.Schema{
						Type: ptrSchemaType(openapi3.SchemaTypeString),
					},
				},
			},
		},
	},
}

var queryParamArchivePrefix = openapi3.ParameterOrRef{
	Parameter: &openapi3.Parameter{
		Name:        request.QueryParamArchivePrefix,
		In:          openapi3.ParameterInQuery,
		Description: ptr.String("Prepend <prefix>/ to paths in the archive."),
		Required:    ptr.Bool(false),
		Schema: &openapi3.SchemaOrRef{
			Schema: &openapi3.Schema{
				Type: ptrSchemaType(openapi3.SchemaTypeString),
			},
		},
	},
}

var queryParamArchiveAttributes = openapi3.ParameterOrRef{
	Parameter: &openapi3.Parameter{
		Name:        request.QueryParamArchiveAttributes,
		In:          openapi3.ParameterInQuery,
		Description: ptr.String("Look for attributes in .gitattributes files in the working tree as well"),
		Required:    ptr.Bool(false),
		Schema: &openapi3.SchemaOrRef{
			Schema: &openapi3.Schema{
				Type: ptrSchemaType(openapi3.SchemaTypeString),
			},
		},
	},
}

var queryParamArchiveTime = openapi3.ParameterOrRef{
	Parameter: &openapi3.Parameter{
		Name: request.QueryParamArchiveTime,
		In:   openapi3.ParameterInQuery,
		Description: ptr.String("Set modification time of archive entries. Without this option the committer " +
			"time is used if <tree-ish> is a commit or tag, and the current time if it is a tree."),
		Required: ptr.Bool(false),
		Schema: &openapi3.SchemaOrRef{
			Schema: &openapi3.Schema{
				Type: ptrSchemaType(openapi3.SchemaTypeString),
			},
		},
	},
}

var queryParamArchiveCompression = openapi3.ParameterOrRef{
	Parameter: &openapi3.Parameter{
		Name: request.QueryParamArchiveCompression,
		In:   openapi3.ParameterInQuery,
		Description: ptr.String("Specify compression level. Larger values allow the command to spend more" +
			" time to compress to smaller size."),
		Required: ptr.Bool(false),
		Schema: &openapi3.SchemaOrRef{
			Schema: &openapi3.Schema{
				Type: ptrSchemaType(openapi3.SchemaTypeInteger),
			},
		},
	},
}

var queryParameterInherited = openapi3.ParameterOrRef{
	Parameter: &openapi3.Parameter{
		Name:        request.QueryParamInherited,
		In:          openapi3.ParameterInQuery,
		Description: ptr.String("The result should inherit labels from parent parent spaces."),
		Required:    ptr.Bool(false),
		Schema: &openapi3.SchemaOrRef{
			Schema: &openapi3.Schema{
				Type:    ptrSchemaType(openapi3.SchemaTypeBoolean),
				Default: ptrptr(false),
			},
		},
	},
}

var queryParameterQueryLabel = openapi3.ParameterOrRef{
	Parameter: &openapi3.Parameter{
		Name:        request.QueryParamQuery,
		In:          openapi3.ParameterInQuery,
		Description: ptr.String("The substring which is used to filter the labels by their key."),
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
	_ = reflector.SetJSONResponse(&createRepository, new(repo.RepositoryOutput), http.StatusCreated)
	_ = reflector.SetJSONResponse(&createRepository, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&createRepository, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&createRepository, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&createRepository, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodPost, "/repos", createRepository)

	importRepository := openapi3.Operation{}
	importRepository.WithTags("repository")
	importRepository.WithMapOfAnything(map[string]interface{}{"operationId": "importRepository"})
	importRepository.WithParameters(queryParameterSpacePath)
	_ = reflector.SetRequest(&importRepository, &struct{ repo.ImportInput }{}, http.MethodPost)
	_ = reflector.SetJSONResponse(&importRepository, new(repo.RepositoryOutput), http.StatusCreated)
	_ = reflector.SetJSONResponse(&importRepository, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&importRepository, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&importRepository, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&importRepository, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodPost, "/repos/import", importRepository)

	opFind := openapi3.Operation{}
	opFind.WithTags("repository")
	opFind.WithMapOfAnything(map[string]interface{}{"operationId": "findRepository"})
	_ = reflector.SetRequest(&opFind, new(repoRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&opFind, new(repo.RepositoryOutput), http.StatusOK)
	_ = reflector.SetJSONResponse(&opFind, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opFind, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opFind, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opFind, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/repos/{repo_ref}", opFind)

	opUpdate := openapi3.Operation{}
	opUpdate.WithTags("repository")
	opUpdate.WithMapOfAnything(map[string]interface{}{"operationId": "updateRepository"})
	_ = reflector.SetRequest(&opUpdate, new(updateRepoRequest), http.MethodPatch)
	_ = reflector.SetJSONResponse(&opUpdate, new(repo.RepositoryOutput), http.StatusOK)
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
	_ = reflector.SetJSONResponse(&opDelete, new(repo.SoftDeleteResponse), http.StatusOK)
	_ = reflector.SetJSONResponse(&opDelete, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opDelete, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opDelete, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opDelete, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodDelete, "/repos/{repo_ref}", opDelete)

	opPurge := openapi3.Operation{}
	opPurge.WithTags("repository")
	opPurge.WithMapOfAnything(map[string]interface{}{"operationId": "purgeRepository"})
	opPurge.WithParameters(queryParameterDeletedAt)
	_ = reflector.SetRequest(&opPurge, new(repoRequest), http.MethodPost)
	_ = reflector.SetJSONResponse(&opPurge, nil, http.StatusNoContent)
	_ = reflector.SetJSONResponse(&opPurge, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opPurge, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opPurge, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opPurge, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodPost, "/repos/{repo_ref}/purge", opPurge)

	opRestore := openapi3.Operation{}
	opRestore.WithTags("repository")
	opRestore.WithMapOfAnything(map[string]interface{}{"operationId": "restoreRepository"})
	opRestore.WithParameters(queryParameterDeletedAt)
	_ = reflector.SetRequest(&opRestore, new(restoreRequest), http.MethodPost)
	_ = reflector.SetJSONResponse(&opRestore, new(repo.RepositoryOutput), http.StatusOK)
	_ = reflector.SetJSONResponse(&opRestore, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opRestore, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opRestore, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opRestore, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opRestore, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodPost, "/repos/{repo_ref}/restore", opRestore)

	opMove := openapi3.Operation{}
	opMove.WithTags("repository")
	opMove.WithMapOfAnything(map[string]interface{}{"operationId": "moveRepository"})
	_ = reflector.SetRequest(&opMove, new(moveRepoRequest), http.MethodPost)
	_ = reflector.SetJSONResponse(&opMove, new(repo.RepositoryOutput), http.StatusOK)
	_ = reflector.SetJSONResponse(&opMove, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opMove, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opMove, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opMove, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodPost, "/repos/{repo_ref}/move", opMove)

	opUpdatePublicAccess := openapi3.Operation{}
	opUpdatePublicAccess.WithTags("repository")
	opUpdatePublicAccess.WithMapOfAnything(
		map[string]interface{}{"operationId": "updatePublicAccess"})
	_ = reflector.SetRequest(
		&opUpdatePublicAccess, new(updateRepoPublicAccessRequest), http.MethodPost)
	_ = reflector.SetJSONResponse(&opUpdatePublicAccess, new(repo.RepositoryOutput), http.StatusOK)
	_ = reflector.SetJSONResponse(&opUpdatePublicAccess, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opUpdatePublicAccess, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opUpdatePublicAccess, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opUpdatePublicAccess, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opUpdatePublicAccess, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(
		http.MethodPost, "/repos/{repo_ref}/public-access", opUpdatePublicAccess)

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

	opListPaths := openapi3.Operation{}
	opListPaths.WithTags("repository")
	opListPaths.WithMapOfAnything(map[string]interface{}{"operationId": "listPaths"})
	opListPaths.WithParameters(queryParameterGitRef, queryParameterIncludeDirectories)
	_ = reflector.SetRequest(&opListPaths, new(repoRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&opListPaths, new(repo.ListPathsOutput), http.StatusOK)
	_ = reflector.SetJSONResponse(&opListPaths, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opListPaths, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opListPaths, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opListPaths, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/repos/{repo_ref}/paths", opListPaths)

	opPathDetails := openapi3.Operation{}
	opPathDetails.WithTags("repository")
	opPathDetails.WithMapOfAnything(map[string]interface{}{"operationId": "pathDetails"})
	opPathDetails.WithParameters(queryParameterGitRef)
	_ = reflector.SetRequest(&opPathDetails, new(pathsDetailsRequest), http.MethodPost)
	_ = reflector.SetJSONResponse(&opPathDetails, new(repo.PathsDetailsOutput), http.StatusOK)
	_ = reflector.SetJSONResponse(&opPathDetails, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opPathDetails, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opPathDetails, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opPathDetails, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodPost, "/repos/{repo_ref}/path-details", opPathDetails)

	opGetRaw := openapi3.Operation{}
	opGetRaw.WithTags("repository")
	opGetRaw.WithMapOfAnything(map[string]interface{}{"operationId": "getRaw"})
	opGetRaw.WithParameters(queryParameterGitRef)
	_ = reflector.SetRequest(&opGetRaw, new(getContentRequest), http.MethodGet)
	// TODO: Figure out how to provide proper list of all potential mime types
	_ = reflector.SetStringResponse(&opGetRaw, http.StatusOK, "")
	_ = reflector.SetJSONResponse(&opGetRaw, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opGetRaw, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opGetRaw, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opGetRaw, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/repos/{repo_ref}/raw/{path}", opGetRaw)

	opGetBlame := openapi3.Operation{}
	opGetBlame.WithTags("repository")
	opGetBlame.WithMapOfAnything(map[string]interface{}{"operationId": "getBlame"})
	opGetBlame.WithParameters(queryParameterGitRef,
		queryParameterLineFrom, queryParameterLineTo)
	_ = reflector.SetRequest(&opGetBlame, new(getBlameRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&opGetBlame, []git.BlamePart{}, http.StatusOK)
	_ = reflector.SetJSONResponse(&opGetBlame, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opGetBlame, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opGetBlame, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opGetBlame, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/repos/{repo_ref}/blame/{path}", opGetBlame)

	opListCommits := openapi3.Operation{}
	opListCommits.WithTags("repository")
	opListCommits.WithMapOfAnything(map[string]interface{}{"operationId": "listCommits"})
	opListCommits.WithParameters(queryParameterGitRef, queryParameterAfterCommits, queryParameterPath,
		queryParameterSince, queryParameterUntil, queryParameterCommitter,
		QueryParameterPage, QueryParameterLimit, QueryParamIncludeStats)
	_ = reflector.SetRequest(&opListCommits, new(listCommitsRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&opListCommits, []types.ListCommitResponse{}, http.StatusOK)
	_ = reflector.SetJSONResponse(&opListCommits, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opListCommits, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opListCommits, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opListCommits, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/repos/{repo_ref}/commits", opListCommits)

	opGetCommit := openapi3.Operation{}
	opGetCommit.WithTags("repository")
	opGetCommit.WithMapOfAnything(map[string]interface{}{"operationId": "getCommit"})
	_ = reflector.SetRequest(&opGetCommit, new(GetCommitRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&opGetCommit, types.Commit{}, http.StatusOK)
	_ = reflector.SetJSONResponse(&opGetCommit, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opGetCommit, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opGetCommit, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opGetCommit, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/repos/{repo_ref}/commits/{commit_sha}", opGetCommit)

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
	_ = reflector.SetJSONResponse(&opCreateBranch, new(types.RulesViolations), http.StatusUnprocessableEntity)
	_ = reflector.Spec.AddOperation(http.MethodPost, "/repos/{repo_ref}/branches", opCreateBranch)

	opGetBranch := openapi3.Operation{}
	opGetBranch.WithTags("repository")
	opGetBranch.WithMapOfAnything(map[string]interface{}{"operationId": "getBranch"})
	_ = reflector.SetRequest(&opGetBranch, new(getBranchRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&opGetBranch, new(repo.Branch), http.StatusOK)
	_ = reflector.SetJSONResponse(&opGetBranch, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opGetBranch, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opGetBranch, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opGetBranch, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/repos/{repo_ref}/branches/{branch_name}", opGetBranch)

	opDeleteBranch := openapi3.Operation{}
	opDeleteBranch.WithTags("repository")
	opDeleteBranch.WithMapOfAnything(map[string]interface{}{"operationId": "deleteBranch"})
	opDeleteBranch.WithParameters(queryParameterBypassRules)
	_ = reflector.SetRequest(&opDeleteBranch, new(deleteBranchRequest), http.MethodDelete)
	_ = reflector.SetJSONResponse(&opDeleteBranch, nil, http.StatusNoContent)
	_ = reflector.SetJSONResponse(&opDeleteBranch, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opDeleteBranch, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opDeleteBranch, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opDeleteBranch, new(usererror.Error), http.StatusNotFound)
	_ = reflector.SetJSONResponse(&opDeleteBranch, new(types.RulesViolations), http.StatusUnprocessableEntity)
	_ = reflector.Spec.AddOperation(http.MethodDelete, "/repos/{repo_ref}/branches/{branch_name}", opDeleteBranch)

	opListBranches := openapi3.Operation{}
	opListBranches.WithTags("repository")
	opListBranches.WithMapOfAnything(map[string]interface{}{"operationId": "listBranches"})
	opListBranches.WithParameters(queryParameterIncludeCommit,
		queryParameterQueryBranches, queryParameterOrder, queryParameterSortBranch,
		QueryParameterPage, QueryParameterLimit)
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
		QueryParameterPage, QueryParameterLimit)
	_ = reflector.SetRequest(&opListTags, new(listTagsRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&opListTags, []repo.CommitTag{}, http.StatusOK)
	_ = reflector.SetJSONResponse(&opListTags, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opListTags, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opListTags, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opListTags, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/repos/{repo_ref}/tags", opListTags)

	opCreateTag := openapi3.Operation{}
	opCreateTag.WithTags("repository")
	opCreateTag.WithMapOfAnything(map[string]interface{}{"operationId": "createTag"})
	_ = reflector.SetRequest(&opCreateTag, new(createTagRequest), http.MethodPost)
	_ = reflector.SetJSONResponse(&opCreateTag, new(repo.CommitTag), http.StatusCreated)
	_ = reflector.SetJSONResponse(&opCreateTag, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opCreateTag, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opCreateTag, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opCreateTag, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opCreateTag, new(usererror.Error), http.StatusConflict)
	_ = reflector.SetJSONResponse(&opCreateTag, new(types.RulesViolations), http.StatusUnprocessableEntity)
	_ = reflector.Spec.AddOperation(http.MethodPost, "/repos/{repo_ref}/tags", opCreateTag)

	opDeleteTag := openapi3.Operation{}
	opDeleteTag.WithTags("repository")
	opDeleteTag.WithMapOfAnything(map[string]interface{}{"operationId": "deleteTag"})
	opDeleteTag.WithParameters(queryParameterBypassRules)
	_ = reflector.SetRequest(&opDeleteTag, new(deleteTagRequest), http.MethodDelete)
	_ = reflector.SetJSONResponse(&opDeleteTag, nil, http.StatusNoContent)
	_ = reflector.SetJSONResponse(&opDeleteTag, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opDeleteTag, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opDeleteTag, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opDeleteTag, new(usererror.Error), http.StatusNotFound)
	_ = reflector.SetJSONResponse(&opDeleteTag, new(usererror.Error), http.StatusConflict)
	_ = reflector.SetJSONResponse(&opDeleteTag, new(types.RulesViolations), http.StatusUnprocessableEntity)
	_ = reflector.Spec.AddOperation(http.MethodDelete, "/repos/{repo_ref}/tags/{tag_name}", opDeleteTag)

	opCommitFiles := openapi3.Operation{}
	opCommitFiles.WithTags("repository")
	opCommitFiles.WithMapOfAnything(map[string]interface{}{"operationId": "commitFiles"})
	_ = reflector.SetRequest(&opCommitFiles, new(commitFilesRequest), http.MethodPost)
	_ = reflector.SetJSONResponse(&opCommitFiles, types.CommitFilesResponse{}, http.StatusOK)
	_ = reflector.SetJSONResponse(&opCommitFiles, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opCommitFiles, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opCommitFiles, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opCommitFiles, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opCommitFiles, new(usererror.Error), http.StatusNotFound)
	_ = reflector.SetJSONResponse(&opCommitFiles, new(usererror.Error), http.StatusPreconditionFailed)
	_ = reflector.SetJSONResponse(&opCommitFiles, new(types.RulesViolations), http.StatusUnprocessableEntity)
	_ = reflector.Spec.AddOperation(http.MethodPost, "/repos/{repo_ref}/commits", opCommitFiles)

	opDiff := openapi3.Operation{}
	opDiff.WithTags("repository")
	opDiff.WithMapOfAnything(map[string]interface{}{"operationId": "rawDiff"})
	panicOnErr(reflector.SetRequest(&opDiff, new(getRawDiffRequest), http.MethodGet))
	panicOnErr(reflector.SetStringResponse(&opDiff, http.StatusOK, "text/plain"))
	panicOnErr(reflector.SetJSONResponse(&opDiff, []git.FileDiff{}, http.StatusOK))
	panicOnErr(reflector.SetJSONResponse(&opDiff, new(usererror.Error), http.StatusInternalServerError))
	panicOnErr(reflector.SetJSONResponse(&opDiff, new(usererror.Error), http.StatusUnauthorized))
	panicOnErr(reflector.SetJSONResponse(&opDiff, new(usererror.Error), http.StatusForbidden))
	panicOnErr(reflector.Spec.AddOperation(http.MethodGet, "/repos/{repo_ref}/diff/{range}", opDiff))

	opPostDiff := openapi3.Operation{}
	opPostDiff.WithTags("repository")
	opPostDiff.WithMapOfAnything(map[string]interface{}{"operationId": "rawDiffPost"})
	panicOnErr(reflector.SetRequest(&opPostDiff, new(postRawDiffRequest), http.MethodPost))
	panicOnErr(reflector.SetStringResponse(&opPostDiff, http.StatusOK, "text/plain"))
	panicOnErr(reflector.SetJSONResponse(&opPostDiff, []git.FileDiff{}, http.StatusOK))
	panicOnErr(reflector.SetJSONResponse(&opPostDiff, new(usererror.Error), http.StatusInternalServerError))
	panicOnErr(reflector.SetJSONResponse(&opPostDiff, new(usererror.Error), http.StatusUnauthorized))
	panicOnErr(reflector.SetJSONResponse(&opPostDiff, new(usererror.Error), http.StatusForbidden))
	panicOnErr(reflector.Spec.AddOperation(http.MethodPost, "/repos/{repo_ref}/diff/{range}", opPostDiff))

	opCommitDiff := openapi3.Operation{}
	opCommitDiff.WithTags("repository")
	opCommitDiff.WithMapOfAnything(map[string]interface{}{"operationId": "getCommitDiff"})
	_ = reflector.SetRequest(&opCommitDiff, new(GetCommitRequest), http.MethodGet)
	_ = reflector.SetStringResponse(&opCommitDiff, http.StatusOK, "text/plain")
	_ = reflector.SetJSONResponse(&opCommitDiff, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opCommitDiff, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opCommitDiff, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opCommitDiff, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/repos/{repo_ref}/commits/{commit_sha}/diff", opCommitDiff)

	opDiffStats := openapi3.Operation{}
	opDiffStats.WithTags("repository")
	opDiffStats.WithMapOfAnything(map[string]interface{}{"operationId": "diffStats"})
	_ = reflector.SetRequest(&opDiffStats, new(getRawDiffRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&opDiffStats, new(types.DiffStats), http.StatusOK)
	_ = reflector.SetJSONResponse(&opDiffStats, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opDiffStats, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opDiffStats, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/repos/{repo_ref}/diff-stats/{range}", opDiffStats)

	opMergeCheck := openapi3.Operation{}
	opMergeCheck.WithTags("repository")
	opMergeCheck.WithMapOfAnything(map[string]interface{}{"operationId": "mergeCheck"})
	_ = reflector.SetRequest(&opMergeCheck, new(getRawDiffRequest), http.MethodPost)
	_ = reflector.SetJSONResponse(&opMergeCheck, new(repo.MergeCheck), http.StatusOK)
	_ = reflector.SetJSONResponse(&opMergeCheck, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opMergeCheck, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opMergeCheck, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodPost, "/repos/{repo_ref}/merge-check/{range}", opMergeCheck)

	opRuleAdd := openapi3.Operation{}
	opRuleAdd.WithTags("repository")
	opRuleAdd.WithMapOfAnything(map[string]interface{}{"operationId": "ruleAdd"})
	_ = reflector.SetRequest(&opRuleAdd, struct {
		repoRequest
		repo.RuleCreateInput

		// overshadow "definition"
		Type       ruleType       `json:"type"`
		Definition ruleDefinition `json:"definition"`
	}{}, http.MethodPost)
	_ = reflector.SetJSONResponse(&opRuleAdd, rule{}, http.StatusCreated)
	_ = reflector.SetJSONResponse(&opRuleAdd, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opRuleAdd, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opRuleAdd, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opRuleAdd, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodPost, "/repos/{repo_ref}/rules", opRuleAdd)

	opRuleDelete := openapi3.Operation{}
	opRuleDelete.WithTags("repository")
	opRuleDelete.WithMapOfAnything(map[string]interface{}{"operationId": "ruleDelete"})
	_ = reflector.SetRequest(&opRuleDelete, struct {
		repoRequest
		RuleIdentifier string `path:"rule_identifier"`
	}{}, http.MethodDelete)
	_ = reflector.SetJSONResponse(&opRuleDelete, nil, http.StatusNoContent)
	_ = reflector.SetJSONResponse(&opRuleDelete, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opRuleDelete, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opRuleDelete, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opRuleDelete, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodDelete, "/repos/{repo_ref}/rules/{rule_identifier}", opRuleDelete)

	opRuleUpdate := openapi3.Operation{}
	opRuleUpdate.WithTags("repository")
	opRuleUpdate.WithMapOfAnything(map[string]interface{}{"operationId": "ruleUpdate"})
	_ = reflector.SetRequest(&opRuleUpdate, &struct {
		repoRequest
		Identifier string `path:"rule_identifier"`
		repo.RuleUpdateInput

		// overshadow Type and Definition to enable oneof.
		Type       ruleType       `json:"type"`
		Definition ruleDefinition `json:"definition"`
	}{}, http.MethodPatch)
	_ = reflector.SetJSONResponse(&opRuleUpdate, rule{}, http.StatusOK)
	_ = reflector.SetJSONResponse(&opRuleUpdate, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opRuleUpdate, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opRuleUpdate, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opRuleUpdate, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodPatch, "/repos/{repo_ref}/rules/{rule_identifier}", opRuleUpdate)

	opRuleList := openapi3.Operation{}
	opRuleList.WithTags("repository")
	opRuleList.WithMapOfAnything(map[string]interface{}{"operationId": "ruleList"})
	opRuleList.WithParameters(
		queryParameterQueryRuleList,
		queryParameterOrder, queryParameterSortRuleList,
		QueryParameterPage, QueryParameterLimit)
	_ = reflector.SetRequest(&opRuleList, &struct {
		repoRequest
	}{}, http.MethodGet)
	_ = reflector.SetJSONResponse(&opRuleList, []rule{}, http.StatusOK)
	_ = reflector.SetJSONResponse(&opRuleList, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opRuleList, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opRuleList, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opRuleList, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/repos/{repo_ref}/rules", opRuleList)

	opRuleGet := openapi3.Operation{}
	opRuleGet.WithTags("repository")
	opRuleGet.WithMapOfAnything(map[string]interface{}{"operationId": "ruleGet"})
	_ = reflector.SetRequest(&opRuleGet, &struct {
		repoRequest
		Identifier string `path:"rule_identifier"`
	}{}, http.MethodGet)
	_ = reflector.SetJSONResponse(&opRuleGet, rule{}, http.StatusOK)
	_ = reflector.SetJSONResponse(&opRuleGet, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opRuleGet, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opRuleGet, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opRuleGet, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/repos/{repo_ref}/rules/{rule_identifier}", opRuleGet)

	opCodeOwnerValidate := openapi3.Operation{}
	opCodeOwnerValidate.WithTags("repository")
	opCodeOwnerValidate.WithMapOfAnything(map[string]interface{}{"operationId": "codeOwnersValidate"})
	opCodeOwnerValidate.WithParameters(queryParameterGitRef)
	_ = reflector.SetRequest(&opCodeOwnerValidate, new(codeOwnersValidate), http.MethodGet)
	_ = reflector.SetJSONResponse(&opCodeOwnerValidate, nil, http.StatusOK)
	_ = reflector.SetJSONResponse(&opCodeOwnerValidate, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opCodeOwnerValidate, new(usererror.Error), http.StatusUnprocessableEntity)
	_ = reflector.SetJSONResponse(&opCodeOwnerValidate, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opCodeOwnerValidate, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opCodeOwnerValidate, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/repos/{repo_ref}/codeowners/validate", opCodeOwnerValidate)

	opSettingsSecurityUpdate := openapi3.Operation{}
	opSettingsSecurityUpdate.WithTags("repository")
	opSettingsSecurityUpdate.WithMapOfAnything(
		map[string]interface{}{"operationId": "updateSecuritySettings"})
	_ = reflector.SetRequest(
		&opSettingsSecurityUpdate, new(securitySettingsRequest), http.MethodPatch)
	_ = reflector.SetJSONResponse(&opSettingsSecurityUpdate, new(reposettings.SecuritySettings), http.StatusOK)
	_ = reflector.SetJSONResponse(&opSettingsSecurityUpdate, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opSettingsSecurityUpdate, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opSettingsSecurityUpdate, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opSettingsSecurityUpdate, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opSettingsSecurityUpdate, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(
		http.MethodPatch, "/repos/{repo_ref}/settings/security", opSettingsSecurityUpdate)

	opSettingsSecurityFind := openapi3.Operation{}
	opSettingsSecurityFind.WithTags("repository")
	opSettingsSecurityFind.WithMapOfAnything(
		map[string]interface{}{"operationId": "findSecuritySettings"})
	_ = reflector.SetRequest(&opSettingsSecurityFind, new(repoRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&opSettingsSecurityFind, new(reposettings.SecuritySettings), http.StatusOK)
	_ = reflector.SetJSONResponse(&opSettingsSecurityFind, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opSettingsSecurityFind, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opSettingsSecurityFind, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opSettingsSecurityFind, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opSettingsSecurityFind, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(
		http.MethodGet, "/repos/{repo_ref}/settings/security", opSettingsSecurityFind)

	opSettingsGeneralUpdate := openapi3.Operation{}
	opSettingsGeneralUpdate.WithTags("repository")
	opSettingsGeneralUpdate.WithMapOfAnything(
		map[string]interface{}{"operationId": "updateGeneralSettings"})
	_ = reflector.SetRequest(
		&opSettingsGeneralUpdate, new(generalSettingsRequest), http.MethodPatch)
	_ = reflector.SetJSONResponse(&opSettingsGeneralUpdate, new(reposettings.GeneralSettings), http.StatusOK)
	_ = reflector.SetJSONResponse(&opSettingsGeneralUpdate, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opSettingsGeneralUpdate, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opSettingsGeneralUpdate, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opSettingsGeneralUpdate, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opSettingsGeneralUpdate, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(
		http.MethodPatch, "/repos/{repo_ref}/settings/general", opSettingsGeneralUpdate)

	opSettingsGeneralFind := openapi3.Operation{}
	opSettingsGeneralFind.WithTags("repository")
	opSettingsGeneralFind.WithMapOfAnything(
		map[string]interface{}{"operationId": "findGeneralSettings"})
	_ = reflector.SetRequest(&opSettingsGeneralFind, new(repoRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&opSettingsGeneralFind, new(reposettings.GeneralSettings), http.StatusOK)
	_ = reflector.SetJSONResponse(&opSettingsGeneralFind, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opSettingsGeneralFind, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opSettingsGeneralFind, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opSettingsGeneralFind, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opSettingsGeneralFind, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(
		http.MethodGet, "/repos/{repo_ref}/settings/general", opSettingsGeneralFind)

	opArchive := openapi3.Operation{}
	opArchive.WithTags("repository")
	opArchive.WithMapOfAnything(map[string]interface{}{"operationId": "archive"})
	opArchive.WithParameters(
		queryParamArchivePaths,
		queryParamArchivePrefix,
		queryParamArchiveAttributes,
		queryParamArchiveTime,
		queryParamArchiveCompression,
	)
	_ = reflector.SetRequest(&opArchive, new(archiveRequest), http.MethodGet)
	_ = reflector.SetStringResponse(&opArchive, http.StatusOK, "application/zip")
	_ = reflector.SetStringResponse(&opArchive, http.StatusOK, "application/tar")
	_ = reflector.SetStringResponse(&opArchive, http.StatusOK, "application/gzip")
	_ = reflector.SetJSONResponse(&opArchive, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opArchive, new(usererror.Error), http.StatusUnprocessableEntity)
	_ = reflector.SetJSONResponse(&opArchive, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opArchive, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opArchive, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/repos/{repo_ref}/archive/{git_ref}.{format}", opArchive)

	opSummary := openapi3.Operation{}
	opSummary.WithTags("repository")
	opSummary.WithMapOfAnything(
		map[string]interface{}{"operationId": "summary"})
	_ = reflector.SetRequest(&opSummary, new(repoRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&opSummary, new(types.RepositorySummary), http.StatusOK)
	_ = reflector.SetJSONResponse(&opSummary, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opSummary, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opSummary, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opSummary, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opSummary, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/repos/{repo_ref}/summary", opSummary)

	opDefineLabel := openapi3.Operation{}
	opDefineLabel.WithTags("repository")
	opDefineLabel.WithMapOfAnything(
		map[string]interface{}{"operationId": "defineRepoLabel"})
	_ = reflector.SetRequest(&opDefineLabel, &struct {
		repoRequest
		labelRequest
	}{}, http.MethodPost)
	_ = reflector.SetJSONResponse(&opDefineLabel, new(types.Label), http.StatusCreated)
	_ = reflector.SetJSONResponse(&opDefineLabel, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opDefineLabel, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opDefineLabel, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opDefineLabel, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opDefineLabel, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodPost, "/repos/{repo_ref}/labels", opDefineLabel)

	opSaveLabel := openapi3.Operation{}
	opSaveLabel.WithTags("repository")
	opSaveLabel.WithMapOfAnything(
		map[string]interface{}{"operationId": "saveRepoLabel"})
	_ = reflector.SetRequest(&opSaveLabel, &struct {
		repoRequest
		types.SaveInput
	}{}, http.MethodPut)
	_ = reflector.SetJSONResponse(&opSaveLabel, new(types.LabelWithValues), http.StatusOK)
	_ = reflector.SetJSONResponse(&opSaveLabel, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opSaveLabel, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opSaveLabel, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opSaveLabel, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opSaveLabel, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodPut, "/repos/{repo_ref}/labels", opSaveLabel)

	opListLabels := openapi3.Operation{}
	opListLabels.WithTags("repository")
	opListLabels.WithMapOfAnything(
		map[string]interface{}{"operationId": "listRepoLabels"})
	opListLabels.WithParameters(
		QueryParameterPage, QueryParameterLimit, queryParameterInherited, queryParameterQueryLabel)
	_ = reflector.SetRequest(&opListLabels, new(repoRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&opListLabels, new([]*types.Label), http.StatusOK)
	_ = reflector.SetJSONResponse(&opListLabels, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opListLabels, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opListLabels, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opListLabels, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opListLabels, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/repos/{repo_ref}/labels", opListLabels)

	opDeleteLabel := openapi3.Operation{}
	opDeleteLabel.WithTags("repository")
	opDeleteLabel.WithMapOfAnything(
		map[string]interface{}{"operationId": "deleteRepoLabel"})
	_ = reflector.SetRequest(&opDeleteLabel, &struct {
		repoRequest
		Key string `path:"key"`
	}{}, http.MethodDelete)
	_ = reflector.SetJSONResponse(&opDeleteLabel, nil, http.StatusNoContent)
	_ = reflector.SetJSONResponse(&opDeleteLabel, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opDeleteLabel, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opDeleteLabel, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opDeleteLabel, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opDeleteLabel, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(
		http.MethodDelete, "/repos/{repo_ref}/labels/{key}", opDeleteLabel)

	opUpdateLabel := openapi3.Operation{}
	opUpdateLabel.WithTags("repository")
	opUpdateLabel.WithMapOfAnything(
		map[string]interface{}{"operationId": "updateRepoLabel"})
	_ = reflector.SetRequest(&opUpdateLabel, &struct {
		repoRequest
		labelRequest
		Key string `path:"key"`
	}{}, http.MethodPatch)
	_ = reflector.SetJSONResponse(&opUpdateLabel, new(types.Label), http.StatusOK)
	_ = reflector.SetJSONResponse(&opUpdateLabel, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opUpdateLabel, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opUpdateLabel, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opUpdateLabel, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opUpdateLabel, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodPatch, "/repos/{repo_ref}/labels/{key}", opUpdateLabel)

	opDefineLabelValue := openapi3.Operation{}
	opDefineLabelValue.WithTags("repository")
	opDefineLabelValue.WithMapOfAnything(
		map[string]interface{}{"operationId": "defineRepoLabelValue"})
	_ = reflector.SetRequest(&opDefineLabelValue, &struct {
		repoRequest
		labelValueRequest
		Key string `path:"key"`
	}{}, http.MethodPost)
	_ = reflector.SetJSONResponse(&opDefineLabelValue, new(types.LabelValue), http.StatusCreated)
	_ = reflector.SetJSONResponse(&opDefineLabelValue, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opDefineLabelValue, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opDefineLabelValue, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opDefineLabelValue, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opDefineLabelValue, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodPost,
		"/repos/{repo_ref}/labels/{key}/values", opDefineLabelValue)

	opListLabelValues := openapi3.Operation{}
	opListLabelValues.WithTags("repository")
	opListLabelValues.WithMapOfAnything(
		map[string]interface{}{"operationId": "listRepoLabelValues"})
	_ = reflector.SetRequest(&opListLabelValues, &struct {
		repoRequest
		Key string `path:"key"`
	}{}, http.MethodGet)
	_ = reflector.SetJSONResponse(&opListLabelValues, new([]*types.LabelValue), http.StatusOK)
	_ = reflector.SetJSONResponse(&opListLabelValues, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opListLabelValues, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opListLabelValues, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opListLabelValues, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opListLabelValues, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet,
		"/repos/{repo_ref}/labels/{key}/values", opListLabelValues)

	opDeleteLabelValue := openapi3.Operation{}
	opDeleteLabelValue.WithTags("repository")
	opDeleteLabelValue.WithMapOfAnything(
		map[string]interface{}{"operationId": "deleteRepoLabelValue"})
	_ = reflector.SetRequest(&opDeleteLabelValue, &struct {
		repoRequest
		Key   string `path:"key"`
		Value string `path:"value"`
	}{}, http.MethodDelete)
	_ = reflector.SetJSONResponse(&opDeleteLabelValue, nil, http.StatusNoContent)
	_ = reflector.SetJSONResponse(&opDeleteLabelValue, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opDeleteLabelValue, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opDeleteLabelValue, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opDeleteLabelValue, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opDeleteLabelValue, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(
		http.MethodDelete, "/repos/{repo_ref}/labels/{key}/values/{value}", opDeleteLabelValue)

	opUpdateLabelValue := openapi3.Operation{}
	opUpdateLabelValue.WithTags("repository")
	opUpdateLabelValue.WithMapOfAnything(
		map[string]interface{}{"operationId": "updateRepoLabelValue"})
	_ = reflector.SetRequest(&opUpdateLabelValue, &struct {
		repoRequest
		labelValueRequest
		Key   string `path:"key"`
		Value string `path:"value"`
	}{}, http.MethodPatch)
	_ = reflector.SetJSONResponse(&opUpdateLabelValue, new(types.LabelValue), http.StatusOK)
	_ = reflector.SetJSONResponse(&opUpdateLabelValue, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opUpdateLabelValue, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opUpdateLabelValue, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opUpdateLabelValue, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opUpdateLabelValue, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodPatch,
		"/repos/{repo_ref}/labels/{key}/values/{value}", opUpdateLabelValue)
}
