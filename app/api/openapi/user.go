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

	"github.com/harness/gitness/app/api/controller/user"
	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/gotidy/ptr"
	"github.com/swaggest/openapi-go/openapi3"
)

type tokensRequest struct {
	Identifier string `path:"token_identifier"`
}

type favoriteRequest struct {
	ResourceID   int64  `path:"resource_id"`
	ResourceType string `query:"resource_type"`
}

var queryParameterMembershipSpaces = openapi3.ParameterOrRef{
	Parameter: &openapi3.Parameter{
		Name:        request.QueryParamQuery,
		In:          openapi3.ParameterInQuery,
		Description: ptr.String("The substring by which the spaces the users is a member of are filtered."),
		Required:    ptr.Bool(false),
		Schema: &openapi3.SchemaOrRef{
			Schema: &openapi3.Schema{
				Type: ptrSchemaType(openapi3.SchemaTypeString),
			},
		},
	},
}

var queryParameterSortMembershipSpaces = openapi3.ParameterOrRef{
	Parameter: &openapi3.Parameter{
		Name:        request.QueryParamSort,
		In:          openapi3.ParameterInQuery,
		Description: ptr.String("The field by which the spaces the user is a member of are sorted."),
		Required:    ptr.Bool(false),
		Schema: &openapi3.SchemaOrRef{
			Schema: &openapi3.Schema{
				Type:    ptrSchemaType(openapi3.SchemaTypeString),
				Default: ptrptr(enum.MembershipSpaceSortIdentifier),
				Enum:    enum.MembershipSpaceSort("").Enum(),
			},
		},
	},
}

var queryParameterQueryPublicKey = openapi3.ParameterOrRef{
	Parameter: &openapi3.Parameter{
		Name:        request.QueryParamQuery,
		In:          openapi3.ParameterInQuery,
		Description: ptr.String("The substring which is used to filter the public keys by their path identifier."),
		Required:    ptr.Bool(false),
		Schema: &openapi3.SchemaOrRef{
			Schema: &openapi3.Schema{
				Type: ptrSchemaType(openapi3.SchemaTypeString),
			},
		},
	},
}

var queryParameterSortPublicKey = openapi3.ParameterOrRef{
	Parameter: &openapi3.Parameter{
		Name:        request.QueryParamSort,
		In:          openapi3.ParameterInQuery,
		Description: ptr.String("The data by which the public keys are sorted."),
		Required:    ptr.Bool(false),
		Schema: &openapi3.SchemaOrRef{
			Schema: &openapi3.Schema{
				Type:    ptrSchemaType(openapi3.SchemaTypeString),
				Default: ptrptr(enum.PublicKeySortCreated),
				Enum:    enum.PublicKeySort("").Enum(),
			},
		},
	},
}

var queryParameterUsagePublicKey = openapi3.ParameterOrRef{
	Parameter: &openapi3.Parameter{
		Name:        request.QueryParamPublicKeyUsage,
		In:          openapi3.ParameterInQuery,
		Description: ptr.String("The public key usage."),
		Required:    ptr.Bool(false),
		Schema: &openapi3.SchemaOrRef{
			Schema: &openapi3.Schema{
				Type: ptrSchemaType(openapi3.SchemaTypeArray),
				Items: &openapi3.SchemaOrRef{
					Schema: &openapi3.Schema{
						Type: ptrSchemaType(openapi3.SchemaTypeString),
						Enum: enum.PublicKeyUsage("").Enum(),
					},
				},
			},
		},
	},
}

var queryParameterSchemePublicKey = openapi3.ParameterOrRef{
	Parameter: &openapi3.Parameter{
		Name:        request.QueryParamPublicKeyScheme,
		In:          openapi3.ParameterInQuery,
		Description: ptr.String("The public key scheme."),
		Required:    ptr.Bool(false),
		Schema: &openapi3.SchemaOrRef{
			Schema: &openapi3.Schema{
				Type: ptrSchemaType(openapi3.SchemaTypeArray),
				Items: &openapi3.SchemaOrRef{
					Schema: &openapi3.Schema{
						Type: ptrSchemaType(openapi3.SchemaTypeString),
						Enum: enum.PublicKeyScheme("").Enum(),
					},
				},
			},
		},
	},
}

