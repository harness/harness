// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package openapi

import (
	"net/http"

	"github.com/gotidy/ptr"
	"github.com/harness/gitness/internal/api/request"
	"github.com/harness/gitness/internal/api/usererror"
	"github.com/harness/gitness/types"

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
	opPlugins.WithParameters(queryParameterPage, queryParameterLimit, queryParameterQueryPlugin)
	_ = reflector.SetRequest(&opPlugins, new(getPluginsRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&opPlugins, []types.Plugin{}, http.StatusOK)
	_ = reflector.SetJSONResponse(&opPlugins, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opPlugins, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opPlugins, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opPlugins, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/plugins", opPlugins)
}
