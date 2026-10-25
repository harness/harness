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

	"github.com/harness/gitness/app/api/controller/aiagent"

	"github.com/swaggest/openapi-go/openapi3"
)

type aiAgentPanelRequest struct {
	spaceRequest
}

func aiAgentOperations(reflector *openapi3.Reflector) {
	panel := openapi3.Operation{}
	panel.WithTags("space")
	panel.WithMapOfAnything(map[string]any{"operationId": "getSpaceAIAgentPanel"})
	_ = reflector.SetRequest(&panel, new(aiAgentPanelRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&panel, new(aiagent.Panel), http.StatusOK)
	setTaskErrorResponses(reflector, &panel, true)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/spaces/{space_ref}/ai/agent-panel", panel)
}
