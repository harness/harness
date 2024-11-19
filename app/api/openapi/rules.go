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

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/app/services/rules"
	"github.com/harness/gitness/types"

	"github.com/swaggest/openapi-go/openapi3"
)

// ruleType is a plugin for types.RuleType to allow using oneof.
type ruleType string

func (ruleType) Enum() []interface{} {
	return []interface{}{protection.TypeBranch}
}

// ruleDefinition is a plugin for types.Rule Definition to allow using oneof.
type ruleDefinition struct{}

func (ruleDefinition) JSONSchemaOneOf() []interface{} {
	return []interface{}{protection.Branch{}}
}

type rule struct {
	types.Rule

	// overshadow Type and Definition to enable oneof.
	Type       ruleType       `json:"type"`
	Definition ruleDefinition `json:"definition"`

	// overshadow Pattern to correct the type
	Pattern protection.Pattern `json:"pattern"`
}

func rulesOperations(reflector *openapi3.Reflector) {
	opSpaceRuleAdd := openapi3.Operation{}
	opSpaceRuleAdd.WithTags("space")
	opSpaceRuleAdd.WithMapOfAnything(map[string]interface{}{"operationId": "spaceRuleAdd"})
	_ = reflector.SetRequest(&opSpaceRuleAdd, struct {
		spaceRequest
		rules.CreateInput

		// overshadow "definition"
		Type       ruleType       `json:"type"`
		Definition ruleDefinition `json:"definition"`
	}{}, http.MethodPost)
	_ = reflector.SetJSONResponse(&opSpaceRuleAdd, rule{}, http.StatusCreated)
	_ = reflector.SetJSONResponse(&opSpaceRuleAdd, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opSpaceRuleAdd, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opSpaceRuleAdd, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opSpaceRuleAdd, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodPost, "/spaces/{space_ref}/rules", opSpaceRuleAdd)

	opSpaceRuleDelete := openapi3.Operation{}
	opSpaceRuleDelete.WithTags("space")
	opSpaceRuleDelete.WithMapOfAnything(map[string]interface{}{"operationId": "spaceRuleDelete"})
	_ = reflector.SetRequest(&opSpaceRuleDelete, struct {
		spaceRequest
		RuleIdentifier string `path:"rule_identifier"`
	}{}, http.MethodDelete)
	_ = reflector.SetJSONResponse(&opSpaceRuleDelete, nil, http.StatusNoContent)
	_ = reflector.SetJSONResponse(&opSpaceRuleDelete, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opSpaceRuleDelete, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opSpaceRuleDelete, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opSpaceRuleDelete, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodDelete, "/spaces/{space_ref}/rules/{rule_identifier}", opSpaceRuleDelete)

	opSpaceRuleUpdate := openapi3.Operation{}
	opSpaceRuleUpdate.WithTags("space")
	opSpaceRuleUpdate.WithMapOfAnything(map[string]interface{}{"operationId": "spaceRuleUpdate"})
	_ = reflector.SetRequest(&opSpaceRuleUpdate, &struct {
		spaceRequest
		Identifier string `path:"rule_identifier"`
		rules.UpdateInput

		// overshadow Type and Definition to enable oneof.
		Type       ruleType       `json:"type"`
		Definition ruleDefinition `json:"definition"`
	}{}, http.MethodPatch)
	_ = reflector.SetJSONResponse(&opSpaceRuleUpdate, rule{}, http.StatusOK)
	_ = reflector.SetJSONResponse(&opSpaceRuleUpdate, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opSpaceRuleUpdate, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opSpaceRuleUpdate, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opSpaceRuleUpdate, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodPatch, "/spaces/{space_ref}/rules/{rule_identifier}", opSpaceRuleUpdate)

	opSpaceRuleList := openapi3.Operation{}
	opSpaceRuleList.WithTags("space")
	opSpaceRuleList.WithMapOfAnything(map[string]interface{}{"operationId": "spaceRuleList"})
	opSpaceRuleList.WithParameters(
		queryParameterQueryRuleList,
		queryParameterOrder, queryParameterSortRuleList,
		QueryParameterPage, QueryParameterLimit)
	_ = reflector.SetRequest(&opSpaceRuleList, &struct {
		spaceRequest
	}{}, http.MethodGet)
	_ = reflector.SetJSONResponse(&opSpaceRuleList, []rule{}, http.StatusOK)
	_ = reflector.SetJSONResponse(&opSpaceRuleList, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opSpaceRuleList, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opSpaceRuleList, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opSpaceRuleList, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/spaces/{space_ref}/rules", opSpaceRuleList)

	opSpaceRuleGet := openapi3.Operation{}
	opSpaceRuleGet.WithTags("space")
	opSpaceRuleGet.WithMapOfAnything(map[string]interface{}{"operationId": "spaceRuleGet"})
	_ = reflector.SetRequest(&opSpaceRuleGet, &struct {
		spaceRequest
		Identifier string `path:"rule_identifier"`
	}{}, http.MethodGet)
	_ = reflector.SetJSONResponse(&opSpaceRuleGet, rule{}, http.StatusOK)
	_ = reflector.SetJSONResponse(&opSpaceRuleGet, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opSpaceRuleGet, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opSpaceRuleGet, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opSpaceRuleGet, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/spaces/{space_ref}/rules/{rule_identifier}", opSpaceRuleGet)

	opRepoRuleAdd := openapi3.Operation{}
	opRepoRuleAdd.WithTags("repository")
	opRepoRuleAdd.WithMapOfAnything(map[string]interface{}{"operationId": "repoRuleAdd"})
	_ = reflector.SetRequest(&opRepoRuleAdd, struct {
		repoRequest
		rules.CreateInput

		// overshadow "definition"
		Type       ruleType       `json:"type"`
		Definition ruleDefinition `json:"definition"`
	}{}, http.MethodPost)
	_ = reflector.SetJSONResponse(&opRepoRuleAdd, rule{}, http.StatusCreated)
	_ = reflector.SetJSONResponse(&opRepoRuleAdd, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opRepoRuleAdd, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opRepoRuleAdd, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opRepoRuleAdd, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodPost, "/repos/{repo_ref}/rules", opRepoRuleAdd)

	opRepoRuleDelete := openapi3.Operation{}
	opRepoRuleDelete.WithTags("repository")
	opRepoRuleDelete.WithMapOfAnything(map[string]interface{}{"operationId": "repoRuleDelete"})
	_ = reflector.SetRequest(&opRepoRuleDelete, struct {
		repoRequest
		RuleIdentifier string `path:"rule_identifier"`
	}{}, http.MethodDelete)
	_ = reflector.SetJSONResponse(&opRepoRuleDelete, nil, http.StatusNoContent)
	_ = reflector.SetJSONResponse(&opRepoRuleDelete, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opRepoRuleDelete, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opRepoRuleDelete, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opRepoRuleDelete, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodDelete, "/repos/{repo_ref}/rules/{rule_identifier}", opRepoRuleDelete)

	opRepoRuleUpdate := openapi3.Operation{}
	opRepoRuleUpdate.WithTags("repository")
	opRepoRuleUpdate.WithMapOfAnything(map[string]interface{}{"operationId": "repoRuleUpdate"})
	_ = reflector.SetRequest(&opRepoRuleUpdate, &struct {
		repoRequest
		Identifier string `path:"rule_identifier"`
		rules.UpdateInput

		// overshadow Type and Definition to enable oneof.
		Type       ruleType       `json:"type"`
		Definition ruleDefinition `json:"definition"`
	}{}, http.MethodPatch)
	_ = reflector.SetJSONResponse(&opRepoRuleUpdate, rule{}, http.StatusOK)
	_ = reflector.SetJSONResponse(&opRepoRuleUpdate, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opRepoRuleUpdate, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opRepoRuleUpdate, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opRepoRuleUpdate, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodPatch, "/repos/{repo_ref}/rules/{rule_identifier}", opRepoRuleUpdate)

	opRepoRuleList := openapi3.Operation{}
	opRepoRuleList.WithTags("repository")
	opRepoRuleList.WithMapOfAnything(map[string]interface{}{"operationId": "repoRuleList"})
	opRepoRuleList.WithParameters(
		queryParameterQueryRuleList,
		queryParameterOrder, queryParameterSortRuleList,
		QueryParameterPage, QueryParameterLimit)
	_ = reflector.SetRequest(&opRepoRuleList, &struct {
		repoRequest
	}{}, http.MethodGet)
	_ = reflector.SetJSONResponse(&opRepoRuleList, []rule{}, http.StatusOK)
	_ = reflector.SetJSONResponse(&opRepoRuleList, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opRepoRuleList, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opRepoRuleList, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opRepoRuleList, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/repos/{repo_ref}/rules", opRepoRuleList)

	opRepoRuleGet := openapi3.Operation{}
	opRepoRuleGet.WithTags("repository")
	opRepoRuleGet.WithMapOfAnything(map[string]interface{}{"operationId": "repoRuleGet"})
	_ = reflector.SetRequest(&opRepoRuleGet, &struct {
		repoRequest
		Identifier string `path:"rule_identifier"`
	}{}, http.MethodGet)
	_ = reflector.SetJSONResponse(&opRepoRuleGet, rule{}, http.StatusOK)
	_ = reflector.SetJSONResponse(&opRepoRuleGet, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opRepoRuleGet, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opRepoRuleGet, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opRepoRuleGet, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/repos/{repo_ref}/rules/{rule_identifier}", opRepoRuleGet)
}
