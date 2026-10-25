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

	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/gotidy/ptr"
	"github.com/swaggest/openapi-go/openapi3"
)

type insightsProjectRequest struct {
	repoRequest
	ProjectID int64 `path:"project_id"`
}

type createInsightsProjectRequest struct {
	repoRequest
	types.InsightsProjectCreateInput
}

type updateInsightsProjectRequest struct {
	insightsProjectRequest
	types.InsightsProjectUpdateInput
}

type insightsIssueRequest struct {
	repoRequest
	IssueID int64 `path:"issue_id"`
}

type updateInsightsIssueRequest struct {
	insightsIssueRequest
	types.InsightsIssueUpdateInput
}

var (
	queryParameterInsightsProject = taskListQueryParameter(request.QueryParamInsightsProject,
		"The id of the Insights project the issues belong to.",
		&openapi3.Schema{Type: ptrSchemaType(openapi3.SchemaTypeInteger)})
	queryParameterInsightsStatus = taskListQueryParameter(request.QueryParamInsightsStatus,
		"The comma separated statuses of the issues to include in the result.",
		taskArrayOf(&openapi3.Schema{
			Type: ptrSchemaType(openapi3.SchemaTypeString),
			Enum: enum.InsightsIssueStatus("").Enum(),
		}))
	queryParameterInsightsLevel = taskListQueryParameter(request.QueryParamInsightsLevel,
		"The level (for example error or warning) of the issues to include in the result.",
		&openapi3.Schema{Type: ptrSchemaType(openapi3.SchemaTypeString)})
	queryParameterInsightsRelease = taskListQueryParameter(request.QueryParamInsightsRelease,
		"Only issues first or last seen in this release.",
		&openapi3.Schema{Type: ptrSchemaType(openapi3.SchemaTypeString)})
	queryParameterInsightsSort = taskListQueryParameter(request.QueryParamSort,
		"The attribute the issues are sorted by.",
		&openapi3.Schema{
			Type:    ptrSchemaType(openapi3.SchemaTypeString),
			Default: ptrptr(string(enum.InsightsIssueSortLastSeen)),
			Enum: []any{
				ptr.String(string(enum.InsightsIssueSortLastSeen)),
				ptr.String(string(enum.InsightsIssueSortTimesSeen)),
			},
		})
)

