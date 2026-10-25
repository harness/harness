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

	"github.com/harness/gitness/app/api/controller/shield"
	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/gotidy/ptr"
	"github.com/swaggest/openapi-go/openapi3"
)

type createShieldScanRequest struct {
	repoRequest
	shield.ScanInput
}

type shieldFindingRequest struct {
	repoRequest
	FindingID int64 `path:"finding_id"`
}

type updateShieldFindingRequest struct {
	shieldFindingRequest
	shield.FindingUpdateInput
}

var (
	queryParameterShieldFindingState = taskListQueryParameter(request.QueryParamState,
		"The states of the findings to include in the result.",
		taskArrayOf(&openapi3.Schema{
			Type: ptrSchemaType(openapi3.SchemaTypeString),
			Enum: enum.ShieldFindingState("").Enum(),
		}))
	queryParameterShieldFindingKind = taskListQueryParameter(request.QueryParamKind,
		"The kinds of the findings to include in the result.",
		taskArrayOf(&openapi3.Schema{
			Type: ptrSchemaType(openapi3.SchemaTypeString),
			Enum: enum.ShieldFindingKind("").Enum(),
		}))
	queryParameterShieldMinRisk = taskListQueryParameter("min_risk",
		"The minimum risk score (0-100) of the findings to include in the result.",
		&openapi3.Schema{
			Type:    ptrSchemaType(openapi3.SchemaTypeInteger),
			Minimum: ptr.Float64(0),
			Maximum: ptr.Float64(100),
		})
)

//nolint:funlen
func shieldOperations(reflector *openapi3.Reflector) {
	listShieldScans := openapi3.Operation{}
	listShieldScans.WithTags("shield")
	listShieldScans.WithMapOfAnything(map[string]any{"operationId": "listShieldScans"})
	listShieldScans.WithParameters(QueryParameterPage, QueryParameterLimit)
	_ = reflector.SetRequest(&listShieldScans, new(repoRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&listShieldScans, new([]types.ShieldScan), http.StatusOK)
	setTaskErrorResponses(reflector, &listShieldScans, true)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/repos/{repo_ref}/shield/scans", listShieldScans)

	createShieldScan := openapi3.Operation{}
	createShieldScan.WithTags("shield")
	createShieldScan.WithMapOfAnything(map[string]any{"operationId": "createShieldScan"})
	_ = reflector.SetRequest(&createShieldScan, new(createShieldScanRequest), http.MethodPost)
	_ = reflector.SetJSONResponse(&createShieldScan, new(types.ShieldScan), http.StatusAccepted)
	setTaskErrorResponses(reflector, &createShieldScan, true)
	_ = reflector.Spec.AddOperation(http.MethodPost, "/repos/{repo_ref}/shield/scans", createShieldScan)

	listShieldFindings := openapi3.Operation{}
	listShieldFindings.WithTags("shield")
	listShieldFindings.WithMapOfAnything(map[string]any{"operationId": "listShieldFindings"})
	listShieldFindings.WithParameters(
		queryParameterShieldFindingState, queryParameterShieldFindingKind, queryParameterShieldMinRisk,
		QueryParameterPage, QueryParameterLimit,
	)
	_ = reflector.SetRequest(&listShieldFindings, new(repoRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&listShieldFindings, new([]types.ShieldFinding), http.StatusOK)
	setTaskErrorResponses(reflector, &listShieldFindings, true)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/repos/{repo_ref}/shield/findings", listShieldFindings)

	updateShieldFinding := openapi3.Operation{}
	updateShieldFinding.WithTags("shield")
	updateShieldFinding.WithMapOfAnything(map[string]any{"operationId": "updateShieldFinding"})
	_ = reflector.SetRequest(&updateShieldFinding, new(updateShieldFindingRequest), http.MethodPatch)
	_ = reflector.SetJSONResponse(&updateShieldFinding, new(types.ShieldFinding), http.StatusOK)
	setTaskErrorResponses(reflector, &updateShieldFinding, true)
	_ = reflector.Spec.AddOperation(http.MethodPatch, "/repos/{repo_ref}/shield/findings/{finding_id}",
		updateShieldFinding)
}
