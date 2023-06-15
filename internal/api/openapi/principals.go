// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package openapi

import (
	"net/http"

	"github.com/harness/gitness/internal/api/request"
	"github.com/harness/gitness/internal/api/usererror"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/gotidy/ptr"
	"github.com/swaggest/openapi-go/openapi3"
)

type principalRequest struct {
}

var queryParameterQueryPrincipals = openapi3.ParameterOrRef{
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

// TODO: this should not be in standalone swagger.
// https://harness.atlassian.net/browse/CODE-521
var queryParameterAccountID = openapi3.ParameterOrRef{
	Parameter: &openapi3.Parameter{
		Name:        "accountIdentifier",
		In:          openapi3.ParameterInQuery,
		Description: ptr.String("The account ID the principals are retrieved for (Not required in standalone)."),
		Required:    ptr.Bool(false),
		Schema: &openapi3.SchemaOrRef{
			Schema: &openapi3.Schema{
				Type: ptrSchemaType(openapi3.SchemaTypeString),
			},
		},
	},
}

var queryParameterPrincipalTypes = openapi3.ParameterOrRef{
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
	opList.WithParameters(queryParameterQueryPrincipals, queryParameterAccountID, queryParameterPage,
		queryParameterLimit, queryParameterPrincipalTypes)
	_ = reflector.SetRequest(&opList, new(principalRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&opList, new([]types.PrincipalInfo), http.StatusOK)
	_ = reflector.SetJSONResponse(&opList, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opList, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opList, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/principals", opList)
}