//nolint:funlen
func insightsOperations(reflector *openapi3.Reflector) {
	listInsightsProjects := openapi3.Operation{}
	listInsightsProjects.WithTags("insights")
	listInsightsProjects.WithMapOfAnything(map[string]any{"operationId": "listInsightsProjects"})
	_ = reflector.SetRequest(&listInsightsProjects, new(repoRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&listInsightsProjects, new([]types.InsightsProjectWithDSN), http.StatusOK)
	setTaskErrorResponses(reflector, &listInsightsProjects, true)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/repos/{repo_ref}/insights/projects", listInsightsProjects)

	createInsightsProject := openapi3.Operation{}
	createInsightsProject.WithTags("insights")
	createInsightsProject.WithMapOfAnything(map[string]any{"operationId": "createInsightsProject"})
	_ = reflector.SetRequest(&createInsightsProject, new(createInsightsProjectRequest), http.MethodPost)
	_ = reflector.SetJSONResponse(&createInsightsProject, new(types.InsightsProjectWithDSN), http.StatusCreated)
	setTaskErrorResponses(reflector, &createInsightsProject, true)
	_ = reflector.Spec.AddOperation(http.MethodPost, "/repos/{repo_ref}/insights/projects", createInsightsProject)

	updateInsightsProject := openapi3.Operation{}
	updateInsightsProject.WithTags("insights")
	updateInsightsProject.WithMapOfAnything(map[string]any{"operationId": "updateInsightsProject"})
	_ = reflector.SetRequest(&updateInsightsProject, new(updateInsightsProjectRequest), http.MethodPatch)
	_ = reflector.SetJSONResponse(&updateInsightsProject, new(types.InsightsProjectWithDSN), http.StatusOK)
	setTaskErrorResponses(reflector, &updateInsightsProject, true)
	_ = reflector.Spec.AddOperation(http.MethodPatch, "/repos/{repo_ref}/insights/projects/{project_id}",
		updateInsightsProject)

	deleteInsightsProject := openapi3.Operation{}
	deleteInsightsProject.WithTags("insights")
	deleteInsightsProject.WithMapOfAnything(map[string]any{"operationId": "deleteInsightsProject"})
	_ = reflector.SetRequest(&deleteInsightsProject, new(insightsProjectRequest), http.MethodDelete)
	_ = reflector.SetJSONResponse(&deleteInsightsProject, nil, http.StatusNoContent)
	setTaskErrorResponses(reflector, &deleteInsightsProject, true)
	_ = reflector.Spec.AddOperation(http.MethodDelete, "/repos/{repo_ref}/insights/projects/{project_id}",
		deleteInsightsProject)

	rotateInsightsProjectKey := openapi3.Operation{}
	rotateInsightsProjectKey.WithTags("insights")
	rotateInsightsProjectKey.WithMapOfAnything(map[string]any{"operationId": "rotateInsightsProjectKey"})
	_ = reflector.SetRequest(&rotateInsightsProjectKey, new(insightsProjectRequest), http.MethodPost)
	_ = reflector.SetJSONResponse(&rotateInsightsProjectKey, new(types.InsightsProjectWithDSN), http.StatusOK)
	setTaskErrorResponses(reflector, &rotateInsightsProjectKey, true)
	_ = reflector.Spec.AddOperation(http.MethodPost, "/repos/{repo_ref}/insights/projects/{project_id}/rotate-key",
		rotateInsightsProjectKey)

	listInsightsIssues := openapi3.Operation{}
	listInsightsIssues.WithTags("insights")
	listInsightsIssues.WithMapOfAnything(map[string]any{"operationId": "listInsightsIssues"})
	listInsightsIssues.WithParameters(
		queryParameterInsightsProject, queryParameterInsightsStatus, queryParameterInsightsLevel,
		queryParameterInsightsRelease, queryParameterInsightsSort, queryParameterOrder,
		QueryParameterPage, QueryParameterLimit,
	)
	_ = reflector.SetRequest(&listInsightsIssues, new(repoRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&listInsightsIssues, new([]types.InsightsIssue), http.StatusOK)
	setTaskErrorResponses(reflector, &listInsightsIssues, true)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/repos/{repo_ref}/insights/issues", listInsightsIssues)

	getInsightsIssue := openapi3.Operation{}
	getInsightsIssue.WithTags("insights")
	getInsightsIssue.WithMapOfAnything(map[string]any{"operationId": "getInsightsIssue"})
	_ = reflector.SetRequest(&getInsightsIssue, new(insightsIssueRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&getInsightsIssue, new(types.InsightsIssue), http.StatusOK)
	setTaskErrorResponses(reflector, &getInsightsIssue, true)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/repos/{repo_ref}/insights/issues/{issue_id}", getInsightsIssue)

	updateInsightsIssue := openapi3.Operation{}
	updateInsightsIssue.WithTags("insights")
	updateInsightsIssue.WithMapOfAnything(map[string]any{"operationId": "updateInsightsIssue"})
	_ = reflector.SetRequest(&updateInsightsIssue, new(updateInsightsIssueRequest), http.MethodPatch)
	_ = reflector.SetJSONResponse(&updateInsightsIssue, new(types.InsightsIssue), http.StatusOK)
	setTaskErrorResponses(reflector, &updateInsightsIssue, true)
	_ = reflector.Spec.AddOperation(http.MethodPatch, "/repos/{repo_ref}/insights/issues/{issue_id}",
		updateInsightsIssue)

	listInsightsIssueEvents := openapi3.Operation{}
	listInsightsIssueEvents.WithTags("insights")
	listInsightsIssueEvents.WithMapOfAnything(map[string]any{"operationId": "listInsightsIssueEvents"})
	listInsightsIssueEvents.WithParameters(QueryParameterPage, QueryParameterLimit)
	_ = reflector.SetRequest(&listInsightsIssueEvents, new(insightsIssueRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&listInsightsIssueEvents, new([]types.InsightsEvent), http.StatusOK)
	setTaskErrorResponses(reflector, &listInsightsIssueEvents, true)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/repos/{repo_ref}/insights/issues/{issue_id}/events",
		listInsightsIssueEvents)
}
