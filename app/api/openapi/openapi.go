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
	"github.com/harness/gitness/app/config"
	"github.com/harness/gitness/version"

	"github.com/swaggest/openapi-go/openapi3"
)

type (
	paginationRequest struct {
		Page int `query:"page"     default:"1"`
		Size int `query:"limit"    default:"30"`
	}
)

var _ Service = (*OpenAPI)(nil)

type OpenAPI struct{}

func NewOpenAPIService() *OpenAPI {
	return &OpenAPI{}
}

// Generate is a helper function that constructs the
// openapi specification object, which can be marshaled
// to json or yaml, as needed.
func (*OpenAPI) Generate() *openapi3.Spec {
	reflector := openapi3.Reflector{}
	reflector.Spec = &openapi3.Spec{Openapi: "3.0.0"}
	reflector.Spec.Info.
		WithTitle("API Specification").
		WithVersion(version.Version.String())
	reflector.Spec.Servers = []openapi3.Server{{
		URL: config.APIURL,
	}}

	//
	// register endpoints
	//

	buildSystem(&reflector)
	buildAccount(&reflector)
	buildUser(&reflector)
	buildAdmin(&reflector)
	buildPrincipals(&reflector)
	spaceOperations(&reflector)
	pluginOperations(&reflector)
	repoOperations(&reflector)
	pipelineOperations(&reflector)
	connectorOperations(&reflector)
	templateOperations(&reflector)
	secretOperations(&reflector)
	resourceOperations(&reflector)
	pullReqOperations(&reflector)
	webhookOperations(&reflector)
	checkOperations(&reflector)
	uploadOperations(&reflector)
	gitspaceOperations(&reflector)
	infraProviderOperations(&reflector)

	//
	// define security scheme
	//

	scheme := openapi3.SecuritySchemeOrRef{
		SecurityScheme: &openapi3.SecurityScheme{
			HTTPSecurityScheme: &openapi3.HTTPSecurityScheme{
				Scheme: "bearerAuth",
				Bearer: &openapi3.Bearer{},
			},
		},
	}
	security := openapi3.ComponentsSecuritySchemes{}
	security.WithMapOfSecuritySchemeOrRefValuesItem("bearerAuth", scheme)
	reflector.Spec.Components.WithSecuritySchemes(security)

	//
	// enforce security scheme globally
	//

	reflector.Spec.WithSecurity(map[string][]string{
		"bearerAuth": {},
	})

	return reflector.Spec
}

func panicOnErr(err error) {
	if err != nil {
		panic(err)
	}
}
