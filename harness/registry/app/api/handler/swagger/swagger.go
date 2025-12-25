//  Copyright 2023 Harness, Inc.
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

package swagger

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
	httpswagger "github.com/swaggo/http-swagger"
)

type Handler interface {
	http.Handler
}

func GetSwaggerHandler(base string) Handler {
	r := chi.NewRouter()
	// Generate OpenAPI specification
	swagger, err := artifact.GetSwagger()
	if err != nil {
		panic(err)
	}

	// Serve the OpenAPI specification JSON
	r.Get(
		fmt.Sprintf("%s/swagger.json", base), http.HandlerFunc(
			func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				jsonResponse, _ := json.Marshal(swagger)
				_, err2 := w.Write(jsonResponse)
				if err2 != nil {
					log.Error().Err(err2).Msg("Failed to write response")
				}
			},
		),
	)

	r.Get(
		fmt.Sprintf("%s/swagger/*", base), httpswagger.Handler(
			httpswagger.URL(fmt.Sprintf("%s/swagger.json", base)), // The url pointing to API definition
		),
	)

	return r
}
