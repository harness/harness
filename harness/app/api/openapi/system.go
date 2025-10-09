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

	"github.com/harness/gitness/app/api/handler/system"
	"github.com/harness/gitness/app/api/usererror"

	"github.com/swaggest/openapi-go/openapi3"
)

// helper function that constructs the openapi specification
// for the system registration config endpoints.
func buildSystem(reflector *openapi3.Reflector) {
	opGetConfig := openapi3.Operation{}
	opGetConfig.WithTags("system")
	opGetConfig.WithMapOfAnything(map[string]interface{}{"operationId": "getSystemConfig"})
	_ = reflector.SetRequest(&opGetConfig, nil, http.MethodGet)
	_ = reflector.SetJSONResponse(&opGetConfig, new(system.ConfigOutput), http.StatusOK)
	_ = reflector.SetJSONResponse(&opGetConfig, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opGetConfig, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/system/config", opGetConfig)
}
