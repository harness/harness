// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package openapi

import (
	"github.com/harness/gitness/internal/api/request"
	"github.com/harness/gitness/types/enum"

	"github.com/gotidy/ptr"
	"github.com/swaggest/openapi-go/openapi3"
)

func ptrSchemaType(t openapi3.SchemaType) *openapi3.SchemaType {
	return &t
}

func ptrptr(i interface{}) *interface{} {
	return &i
}

func toInterfaceSlice[T interface{}](vals []T) []interface{} {
	res := make([]interface{}, len(vals))
	for i := range vals {
		res[i] = vals[i]
	}
	return res
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

var queryParameterOrder = openapi3.ParameterOrRef{
	Parameter: &openapi3.Parameter{
		Name:        request.QueryParamOrder,
		In:          openapi3.ParameterInQuery,
		Description: ptr.String("The order of the output."),
		Required:    ptr.Bool(false),
		Schema: &openapi3.SchemaOrRef{
			Schema: &openapi3.Schema{
				Type:    ptrSchemaType(openapi3.SchemaTypeString),
				Default: ptrptr(enum.OrderAsc.String()),
				Enum: []interface{}{
					ptr.String(enum.OrderAsc.String()),
					ptr.String(enum.OrderDesc.String()),
				},
			},
		},
	},
}

var queryParameterLimit = openapi3.ParameterOrRef{
	Parameter: &openapi3.Parameter{
		Name:        request.QueryParamLimit,
		In:          openapi3.ParameterInQuery,
		Description: ptr.String("The maximum number of results to return."),
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

var queryParameterAfter = openapi3.ParameterOrRef{
	Parameter: &openapi3.Parameter{
		Name:        request.QueryParamAfter,
		In:          openapi3.ParameterInQuery,
		Description: ptr.String("The result should contain only entries created at and after this timestamp (unix millis)."),
		Required:    ptr.Bool(false),
		Schema: &openapi3.SchemaOrRef{
			Schema: &openapi3.Schema{
				Type:    ptrSchemaType(openapi3.SchemaTypeInteger),
				Minimum: ptr.Float64(0),
			},
		},
	},
}
