// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package openapi

import (
	"net/http"

	"github.com/harness/gitness/internal/api/controller/space"
	"github.com/harness/gitness/internal/api/request"
	"github.com/harness/gitness/internal/api/usererror"
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

type moveSpaceRequest struct {
	spaceRequest
	space.MoveInput
}

type createPathRequest struct {
	spaceRequest
	space.CreatePathInput
}

type deletePathRequest struct {
	spaceRequest
	PathID string `path:"path_id"`
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
				Default: ptrptr(enum.RepoAttrUID.String()),
				Enum: []interface{}{
					ptr.String(enum.RepoAttrUID.String()),
					ptr.String(enum.RepoAttrPath.String()),
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
				Default: ptrptr(enum.SpaceAttrUID.String()),
				Enum: []interface{}{
					ptr.String(enum.SpaceAttrUID.String()),
					ptr.String(enum.SpaceAttrPath.String()),
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

var queryParameterSpaceMembers = openapi3.ParameterOrRef{
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

var queryParameterSortSpaceMembers = openapi3.ParameterOrRef{
	Parameter: &openapi3.Parameter{
		Name:        request.QueryParamSort,
		In:          openapi3.ParameterInQuery,
		Description: ptr.String("The field by which the space members are sorted."),
		Required:    ptr.Bool(false),
		Schema: &openapi3.SchemaOrRef{
			Schema: &openapi3.Schema{
				Type:    ptrSchemaType(openapi3.SchemaTypeString),
				Default: ptrptr(enum.MembershipSortName),
				Enum:    enum.MembershipSort("").Enum(),
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
	_ = reflector.SetJSONResponse(&opCreate, new(types.Space), http.StatusCreated)
	_ = reflector.SetJSONResponse(&opCreate, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opCreate, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opCreate, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opCreate, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodPost, "/spaces", opCreate)

	opGet := openapi3.Operation{}
	opGet.WithTags("space")
	opGet.WithMapOfAnything(map[string]interface{}{"operationId": "getSpace"})
	_ = reflector.SetRequest(&opGet, new(spaceRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&opGet, new(types.Space), http.StatusOK)
	_ = reflector.SetJSONResponse(&opGet, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opGet, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opGet, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opGet, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/spaces/{space_ref}", opGet)

	opUpdate := openapi3.Operation{}
	opUpdate.WithTags("space")
	opUpdate.WithMapOfAnything(map[string]interface{}{"operationId": "updateSpace"})
	_ = reflector.SetRequest(&opUpdate, new(updateSpaceRequest), http.MethodPatch)
	_ = reflector.SetJSONResponse(&opUpdate, new(types.Space), http.StatusOK)
	_ = reflector.SetJSONResponse(&opUpdate, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opUpdate, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opUpdate, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opUpdate, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opUpdate, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodPatch, "/spaces/{space_ref}", opUpdate)

	opDelete := openapi3.Operation{}
	opDelete.WithTags("space")
	opDelete.WithMapOfAnything(map[string]interface{}{"operationId": "deleteSpace"})
	_ = reflector.SetRequest(&opDelete, new(spaceRequest), http.MethodDelete)
	_ = reflector.SetJSONResponse(&opDelete, nil, http.StatusNoContent)
	_ = reflector.SetJSONResponse(&opDelete, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opDelete, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opDelete, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opDelete, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodDelete, "/spaces/{space_ref}", opDelete)

	opMove := openapi3.Operation{}
	opMove.WithTags("space")
	opMove.WithMapOfAnything(map[string]interface{}{"operationId": "moveSpace"})
	_ = reflector.SetRequest(&opMove, new(moveSpaceRequest), http.MethodPost)
	_ = reflector.SetJSONResponse(&opMove, new(types.Space), http.StatusOK)
	_ = reflector.SetJSONResponse(&opMove, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opMove, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opMove, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opMove, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodPost, "/spaces/{space_ref}/move", opMove)

	opSpaces := openapi3.Operation{}
	opSpaces.WithTags("space")
	opSpaces.WithMapOfAnything(map[string]interface{}{"operationId": "listSpaces"})
	opSpaces.WithParameters(queryParameterPage, queryParameterLimit)
	opSpaces.WithParameters(queryParameterQuerySpace, queryParameterSortSpace, queryParameterOrder,
		queryParameterPage, queryParameterLimit)
	_ = reflector.SetRequest(&opSpaces, new(spaceRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&opSpaces, []types.Space{}, http.StatusOK)
	_ = reflector.SetJSONResponse(&opSpaces, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opSpaces, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opSpaces, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opSpaces, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/spaces/{space_ref}/spaces", opSpaces)

	opRepos := openapi3.Operation{}
	opRepos.WithTags("space")
	opRepos.WithMapOfAnything(map[string]interface{}{"operationId": "listRepos"})
	opRepos.WithParameters(queryParameterQueryRepo, queryParameterSortRepo, queryParameterOrder,
		queryParameterPage, queryParameterLimit)
	_ = reflector.SetRequest(&opRepos, new(spaceRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&opRepos, []types.Repository{}, http.StatusOK)
	_ = reflector.SetJSONResponse(&opRepos, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opRepos, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opRepos, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opRepos, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/spaces/{space_ref}/repos", opRepos)

	opPipelines := openapi3.Operation{}
	opPipelines.WithTags("space")
	opPipelines.WithMapOfAnything(map[string]interface{}{"operationId": "listPipelines"})
	opPipelines.WithParameters(queryParameterQueryRepo, queryParameterPage, queryParameterLimit)
	_ = reflector.SetRequest(&opPipelines, new(spaceRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&opPipelines, []types.Pipeline{}, http.StatusOK)
	_ = reflector.SetJSONResponse(&opPipelines, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opPipelines, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opPipelines, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opPipelines, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/spaces/{space_ref}/pipelines", opPipelines)

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

	opListPaths := openapi3.Operation{}
	opListPaths.WithTags("space")
	opListPaths.WithMapOfAnything(map[string]interface{}{"operationId": "listPaths"})
	opListPaths.WithParameters(queryParameterPage, queryParameterLimit)
	_ = reflector.SetRequest(&opListPaths, new(spaceRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&opListPaths, []types.Path{}, http.StatusOK)
	_ = reflector.SetJSONResponse(&opListPaths, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opListPaths, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opListPaths, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opListPaths, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/spaces/{space_ref}/paths", opListPaths)

	opCreatePath := openapi3.Operation{}
	opCreatePath.WithTags("space")
	opCreatePath.WithMapOfAnything(map[string]interface{}{"operationId": "createPath"})
	_ = reflector.SetRequest(&opCreatePath, new(createPathRequest), http.MethodPost)
	_ = reflector.SetJSONResponse(&opCreatePath, new(types.Path), http.StatusCreated)
	_ = reflector.SetJSONResponse(&opCreatePath, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opCreatePath, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opCreatePath, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opCreatePath, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodPost, "/spaces/{space_ref}/paths", opCreatePath)

	onDeletePath := openapi3.Operation{}
	onDeletePath.WithTags("space")
	onDeletePath.WithMapOfAnything(map[string]interface{}{"operationId": "deletePath"})
	_ = reflector.SetRequest(&onDeletePath, new(deletePathRequest), http.MethodDelete)
	_ = reflector.SetJSONResponse(&onDeletePath, nil, http.StatusNoContent)
	_ = reflector.SetJSONResponse(&onDeletePath, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&onDeletePath, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&onDeletePath, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&onDeletePath, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodDelete, "/spaces/{space_ref}/paths/{path_id}", onDeletePath)

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
		queryParameterSpaceMembers,
		queryParameterOrder, queryParameterSortSpaceMembers,
		queryParameterPage, queryParameterLimit)
	_ = reflector.SetRequest(&opMembershipList, &struct {
		spaceRequest
	}{}, http.MethodGet)
	_ = reflector.SetJSONResponse(&opMembershipList, []types.MembershipUser{}, http.StatusOK)
	_ = reflector.SetJSONResponse(&opMembershipList, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opMembershipList, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opMembershipList, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opMembershipList, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/spaces/{space_ref}/members", opMembershipList)
}
