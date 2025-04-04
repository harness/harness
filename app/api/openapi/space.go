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
	"github.com/harness/gitness/app/api/controller/space"
	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/gotidy/ptr"
	"github.com/swaggest/openapi-go/openapi3"
)

type createSpaceRequest struct {
	space.CreateInput
}

type spaceRequest struct {
	Ref string `path:"space_ref"`
}

type updateSpaceRequest struct {
	spaceRequest
	space.UpdateInput
}

type updateSpacePublicAccessRequest struct {
	spaceRequest
	space.UpdatePublicAccessInput
}
type moveSpaceRequest struct {
	spaceRequest
	space.MoveInput
}

type exportSpaceRequest struct {
	spaceRequest
	space.ExportInput
}

type restoreSpaceRequest struct {
	spaceRequest
	space.RestoreInput
}

type importRepositoriesRequest struct {
	spaceRequest
	space.ImportRepositoriesInput
}

var queryParameterSortRepo = openapi3.ParameterOrRef{
	Parameter: &openapi3.Parameter{
		Name:        request.QueryParamSort,
		In:          openapi3.ParameterInQuery,
		Description: ptr.String("The data by which the repositories are sorted."),
		Required:    ptr.Bool(false),
		Schema: &openapi3.SchemaOrRef{
			Schema: &openapi3.Schema{
				Type:    ptrSchemaType(openapi3.SchemaTypeString),
				Default: ptrptr(enum.RepoAttrIdentifier.String()),
				Enum: []interface{}{
					ptr.String(enum.RepoAttrIdentifier.String()),
					ptr.String(enum.RepoAttrCreated.String()),
					ptr.String(enum.RepoAttrUpdated.String()),
				},
			},
		},
	},
}

var queryParameterQueryRepo = openapi3.ParameterOrRef{
	Parameter: &openapi3.Parameter{
		Name:        request.QueryParamQuery,
		In:          openapi3.ParameterInQuery,
		Description: ptr.String("The substring which is used to filter the repositories by their path name."),
		Required:    ptr.Bool(false),
		Schema: &openapi3.SchemaOrRef{
			Schema: &openapi3.Schema{
				Type: ptrSchemaType(openapi3.SchemaTypeString),
			},
		},
	},
}

var queryParameterSortSpace = openapi3.ParameterOrRef{
	Parameter: &openapi3.Parameter{
		Name:        request.QueryParamSort,
		In:          openapi3.ParameterInQuery,
		Description: ptr.String("The data by which the spaces are sorted."),
		Required:    ptr.Bool(false),
		Schema: &openapi3.SchemaOrRef{
			Schema: &openapi3.Schema{
				Type:    ptrSchemaType(openapi3.SchemaTypeString),
				Default: ptrptr(enum.SpaceAttrIdentifier.String()),
				Enum: []interface{}{
					ptr.String(enum.SpaceAttrIdentifier.String()),
					ptr.String(enum.SpaceAttrCreated.String()),
					ptr.String(enum.SpaceAttrUpdated.String()),
				},
			},
		},
	},
}

var queryParameterQuerySpace = openapi3.ParameterOrRef{
	Parameter: &openapi3.Parameter{
		Name:        request.QueryParamQuery,
		In:          openapi3.ParameterInQuery,
		Description: ptr.String("The substring which is used to filter the spaces by their path name."),
		Required:    ptr.Bool(false),
		Schema: &openapi3.SchemaOrRef{
			Schema: &openapi3.Schema{
				Type: ptrSchemaType(openapi3.SchemaTypeString),
			},
		},
	},
}

var queryParameterMembershipUsers = openapi3.ParameterOrRef{
	Parameter: &openapi3.Parameter{
		Name:        request.QueryParamQuery,
		In:          openapi3.ParameterInQuery,
		Description: ptr.String("The substring by which the space members are filtered."),
		Required:    ptr.Bool(false),
		Schema: &openapi3.SchemaOrRef{
			Schema: &openapi3.Schema{
				Type: ptrSchemaType(openapi3.SchemaTypeString),
			},
		},
	},
}

