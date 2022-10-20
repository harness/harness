// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package openapi

import (
	"github.com/gotidy/ptr"
	"github.com/harness/gitness/internal/api/request"
	"github.com/swaggest/openapi-go/openapi3"
)

func ptrSchemaType(t openapi3.SchemaType) *openapi3.SchemaType {
	return &t
}

func ptrptr(i interface{}) *interface{} {
	return &i
}

var queryParameterPage = openapi3.ParameterOrRef{
	Parameter: &openapi3.Parameter{
		Name:        request.QueryParamPage,
		In:          openapi3.ParameterInQuery,
		Description: ptr.String("The page to return."),
		Required:    ptr.Bool(false),
		Schema: &openapi3.SchemaOrRef{
			Schema: &openapi3.Schema{
				Type:    ptrSchemaType(openapi3.SchemaTypeInteger),
				Default: ptrptr(1),
				Minimum: ptr.Float64(1),
			},
		},
	},
}

var queryParameterPerPage = openapi3.ParameterOrRef{
	Parameter: &openapi3.Parameter{
		Name:        request.QueryParamPerPage,
		In:          openapi3.ParameterInQuery,
		Description: ptr.String("The number of entries returned per page."),
		Required:    ptr.Bool(false),
		Schema: &openapi3.SchemaOrRef{
			Schema: &openapi3.Schema{
				Type:    ptrSchemaType(openapi3.SchemaTypeInteger),
				Default: ptrptr(request.PerPageDefault),
				Minimum: ptr.Float64(1.0),
				Maximum: ptr.Float64(request.PerPageMax),
			},
		},
	},
}
