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
	"github.com/harness/gitness/app/api/request"
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

var QueryParameterPage = openapi3.ParameterOrRef{
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

var QueryParameterLimit = openapi3.ParameterOrRef{
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

var queryParameterCreatedLt = openapi3.ParameterOrRef{
	Parameter: &openapi3.Parameter{
		Name:        request.QueryParamCreatedLt,
		In:          openapi3.ParameterInQuery,
		Description: ptr.String("The result should contain only entries created before this timestamp (unix millis)."),
		Required:    ptr.Bool(false),
		Schema: &openapi3.SchemaOrRef{
			Schema: &openapi3.Schema{
				Type:    ptrSchemaType(openapi3.SchemaTypeInteger),
				Minimum: ptr.Float64(0),
			},
		},
	},
}

var queryParameterCreatedGt = openapi3.ParameterOrRef{
	Parameter: &openapi3.Parameter{
		Name:        request.QueryParamCreatedGt,
		In:          openapi3.ParameterInQuery,
		Description: ptr.String("The result should contain only entries created after this timestamp (unix millis)."),
		Required:    ptr.Bool(false),
		Schema: &openapi3.SchemaOrRef{
			Schema: &openapi3.Schema{
				Type:    ptrSchemaType(openapi3.SchemaTypeInteger),
				Minimum: ptr.Float64(0),
			},
		},
	},
}
