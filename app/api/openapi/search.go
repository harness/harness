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

var QueryParameterQuerySearch = openapi3.ParameterOrRef{
	Parameter: &openapi3.Parameter{
		Name:        request.QueryParamQuery,
		In:          openapi3.ParameterInQuery,
		Description: ptr.String("The keyword search term."),
		Required:    ptr.Bool(true),
		Schema: &openapi3.SchemaOrRef{
			Schema: &openapi3.Schema{
				Type: ptrSchemaType(openapi3.SchemaTypeString),
			},
		},
	},
}

var QueryParameterRegexSearch = openapi3.ParameterOrRef{
	Parameter: &openapi3.Parameter{
		Name:        request.QueryParamRegex,
		In:          openapi3.ParameterInQuery,
		Description: ptr.String("Whether the search term should be interpreted as a regular expression."),
		Required:    ptr.Bool(false),
		Schema: &openapi3.SchemaOrRef{
			Schema: &openapi3.Schema{
				Type:    ptrSchemaType(openapi3.SchemaTypeBoolean),
				Default: ptrptr(false),
			},
		},
	},
}

var QueryParameterRepoPathSearch = openapi3.ParameterOrRef{
	Parameter: &openapi3.Parameter{
		Name: request.QueryParamRepoPath,
		In:   openapi3.ParameterInQuery,
		Description: ptr.String("The paths of the repositories in the space to search. " +
			"Can be repeated. Can't be combined with recursive search."),
		Required: ptr.Bool(false),
		Schema: &openapi3.SchemaOrRef{
			Schema: &openapi3.Schema{
				Type: ptrSchemaType(openapi3.SchemaTypeArray),
				Items: &openapi3.SchemaOrRef{
					Schema: &openapi3.Schema{
						Type: ptrSchemaType(openapi3.SchemaTypeString),
					},
				},
			},
		},
	},
}

func searchOperations(reflector *openapi3.Reflector) {
	const tag = "keyword_search"

	opSearchRepo := openapi3.Operation{}
	opSearchRepo.WithTags(tag)
	opSearchRepo.WithSummary("Keyword search in a repository")
	opSearchRepo.WithMapOfAnything(map[string]any{"operationId": "searchRepo"})
	opSearchRepo.WithParameters(
		QueryParameterQuerySearch, QueryParameterLimit, QueryParameterRegexSearch)
	_ = reflector.SetRequest(&opSearchRepo, new(repoRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&opSearchRepo, new(types.SearchResult), http.StatusOK)
	_ = reflector.SetJSONResponse(&opSearchRepo, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opSearchRepo, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opSearchRepo, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opSearchRepo, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opSearchRepo, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/repos/{repo_ref}/code-search", opSearchRepo)

	opSearchSpace := openapi3.Operation{}
	opSearchSpace.WithTags(tag)
	opSearchSpace.WithSummary("Keyword search in a space")
	opSearchSpace.WithMapOfAnything(map[string]any{"operationId": "searchSpace"})
	opSearchSpace.WithParameters(
		QueryParameterQuerySearch, QueryParameterLimit, QueryParameterRegexSearch,
		QueryParameterRepoPathSearch, QueryParameterRecursive)
	_ = reflector.SetRequest(&opSearchSpace, new(spaceRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&opSearchSpace, new(types.SearchResult), http.StatusOK)
	_ = reflector.SetJSONResponse(&opSearchSpace, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opSearchSpace, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opSearchSpace, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opSearchSpace, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opSearchSpace, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/spaces/{space_ref}/code-search", opSearchSpace)
}