var queryParameterSortMembershipUsers = openapi3.ParameterOrRef{
	Parameter: &openapi3.Parameter{
		Name:        request.QueryParamSort,
		In:          openapi3.ParameterInQuery,
		Description: ptr.String("The field by which the space members are sorted."),
		Required:    ptr.Bool(false),
		Schema: &openapi3.SchemaOrRef{
			Schema: &openapi3.Schema{
				Type:    ptrSchemaType(openapi3.SchemaTypeString),
				Default: ptrptr(enum.MembershipUserSortName),
				Enum:    enum.MembershipUserSort("").Enum(),
			},
		},
	},
}

var queryParameterQueryExecution = openapi3.ParameterOrRef{
	Parameter: &openapi3.Parameter{
		Name:        request.QueryParamQuery,
		In:          openapi3.ParameterInQuery,
		Description: ptr.String("The substring which is used to filter the execution by their pipeline names."),
		Required:    ptr.Bool(false),
		Schema: &openapi3.SchemaOrRef{
			Schema: &openapi3.Schema{
				Type: ptrSchemaType(openapi3.SchemaTypeString),
			},
		},
	},
}

var queryParameterSortExecution = openapi3.ParameterOrRef{
	Parameter: &openapi3.Parameter{
		Name:        request.QueryParamSort,
		In:          openapi3.ParameterInQuery,
		Description: ptr.String("The data by which the executions are sorted."),
		Required:    ptr.Bool(false),
		Schema: &openapi3.SchemaOrRef{
			Schema: &openapi3.Schema{
				Type:    ptrSchemaType(openapi3.SchemaTypeString),
				Default: ptrptr(enum.ExecutionSortStarted),
				Enum:    enum.ExecutionSort("").Enum(),
			},
		},
	},
}

var queryParameterPipelineIdentifier = openapi3.ParameterOrRef{
	Parameter: &openapi3.Parameter{
		Name:        request.QueryParamPipelineIdentifier,
		In:          openapi3.ParameterInQuery,
		Description: ptr.String("The pipeline identifier whose executions are to be returned"),
		Required:    ptr.Bool(false),
		Schema: &openapi3.SchemaOrRef{
			Schema: &openapi3.Schema{
				Type: ptrSchemaType(openapi3.SchemaTypeString),
			},
		},
	},
}

