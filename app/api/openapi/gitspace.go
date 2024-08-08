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

	"github.com/harness/gitness/app/api/controller/gitspace"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/gitspace/scm"
	"github.com/harness/gitness/livelog"
	"github.com/harness/gitness/types"

	"github.com/swaggest/openapi-go/openapi3"
)

type createGitspaceRequest struct {
	gitspace.CreateInput
}

type lookupRepoGitspaceRequest struct {
	gitspace.LookupRepoInput
}

type actionGitspaceRequest struct {
	gitspaceRequest
	gitspace.ActionInput
}

type updateGitspaceRequest struct {
}

type gitspaceRequest struct {
	Ref string `path:"gitspace_identifier"`
}

type getGitspaceRequest struct {
	gitspaceRequest
}

type gitspacesListRequest struct {
	Sort  string `query:"sort"      enum:"id,created,updated"`
	Order string `query:"order"     enum:"asc,desc"`

	// include pagination request
	paginationRequest
}

type gitspaceEventsListRequest struct {
	Ref string `path:"gitspace_identifier"`
	paginationRequest
}

type gitspacesListAllRequest struct {
}

func gitspaceOperations(reflector *openapi3.Reflector) {
	opCreate := openapi3.Operation{}
	opCreate.WithTags("gitspaces")
	opCreate.WithSummary("Create gitspace config")
	opCreate.WithMapOfAnything(map[string]interface{}{"operationId": "createGitspace"})
	_ = reflector.SetRequest(&opCreate, new(createGitspaceRequest), http.MethodPost)
	_ = reflector.SetJSONResponse(&opCreate, new(types.GitspaceConfig), http.StatusCreated)
	_ = reflector.SetJSONResponse(&opCreate, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opCreate, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opCreate, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opCreate, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodPost, "/gitspaces", opCreate)

	opUpdate := openapi3.Operation{}
	opUpdate.WithTags("gitspaces")
	opUpdate.WithSummary("Update gitspace config")
	opUpdate.WithMapOfAnything(map[string]interface{}{"operationId": "updateGitspace"})
	_ = reflector.SetRequest(&opUpdate, new(updateGitspaceRequest), http.MethodPut)
	_ = reflector.SetJSONResponse(&opUpdate, new(types.GitspaceConfig), http.StatusOK)
	_ = reflector.SetJSONResponse(&opUpdate, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opUpdate, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opUpdate, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opUpdate, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opUpdate, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodPatch, "/gitspaces/{gitspace_identifier}", opUpdate)

	opFind := openapi3.Operation{}
	opFind.WithTags("gitspaces")
	opFind.WithSummary("Get gitspace")
	opFind.WithMapOfAnything(map[string]interface{}{"operationId": "findGitspace"})
	_ = reflector.SetRequest(&opFind, new(getGitspaceRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&opFind, new(types.GitspaceConfig), http.StatusOK)
	_ = reflector.SetJSONResponse(&opFind, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opFind, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opFind, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opFind, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/gitspaces/{gitspace_identifier}", opFind)

	opDelete := openapi3.Operation{}
	opDelete.WithTags("gitspaces")
	opDelete.WithSummary("Delete gitspace config")
	opDelete.WithMapOfAnything(map[string]interface{}{"operationId": "deleteGitspace"})
	_ = reflector.SetRequest(&opDelete, new(getGitspaceRequest), http.MethodDelete)
	_ = reflector.SetJSONResponse(&opDelete, nil, http.StatusNoContent)
	_ = reflector.SetJSONResponse(&opDelete, nil, http.StatusNoContent)
	_ = reflector.SetJSONResponse(&opDelete, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opDelete, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opDelete, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opDelete, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(
		http.MethodDelete, "/gitspaces/{gitspace_identifier}", opDelete)

	opList := openapi3.Operation{}
	opList.WithTags("gitspaces")
	opList.WithSummary("List gitspaces")
	opList.WithMapOfAnything(map[string]interface{}{"operationId": "listGitspaces"})
	_ = reflector.SetRequest(&opList, new(gitspacesListRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&opList, new([]*types.GitspaceConfig), http.StatusOK)
	_ = reflector.SetJSONResponse(&opList, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opList, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opList, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/gitspaces", opList)

	opEventList := openapi3.Operation{}
	opEventList.WithTags("gitspaces")
	opEventList.WithSummary("List gitspace events")
	opEventList.WithMapOfAnything(map[string]interface{}{"operationId": "listGitspaceEvents"})
	_ = reflector.SetRequest(&opEventList, new(gitspaceEventsListRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&opEventList, new([]*types.GitspaceEventResponse), http.StatusOK)
	_ = reflector.SetJSONResponse(&opEventList, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opEventList, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opEventList, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/gitspaces/{gitspace_identifier}/events", opEventList)

	opStreamLogs := openapi3.Operation{}
	opStreamLogs.WithTags("gitspaces")
	opStreamLogs.WithSummary("Stream gitspace logs")
	opStreamLogs.WithMapOfAnything(map[string]interface{}{"operationId": "opStreamLogs"})
	_ = reflector.SetRequest(&opStreamLogs, new(gitspaceRequest), http.MethodGet)
	_ = reflector.SetStringResponse(&opStreamLogs, http.StatusOK, "text/event-stream")
	_ = reflector.SetJSONResponse(&opStreamLogs, []*livelog.Line{}, http.StatusOK)
	_ = reflector.SetJSONResponse(&opStreamLogs, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opStreamLogs, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opStreamLogs, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/gitspaces/{gitspace_identifier}/logs/stream", opStreamLogs)

	opRepoLookup := openapi3.Operation{}
	opRepoLookup.WithTags("gitspaces")
	opRepoLookup.WithSummary("Validate git repo for gitspaces")
	opRepoLookup.WithMapOfAnything(map[string]interface{}{"operationId": "repoLookupForGitspace"})
	_ = reflector.SetRequest(&opRepoLookup, new(lookupRepoGitspaceRequest), http.MethodPost)
	_ = reflector.SetJSONResponse(&opRepoLookup, new(scm.CodeRepositoryResponse), http.StatusCreated)
	_ = reflector.SetJSONResponse(&opRepoLookup, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opRepoLookup, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opRepoLookup, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opRepoLookup, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodPost, "/gitspaces/lookup-repo", opRepoLookup)

	opListAll := openapi3.Operation{}
	opListAll.WithTags("gitspaces")
	opListAll.WithSummary("List all gitspaces")
	opListAll.WithMapOfAnything(map[string]interface{}{"operationId": "listAllGitspaces"})
	_ = reflector.SetRequest(&opListAll, new(gitspacesListAllRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&opListAll, new([]*types.GitspaceConfig), http.StatusOK)
	_ = reflector.SetJSONResponse(&opListAll, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opListAll, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opListAll, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/gitspaces", opListAll)

	opAction := openapi3.Operation{}
	opAction.WithTags("gitspaces")
	opAction.WithSummary("Perform action on a gitspace")
	opAction.WithMapOfAnything(map[string]interface{}{"operationId": "actionOnGitspace"})
	_ = reflector.SetRequest(&opAction, new(actionGitspaceRequest), http.MethodPost)
	_ = reflector.SetJSONResponse(&opAction, new(types.GitspaceConfig), http.StatusOK)
	_ = reflector.SetJSONResponse(&opAction, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opAction, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opAction, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodPost, "/gitspaces/{gitspace_identifier}/action", opAction)
}