var QueryParameterResourceType = openapi3.ParameterOrRef{
	Parameter: &openapi3.Parameter{
		Name:        request.QueryParamResourceType,
		In:          openapi3.ParameterInQuery,
		Description: ptr.String("The type of the resource to be unfavorited."),
		Required:    ptr.Bool(false),
		Schema: &openapi3.SchemaOrRef{
			Schema: &openapi3.Schema{
				Type: ptrSchemaType(openapi3.SchemaTypeString),
				Enum: enum.ResourceType("").Enum(),
			},
		},
	},
}

// helper function that constructs the openapi specification
// for user account resources.
func buildUser(reflector *openapi3.Reflector) {
	opFind := openapi3.Operation{}
	opFind.WithTags("user")
	opFind.WithMapOfAnything(map[string]any{"operationId": "getUser"})
	_ = reflector.SetRequest(&opFind, nil, http.MethodGet)
	_ = reflector.SetJSONResponse(&opFind, new(types.User), http.StatusOK)
	_ = reflector.SetJSONResponse(&opFind, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/user", opFind)

	opUpdate := openapi3.Operation{}
	opUpdate.WithTags("user")
	opUpdate.WithMapOfAnything(map[string]any{"operationId": "updateUser"})
	_ = reflector.SetRequest(&opUpdate, new(user.UpdateInput), http.MethodPatch)
	_ = reflector.SetJSONResponse(&opUpdate, new(types.User), http.StatusOK)
	_ = reflector.SetJSONResponse(&opUpdate, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.Spec.AddOperation(http.MethodPatch, "/user", opUpdate)

	opMemberSpaces := openapi3.Operation{}
	opMemberSpaces.WithTags("user")
	opMemberSpaces.WithMapOfAnything(map[string]any{"operationId": "membershipSpaces"})
	opMemberSpaces.WithParameters(
		queryParameterMembershipSpaces,
		queryParameterOrder, queryParameterSortMembershipSpaces,
		QueryParameterPage, QueryParameterLimit)
	_ = reflector.SetRequest(&opMemberSpaces, struct{}{}, http.MethodGet)
	_ = reflector.SetJSONResponse(&opMemberSpaces, new([]types.MembershipSpace), http.StatusOK)
	_ = reflector.SetJSONResponse(&opMemberSpaces, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/user/memberships", opMemberSpaces)

	opKeyCreate := openapi3.Operation{}
	opKeyCreate.WithTags("user")
	opKeyCreate.WithMapOfAnything(map[string]any{"operationId": "createPublicKey"})
	_ = reflector.SetRequest(&opKeyCreate, new(user.CreatePublicKeyInput), http.MethodPost)
	_ = reflector.SetJSONResponse(&opKeyCreate, new(types.PublicKey), http.StatusCreated)
	_ = reflector.SetJSONResponse(&opKeyCreate, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opKeyCreate, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.Spec.AddOperation(http.MethodPost, "/user/keys", opKeyCreate)

	opKeyDelete := openapi3.Operation{}
	opKeyDelete.WithTags("user")
	opKeyDelete.WithMapOfAnything(map[string]any{"operationId": "deletePublicKey"})
	_ = reflector.SetRequest(&opKeyDelete, struct {
		ID string `path:"public_key_identifier"`
	}{}, http.MethodDelete)
	_ = reflector.SetJSONResponse(&opKeyDelete, nil, http.StatusNoContent)
	_ = reflector.SetJSONResponse(&opKeyDelete, new(usererror.Error), http.StatusNotFound)
	_ = reflector.SetJSONResponse(&opKeyDelete, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.Spec.AddOperation(http.MethodDelete, "/user/keys/{public_key_identifier}", opKeyDelete)

	opKeyUpdate := openapi3.Operation{}
	opKeyUpdate.WithTags("user")
	opKeyUpdate.WithMapOfAnything(map[string]any{"operationId": "updatePublicKey"})
	_ = reflector.SetRequest(&opKeyUpdate, struct {
		ID string `path:"public_key_identifier"`
	}{}, http.MethodPatch)
	_ = reflector.SetJSONResponse(&opKeyUpdate, &types.PublicKey{}, http.StatusOK)
	_ = reflector.SetJSONResponse(&opKeyUpdate, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opKeyUpdate, new(usererror.Error), http.StatusNotFound)
	_ = reflector.SetJSONResponse(&opKeyUpdate, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.Spec.AddOperation(http.MethodPatch, "/user/keys/{public_key_identifier}", opKeyUpdate)

	opKeyList := openapi3.Operation{}
	opKeyList.WithTags("user")
	opKeyList.WithMapOfAnything(map[string]any{"operationId": "listPublicKey"})
	opKeyList.WithParameters(QueryParameterPage, QueryParameterLimit,
		queryParameterQueryPublicKey, queryParameterSortPublicKey, queryParameterOrder,
		queryParameterUsagePublicKey, queryParameterSchemePublicKey,
	)
	_ = reflector.SetRequest(&opKeyList, struct{}{}, http.MethodGet)
	_ = reflector.SetJSONResponse(&opKeyList, new([]types.PublicKey), http.StatusOK)
	_ = reflector.SetJSONResponse(&opKeyList, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opKeyList, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/user/keys", opKeyList)

	opListTokens := openapi3.Operation{}
	opListTokens.WithTags("user")
	opListTokens.WithMapOfAnything(map[string]any{"operationId": "listTokens"})
	_ = reflector.SetRequest(&opListTokens, nil, http.MethodGet)
	_ = reflector.SetJSONResponse(&opListTokens, new([]types.Token), http.StatusOK)
	_ = reflector.SetJSONResponse(&opListTokens, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opListTokens, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opListTokens, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/user/tokens", opListTokens)

	opCreateToken := openapi3.Operation{}
	opCreateToken.WithTags("user")
	opCreateToken.WithMapOfAnything(map[string]any{"operationId": "createToken"})
	_ = reflector.SetRequest(&opCreateToken, new(user.CreateTokenInput), http.MethodPost)
	_ = reflector.SetJSONResponse(&opCreateToken, new(types.TokenResponse), http.StatusCreated)
	_ = reflector.SetJSONResponse(&opCreateToken, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opCreateToken, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opCreateToken, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.Spec.AddOperation(http.MethodPost, "/user/tokens", opCreateToken)

	opDeleteToken := openapi3.Operation{}
	opDeleteToken.WithTags("user")
	opDeleteToken.WithMapOfAnything(map[string]any{"operationId": "deleteToken"})
	_ = reflector.SetRequest(&opDeleteToken, new(tokensRequest), http.MethodDelete)
	_ = reflector.SetJSONResponse(&opDeleteToken, nil, http.StatusNoContent)
	_ = reflector.SetJSONResponse(&opDeleteToken, new(usererror.Error), http.StatusNotFound)
	_ = reflector.SetJSONResponse(&opDeleteToken, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opDeleteToken, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opDeleteToken, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.Spec.AddOperation(http.MethodDelete, "/user/tokens/{token_identifier}", opDeleteToken)

	opCreateFavorite := openapi3.Operation{}
	opCreateFavorite.WithTags("user")
	opCreateFavorite.WithMapOfAnything(map[string]any{"operationId": "createFavorite"})
	_ = reflector.SetRequest(&opCreateFavorite, new(types.FavoriteResource), http.MethodPost)
	_ = reflector.SetJSONResponse(&opCreateFavorite, new(types.FavoriteResource), http.StatusCreated)
	_ = reflector.SetJSONResponse(&opCreateFavorite, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opCreateFavorite, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opCreateFavorite, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.Spec.AddOperation(http.MethodPost, "/user/favorite", opCreateFavorite)

	opDeleteFavorite := openapi3.Operation{}
	opDeleteFavorite.WithTags("user")
	opDeleteFavorite.WithMapOfAnything(map[string]any{"operationId": "deleteFavorite"})
	opDeleteFavorite.WithParameters(QueryParameterResourceType)
	_ = reflector.SetRequest(&opDeleteFavorite, new(favoriteRequest), http.MethodDelete)
	_ = reflector.SetJSONResponse(&opDeleteFavorite, nil, http.StatusNoContent)
	_ = reflector.SetJSONResponse(&opDeleteFavorite, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opDeleteFavorite, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opDeleteFavorite, new(usererror.Error), http.StatusNotFound)
	_ = reflector.SetJSONResponse(&opDeleteFavorite, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.Spec.AddOperation(http.MethodDelete, "/user/favorite/{resource_id}", opDeleteFavorite)
}
