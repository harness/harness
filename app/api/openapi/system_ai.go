// Copyright 2026 Harness, Inc.
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

	"github.com/harness/gitness/app/api/controller/system"

	"github.com/swaggest/openapi-go/openapi3"
)

func systemAIOperations(reflector *openapi3.Reflector) {
	listAIModels := openapi3.Operation{}
	listAIModels.WithTags("system")
	listAIModels.WithMapOfAnything(map[string]any{"operationId": "listSystemAIModels"})
	_ = reflector.SetRequest(&listAIModels, nil, http.MethodGet)
	_ = reflector.SetJSONResponse(&listAIModels, new(system.AIModelsOutput), http.StatusOK)
	setTaskErrorResponses(reflector, &listAIModels, false)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/system/ai/models", listAIModels)

	getAIQuota := openapi3.Operation{}
	getAIQuota.WithTags("system")
	getAIQuota.WithMapOfAnything(map[string]any{"operationId": "getSystemAIQuota"})
	_ = reflector.SetRequest(&getAIQuota, nil, http.MethodGet)
	_ = reflector.SetJSONResponse(&getAIQuota, new(system.AIQuotaOutput), http.StatusOK)
	setTaskErrorResponses(reflector, &getAIQuota, false)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/system/ai/quota", getAIQuota)
}
