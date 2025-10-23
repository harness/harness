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

	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/gotidy/ptr"
	"github.com/swaggest/openapi-go/openapi3"
)

type principalInfoRequest struct {
	ID int64 `path:"id"`
}

var QueryParameterQueryPrincipals = openapi3.ParameterOrRef{
	Parameter: &openapi3.Parameter{
		Name:        request.QueryParamQuery,
		In:          openapi3.ParameterInQuery,
		Description: ptr.String("The substring by which the principals are filtered."),
		Required:    ptr.Bool(false),
		Schema: &openapi3.SchemaOrRef{
			Schema: &openapi3.Schema{
				Type: ptrSchemaType(openapi3.SchemaTypeString),
			},
		},
	},
}

var QueryParameterPrincipalTypes = openapi3.ParameterOrRef{
	Parameter: &openapi3.Parameter{
		Name:        request.QueryParamType,
		In:          openapi3.ParameterInQuery,
		Description: ptr.String("The types of principals to include."),
		Required:    ptr.Bool(false),
		Schema: &openapi3.SchemaOrRef{
			Schema: &openapi3.Schema{
				Type: ptrSchemaType(openapi3.SchemaTypeArray),
				Items: &openapi3.SchemaOrRef{
					Schema: &openapi3.Schema{
						Type: ptrSchemaType(openapi3.SchemaTypeString),
						Enum: enum.PrincipalType("").Enum(),
					},
				},
			},
		},
	},
}

// buildPrincipals function that constructs the openapi specification
// for principal resources.
func buildPrincipals(reflector *openapi3.Reflector) {
	opList := openapi3.Operation{}
	opList.WithTags("principals")
	opList.WithMapOfAnything(map[string]interface{}{"operationId": "listPrincipals"})
	opList.WithParameters(QueryParameterQueryPrincipals, QueryParameterPage,
		QueryParameterLimit, QueryParameterPrincipalTypes)
	_ = reflector.SetRequest(&opList, nil, http.MethodGet)
	_ = reflector.SetJSONResponse(&opList, new([]types.PrincipalInfo), http.StatusOK)
	_ = reflector.SetJSONResponse(&opList, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opList, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opList, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/principals", opList)

	getPrincipal := openapi3.Operation{}
	getPrincipal.WithTags("principals")
	getPrincipal.WithMapOfAnything(map[string]interface{}{"operationId": "getPrincipal"})
	_ = reflector.SetRequest(&getPrincipal, new(principalInfoRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&getPrincipal, new(types.PrincipalInfo), http.StatusOK)
	_ = reflector.SetJSONResponse(&getPrincipal, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&getPrincipal, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&getPrincipal, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&getPrincipal, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&getPrincipal, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/principals/{id}", getPrincipal)
}
