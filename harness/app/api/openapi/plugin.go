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

	"github.com/gotidy/ptr"
	"github.com/swaggest/openapi-go/openapi3"
)

var queryParameterQueryPlugin = openapi3.ParameterOrRef{
	Parameter: &openapi3.Parameter{
		Name:        request.QueryParamQuery,
		In:          openapi3.ParameterInQuery,
		Description: ptr.String("The substring which is used to filter the plugins by their name."),
		Required:    ptr.Bool(false),
		Schema: &openapi3.SchemaOrRef{
			Schema: &openapi3.Schema{
				Type: ptrSchemaType(openapi3.SchemaTypeString),
			},
		},
	},
}

type getPluginsRequest struct {
}

func pluginOperations(reflector *openapi3.Reflector) {
	opPlugins := openapi3.Operation{}
	opPlugins.WithTags("plugins")
	opPlugins.WithMapOfAnything(map[string]interface{}{"operationId": "listPlugins"})
	opPlugins.WithParameters(QueryParameterPage, QueryParameterLimit, queryParameterQueryPlugin)
	_ = reflector.SetRequest(&opPlugins, new(getPluginsRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&opPlugins, []types.Plugin{}, http.StatusOK)
	_ = reflector.SetJSONResponse(&opPlugins, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opPlugins, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opPlugins, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opPlugins, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/plugins", opPlugins)
}