//nolint:funlen // api spec generation no need for checking func complexity
func spaceOperations(reflector *openapi3.Reflector) {
	opCreate := openapi3.Operation{}
	opCreate.WithTags("space")
	opCreate.WithMapOfAnything(map[string]interface{}{"operationId": "createSpace"})
	_ = reflector.SetRequest(&opCreate, new(createSpaceRequest), http.MethodPost)
	_ = reflector.SetJSONResponse(&opCreate, new(space.SpaceOutput), http.StatusCreated)
	_ = reflector.SetJSONResponse(&opCreate, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opCreate, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opCreate, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opCreate, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodPost, "/spaces", opCreate)

	opImport := openapi3.Operation{}
	opImport.WithTags("space")
	opImport.WithMapOfAnything(map[string]interface{}{"operationId": "importSpace"})
	_ = reflector.SetRequest(&opImport, new(space.ImportInput), http.MethodPost)
	_ = reflector.SetJSONResponse(&opImport, new(space.SpaceOutput), http.StatusCreated)
	_ = reflector.SetJSONResponse(&opImport, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opImport, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opImport, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opImport, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodPost, "/spaces/import", opImport)

	opImportRepositories := openapi3.Operation{}
	opImportRepositories.WithTags("space")
	opImportRepositories.WithMapOfAnything(map[string]interface{}{"operationId": "importSpaceRepositories"})
	_ = reflector.SetRequest(&opImportRepositories, new(importRepositoriesRequest), http.MethodPost)
	_ = reflector.SetJSONResponse(&opImportRepositories, new(space.ImportRepositoriesOutput), http.StatusOK)
	_ = reflector.SetJSONResponse(&opImportRepositories, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opImportRepositories, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opImportRepositories, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opImportRepositories, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodPost, "/spaces/{space_ref}/import", opImportRepositories)

	opExport := openapi3.Operation{}
	opExport.WithTags("space")
	opExport.WithMapOfAnything(map[string]interface{}{"operationId": "exportSpace"})
	_ = reflector.SetRequest(&opExport, new(exportSpaceRequest), http.MethodPost)
	_ = reflector.SetJSONResponse(&opExport, nil, http.StatusAccepted)
	_ = reflector.SetJSONResponse(&opExport, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opExport, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opExport, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opExport, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodPost, "/spaces/{space_ref}/export", opExport)

	opExportProgress := openapi3.Operation{}
	opExportProgress.WithTags("space")
	opExportProgress.WithMapOfAnything(map[string]interface{}{"operationId": "exportProgressSpace"})
	_ = reflector.SetRequest(&opExportProgress, new(spaceRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&opExportProgress, new(space.ExportProgressOutput), http.StatusOK)
	_ = reflector.SetJSONResponse(&opExportProgress, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opExportProgress, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opExportProgress, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opExportProgress, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/spaces/{space_ref}/export-progress", opExportProgress)

	opGet := openapi3.Operation{}
	opGet.WithTags("space")
	opGet.WithMapOfAnything(map[string]interface{}{"operationId": "getSpace"})
	_ = reflector.SetRequest(&opGet, new(spaceRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&opGet, new(space.SpaceOutput), http.StatusOK)
	_ = reflector.SetJSONResponse(&opGet, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opGet, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opGet, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opGet, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/spaces/{space_ref}", opGet)

	opUpdate := openapi3.Operation{}
	opUpdate.WithTags("space")
	opUpdate.WithMapOfAnything(map[string]interface{}{"operationId": "updateSpace"})
	_ = reflector.SetRequest(&opUpdate, new(updateSpaceRequest), http.MethodPatch)
	_ = reflector.SetJSONResponse(&opUpdate, new(space.SpaceOutput), http.StatusOK)
	_ = reflector.SetJSONResponse(&opUpdate, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opUpdate, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opUpdate, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opUpdate, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opUpdate, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodPatch, "/spaces/{space_ref}", opUpdate)

	opUpdatePublicAccess := openapi3.Operation{}
	opUpdatePublicAccess.WithTags("space")
	opUpdatePublicAccess.WithMapOfAnything(
		map[string]interface{}{"operationId": "updateSpacePublicAccess"})
	_ = reflector.SetRequest(
		&opUpdatePublicAccess, new(updateSpacePublicAccessRequest), http.MethodPost)
	_ = reflector.SetJSONResponse(&opUpdatePublicAccess, new(space.SpaceOutput), http.StatusOK)
	_ = reflector.SetJSONResponse(&opUpdatePublicAccess, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opUpdatePublicAccess, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opUpdatePublicAccess, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opUpdatePublicAccess, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opUpdatePublicAccess, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(
		http.MethodPost, "/spaces/{space_ref}/public-access", opUpdatePublicAccess)

	opDelete := openapi3.Operation{}
	opDelete.WithTags("space")
	opDelete.WithMapOfAnything(map[string]interface{}{"operationId": "deleteSpace"})
	_ = reflector.SetRequest(&opDelete, new(spaceRequest), http.MethodDelete)
	_ = reflector.SetJSONResponse(&opDelete, new(space.SoftDeleteResponse), http.StatusOK)
	_ = reflector.SetJSONResponse(&opDelete, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opDelete, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opDelete, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opDelete, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodDelete, "/spaces/{space_ref}", opDelete)

	opPurge := openapi3.Operation{}
	opPurge.WithTags("space")
	opPurge.WithMapOfAnything(map[string]interface{}{"operationId": "purgeSpace"})
	opPurge.WithParameters(queryParameterDeletedAt)
	_ = reflector.SetRequest(&opPurge, new(spaceRequest), http.MethodPost)
	_ = reflector.SetJSONResponse(&opPurge, nil, http.StatusNoContent)
	_ = reflector.SetJSONResponse(&opPurge, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opPurge, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opPurge, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opPurge, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodPost, "/spaces/{space_ref}/purge", opPurge)

	opRestore := openapi3.Operation{}
	opRestore.WithTags("space")
	opRestore.WithMapOfAnything(map[string]interface{}{"operationId": "restoreSpace"})
	opRestore.WithParameters(queryParameterDeletedAt)
	_ = reflector.SetRequest(&opRestore, new(restoreSpaceRequest), http.MethodPost)
	_ = reflector.SetJSONResponse(&opRestore, new(space.SpaceOutput), http.StatusOK)
	_ = reflector.SetJSONResponse(&opRestore, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opRestore, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opRestore, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opRestore, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opRestore, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodPost, "/spaces/{space_ref}/restore", opRestore)

	opMove := openapi3.Operation{}
	opMove.WithTags("space")
	opMove.WithMapOfAnything(map[string]interface{}{"operationId": "moveSpace"})
	_ = reflector.SetRequest(&opMove, new(moveSpaceRequest), http.MethodPost)
	_ = reflector.SetJSONResponse(&opMove, new(space.SpaceOutput), http.StatusOK)
	_ = reflector.SetJSONResponse(&opMove, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opMove, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opMove, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opMove, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodPost, "/spaces/{space_ref}/move", opMove)

	opSpaces := openapi3.Operation{}
	opSpaces.WithTags("space")
	opSpaces.WithMapOfAnything(map[string]interface{}{"operationId": "listSpaces"})
	opSpaces.WithParameters(QueryParameterPage, QueryParameterLimit)
	opSpaces.WithParameters(queryParameterQuerySpace, queryParameterSortSpace, queryParameterOrder,
		QueryParameterPage, QueryParameterLimit)
	_ = reflector.SetRequest(&opSpaces, new(spaceRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&opSpaces, []space.SpaceOutput{}, http.StatusOK)
	_ = reflector.SetJSONResponse(&opSpaces, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opSpaces, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opSpaces, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opSpaces, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/spaces/{space_ref}/spaces", opSpaces)

	opRepos := openapi3.Operation{}
	opRepos.WithTags("space")
	opRepos.WithMapOfAnything(map[string]interface{}{"operationId": "listRepos"})
	opRepos.WithParameters(queryParameterQueryRepo, queryParameterSortRepo, queryParameterOrder,
		QueryParameterPage, QueryParameterLimit)
	_ = reflector.SetRequest(&opRepos, new(spaceRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&opRepos, []repo.RepositoryOutput{}, http.StatusOK)
	_ = reflector.SetJSONResponse(&opRepos, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opRepos, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opRepos, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opRepos, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/spaces/{space_ref}/repos", opRepos)

	opPipelines := openapi3.Operation{}
	opPipelines.WithTags("space")
	opPipelines.WithMapOfAnything(map[string]interface{}{"operationId": "listSpacePipelines"})
	opPipelines.WithParameters(queryParameterQueryPipeline, QueryParameterPage,
		QueryParameterLimit, queryParameterLastExecutions)
	_ = reflector.SetRequest(&opPipelines, new(spaceRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&opPipelines, []types.Pipeline{}, http.StatusOK)
	_ = reflector.SetJSONResponse(&opPipelines, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opPipelines, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opPipelines, new(usererror.Error), http.StatusNotFound)
	_ = reflector.SetJSONResponse(&opPipelines, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/spaces/{space_ref}/pipelines", opPipelines)

	opExecutions := openapi3.Operation{}
	opExecutions.WithTags("space")
	opExecutions.WithMapOfAnything(map[string]interface{}{"operationId": "listSpaceExecutions"})
	opExecutions.WithParameters(queryParameterQueryExecution, QueryParameterPage, QueryParameterLimit,
		queryParameterSortExecution, queryParameterOrder, queryParameterPipelineIdentifier)
	_ = reflector.SetRequest(&opExecutions, new(spaceRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&opExecutions, []types.Execution{}, http.StatusOK)
	_ = reflector.SetJSONResponse(&opExecutions, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opExecutions, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opExecutions, new(usererror.Error), http.StatusNotFound)
	_ = reflector.SetJSONResponse(&opExecutions, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/spaces/{space_ref}/executions", opExecutions)

	opTemplates := openapi3.Operation{}
	opTemplates.WithTags("space")
	opTemplates.WithMapOfAnything(map[string]interface{}{"operationId": "listTemplates"})
	opTemplates.WithParameters(queryParameterQueryRepo, QueryParameterPage, QueryParameterLimit)
	_ = reflector.SetRequest(&opTemplates, new(spaceRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&opTemplates, []types.Template{}, http.StatusOK)
	_ = reflector.SetJSONResponse(&opTemplates, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opTemplates, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opTemplates, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opTemplates, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/spaces/{space_ref}/templates", opTemplates)

	opConnectors := openapi3.Operation{}
	opConnectors.WithTags("space")
	opConnectors.WithMapOfAnything(map[string]interface{}{"operationId": "listConnectors"})
	opConnectors.WithParameters(queryParameterQueryRepo, QueryParameterPage, QueryParameterLimit)
	_ = reflector.SetRequest(&opConnectors, new(spaceRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&opConnectors, []types.Connector{}, http.StatusOK)
	_ = reflector.SetJSONResponse(&opConnectors, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opConnectors, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opConnectors, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opConnectors, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/spaces/{space_ref}/connectors", opConnectors)

	opSecrets := openapi3.Operation{}
	opSecrets.WithTags("space")
	opSecrets.WithMapOfAnything(map[string]interface{}{"operationId": "listSecrets"})
	opSecrets.WithParameters(queryParameterQueryRepo, QueryParameterPage, QueryParameterLimit)
	_ = reflector.SetRequest(&opSecrets, new(spaceRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&opSecrets, []types.Secret{}, http.StatusOK)
	_ = reflector.SetJSONResponse(&opSecrets, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opSecrets, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opSecrets, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opSecrets, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/spaces/{space_ref}/secrets", opSecrets)

	opServiceAccounts := openapi3.Operation{}
	opServiceAccounts.WithTags("space")
	opServiceAccounts.WithMapOfAnything(map[string]interface{}{"operationId": "listServiceAccounts"})
	_ = reflector.SetRequest(&opServiceAccounts, new(spaceRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&opServiceAccounts, []types.ServiceAccount{}, http.StatusOK)
	_ = reflector.SetJSONResponse(&opServiceAccounts, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opServiceAccounts, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opServiceAccounts, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opServiceAccounts, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/spaces/{space_ref}/service-accounts", opServiceAccounts)

	opMembershipAdd := openapi3.Operation{}
	opMembershipAdd.WithTags("space")
	opMembershipAdd.WithMapOfAnything(map[string]interface{}{"operationId": "membershipAdd"})
	_ = reflector.SetRequest(&opMembershipAdd, struct {
		spaceRequest
		space.MembershipAddInput
	}{}, http.MethodPost)
	_ = reflector.SetJSONResponse(&opMembershipAdd, &types.MembershipUser{}, http.StatusCreated)
	_ = reflector.SetJSONResponse(&opMembershipAdd, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opMembershipAdd, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opMembershipAdd, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opMembershipAdd, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodPost, "/spaces/{space_ref}/members", opMembershipAdd)

	opMembershipDelete := openapi3.Operation{}
	opMembershipDelete.WithTags("space")
	opMembershipDelete.WithMapOfAnything(map[string]interface{}{"operationId": "membershipDelete"})
	_ = reflector.SetRequest(&opMembershipDelete, struct {
		spaceRequest
		UserUID string `path:"user_uid"`
	}{}, http.MethodDelete)
	_ = reflector.SetJSONResponse(&opMembershipDelete, nil, http.StatusNoContent)
	_ = reflector.SetJSONResponse(&opMembershipDelete, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opMembershipDelete, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opMembershipDelete, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opMembershipDelete, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodDelete, "/spaces/{space_ref}/members/{user_uid}", opMembershipDelete)

	opMembershipUpdate := openapi3.Operation{}
	opMembershipUpdate.WithTags("space")
	opMembershipUpdate.WithMapOfAnything(map[string]interface{}{"operationId": "membershipUpdate"})
	_ = reflector.SetRequest(&opMembershipUpdate, &struct {
		spaceRequest
		UserUID string `path:"user_uid"`
		space.MembershipUpdateInput
	}{}, http.MethodPatch)
	_ = reflector.SetJSONResponse(&opMembershipUpdate, &types.MembershipUser{}, http.StatusOK)
	_ = reflector.SetJSONResponse(&opMembershipUpdate, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opMembershipUpdate, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opMembershipUpdate, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opMembershipUpdate, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodPatch, "/spaces/{space_ref}/members/{user_uid}", opMembershipUpdate)

	opMembershipList := openapi3.Operation{}
	opMembershipList.WithTags("space")
	opMembershipList.WithMapOfAnything(map[string]interface{}{"operationId": "membershipList"})
	opMembershipList.WithParameters(
		queryParameterMembershipUsers,
		queryParameterOrder, queryParameterSortMembershipUsers,
		QueryParameterPage, QueryParameterLimit)
	_ = reflector.SetRequest(&opMembershipList, &struct {
		spaceRequest
	}{}, http.MethodGet)
	_ = reflector.SetJSONResponse(&opMembershipList, []types.MembershipUser{}, http.StatusOK)
	_ = reflector.SetJSONResponse(&opMembershipList, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opMembershipList, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opMembershipList, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opMembershipList, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/spaces/{space_ref}/members", opMembershipList)

	opDefineLabel := openapi3.Operation{}
	opDefineLabel.WithTags("space")
	opDefineLabel.WithMapOfAnything(
		map[string]interface{}{"operationId": "defineSpaceLabel"})
	_ = reflector.SetRequest(&opDefineLabel, &struct {
		spaceRequest
		LabelRequest
	}{}, http.MethodPost)
	_ = reflector.SetJSONResponse(&opDefineLabel, new(types.Label), http.StatusCreated)
	_ = reflector.SetJSONResponse(&opDefineLabel, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opDefineLabel, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opDefineLabel, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opDefineLabel, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opDefineLabel, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodPost, "/spaces/{space_ref}/labels", opDefineLabel)

	opSaveLabel := openapi3.Operation{}
	opSaveLabel.WithTags("space")
	opSaveLabel.WithMapOfAnything(
		map[string]interface{}{"operationId": "saveSpaceLabel"})
	_ = reflector.SetRequest(&opSaveLabel, &struct {
		spaceRequest
		types.SaveInput
	}{}, http.MethodPut)
	_ = reflector.SetJSONResponse(&opSaveLabel, new(types.LabelWithValues), http.StatusOK)
	_ = reflector.SetJSONResponse(&opSaveLabel, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opSaveLabel, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opSaveLabel, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opSaveLabel, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opSaveLabel, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodPut, "/spaces/{space_ref}/labels", opSaveLabel)

	opListLabels := openapi3.Operation{}
	opListLabels.WithTags("space")
	opListLabels.WithMapOfAnything(
		map[string]interface{}{"operationId": "listSpaceLabels"})
	opListLabels.WithParameters(
		QueryParameterPage, QueryParameterLimit, QueryParameterInherited, QueryParameterQueryLabel)
	_ = reflector.SetRequest(&opListLabels, new(spaceRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&opListLabels, new([]*types.Label), http.StatusOK)
	_ = reflector.SetJSONResponse(&opListLabels, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opListLabels, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opListLabels, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opListLabels, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opListLabels, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/spaces/{space_ref}/labels", opListLabels)

	opFindLabel := openapi3.Operation{}
	opFindLabel.WithTags("space")
	opFindLabel.WithMapOfAnything(map[string]interface{}{"operationId": "findSpaceLabel"})
	opFindLabel.WithParameters(queryParameterIncludeValues)
	_ = reflector.SetRequest(&opFindLabel, &struct {
		spaceRequest
		Key string `path:"key"`
	}{}, http.MethodGet)
	_ = reflector.SetJSONResponse(&opFindLabel, new(types.LabelWithValues), http.StatusCreated)
	_ = reflector.SetJSONResponse(&opFindLabel, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opFindLabel, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opFindLabel, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opFindLabel, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opFindLabel, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/spaces/{space_ref}/labels/{key}", opFindLabel)

	opDeleteLabel := openapi3.Operation{}
	opDeleteLabel.WithTags("space")
	opDeleteLabel.WithMapOfAnything(
		map[string]interface{}{"operationId": "deleteSpaceLabel"})
	_ = reflector.SetRequest(&opDeleteLabel, &struct {
		spaceRequest
		Key string `path:"key"`
	}{}, http.MethodDelete)
	_ = reflector.SetJSONResponse(&opDeleteLabel, nil, http.StatusNoContent)
	_ = reflector.SetJSONResponse(&opDeleteLabel, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opDeleteLabel, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opDeleteLabel, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opDeleteLabel, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opDeleteLabel, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(
		http.MethodDelete, "/spaces/{space_ref}/labels/{key}", opDeleteLabel)

	opUpdateLabel := openapi3.Operation{}
	opUpdateLabel.WithTags("space")
	opUpdateLabel.WithMapOfAnything(
		map[string]interface{}{"operationId": "updateSpaceLabel"})
	_ = reflector.SetRequest(&opUpdateLabel, &struct {
		spaceRequest
		LabelRequest
		Key string `path:"key"`
	}{}, http.MethodPatch)
	_ = reflector.SetJSONResponse(&opUpdateLabel, new(types.Label), http.StatusOK)
	_ = reflector.SetJSONResponse(&opUpdateLabel, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opUpdateLabel, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opUpdateLabel, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opUpdateLabel, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opUpdateLabel, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodPatch,
		"/spaces/{space_ref}/labels/{key}", opUpdateLabel)

	opDefineLabelValue := openapi3.Operation{}
	opDefineLabelValue.WithTags("space")
	opDefineLabelValue.WithMapOfAnything(
		map[string]interface{}{"operationId": "defineSpaceLabelValue"})
	_ = reflector.SetRequest(&opDefineLabelValue, &struct {
		spaceRequest
		LabelValueRequest
		Key string `path:"key"`
	}{}, http.MethodPost)
	_ = reflector.SetJSONResponse(&opDefineLabelValue, new(types.LabelValue), http.StatusCreated)
	_ = reflector.SetJSONResponse(&opDefineLabelValue, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opDefineLabelValue, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opDefineLabelValue, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opDefineLabelValue, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opDefineLabelValue, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodPost,
		"/spaces/{space_ref}/labels/{key}/values", opDefineLabelValue)

	opListLabelValues := openapi3.Operation{}
	opListLabelValues.WithTags("space")
	opListLabelValues.WithMapOfAnything(
		map[string]interface{}{"operationId": "listSpaceLabelValues"})
	_ = reflector.SetRequest(&opListLabelValues, &struct {
		spaceRequest
		Key string `path:"key"`
	}{}, http.MethodGet)
	_ = reflector.SetJSONResponse(&opListLabelValues, new([]*types.LabelValue), http.StatusOK)
	_ = reflector.SetJSONResponse(&opListLabelValues, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opListLabelValues, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opListLabelValues, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opListLabelValues, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opListLabelValues, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet,
		"/spaces/{space_ref}/labels/{key}/values", opListLabelValues)

	opDeleteLabelValue := openapi3.Operation{}
	opDeleteLabelValue.WithTags("space")
	opDeleteLabelValue.WithMapOfAnything(
		map[string]interface{}{"operationId": "deleteSpaceLabelValue"})
	_ = reflector.SetRequest(&opDeleteLabelValue, &struct {
		spaceRequest
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
		http.MethodDelete, "/spaces/{space_ref}/labels/{key}/values/{value}", opDeleteLabelValue)

	opUpdateLabelValue := openapi3.Operation{}
	opUpdateLabelValue.WithTags("space")
	opUpdateLabelValue.WithMapOfAnything(
		map[string]interface{}{"operationId": "updateSpaceLabelValue"})
	_ = reflector.SetRequest(&opUpdateLabelValue, &struct {
		spaceRequest
		LabelValueRequest
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
		"/spaces/{space_ref}/labels/{key}/values/{value}", opUpdateLabelValue)

	countPullReq := openapi3.Operation{}
	countPullReq.WithTags("space")
	countPullReq.WithMapOfAnything(map[string]interface{}{"operationId": "countSpacePullReq"})
	countPullReq.WithParameters(
		queryParameterStatePullRequest, queryParameterSourceRepoRefPullRequest,
		queryParameterSourceBranchPullRequest, queryParameterTargetBranchPullRequest,
		queryParameterQueryPullRequest, queryParameterCreatedByPullRequest,
		queryParameterCreatedLt, queryParameterCreatedGt, queryParameterUpdatedLt,
		queryParameterIncludeSubspaces,
		QueryParameterLabelID, QueryParameterValueID,
		queryParameterAuthorID, queryParameterCommenterID, queryParameterMentionedID,
		queryParameterReviewerID, queryParameterReviewDecision)
	_ = reflector.SetRequest(&countPullReq, new(spaceRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&countPullReq, new([]types.PullReqRepo), http.StatusOK)
	_ = reflector.SetJSONResponse(&countPullReq, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&countPullReq, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&countPullReq, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&countPullReq, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/spaces/{space_ref}/pullreq/count", countPullReq)

	listPullReq := openapi3.Operation{}
	listPullReq.WithTags("space")
	listPullReq.WithMapOfAnything(map[string]interface{}{"operationId": "listSpacePullReq"})
	listPullReq.WithParameters(
		queryParameterStatePullRequest, queryParameterSourceRepoRefPullRequest,
		queryParameterSourceBranchPullRequest, queryParameterTargetBranchPullRequest,
		queryParameterQueryPullRequest, queryParameterCreatedByPullRequest,
		queryParameterCreatedLt, queryParameterCreatedGt, queryParameterUpdatedLt,
		queryParameterExcludeDescription, queryParameterIncludeSubspaces,
		QueryParameterLimit,
		QueryParameterLabelID, QueryParameterValueID,
		queryParameterAuthorID, queryParameterCommenterID, queryParameterMentionedID,
		queryParameterReviewerID, queryParameterReviewDecision,
		queryParamIncludeGitStats, queryParameterIncludeChecks, queryParameterIncludeRules)
	_ = reflector.SetRequest(&listPullReq, new(spaceRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&listPullReq, new([]types.PullReqRepo), http.StatusOK)
	_ = reflector.SetJSONResponse(&listPullReq, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&listPullReq, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&listPullReq, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&listPullReq, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/spaces/{space_ref}/pullreq", listPullReq)

	opGetUsageMetrics := openapi3.Operation{}
	opGetUsageMetrics.WithTags("space")
	opGetUsageMetrics.WithMapOfAnything(map[string]interface{}{"operationId": "getSpaceUsageMetric"})
	_ = reflector.SetRequest(&opGetUsageMetrics, new(spaceRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&opGetUsageMetrics, new(types.UsageMetric), http.StatusOK)
	_ = reflector.SetJSONResponse(&opGetUsageMetrics, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opGetUsageMetrics, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opGetUsageMetrics, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/spaces/{space_ref}/usage/metric", opGetUsageMetrics)
}
