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

package request

import (
	"net/http"

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

const (
	PathParamRuleIdentifier = "rule_identifier"

	QueryParamBypassRules = "bypass_rules"
	QueryParamDryRunRules = "dry_run_rules"
)

// ParseRuleFilter extracts the protection rule query parameters from the url.
func ParseRuleFilter(r *http.Request) *types.RuleFilter {
	return &types.RuleFilter{
		ListQueryFilter: ParseListQueryFilterFromRequest(r),
		States:          parseRuleStates(r),
		Types:           parseRuleTypes(r),
		Sort:            parseRuleSort(r),
		Order:           ParseOrder(r),
	}
}

// parseRuleStates extracts the protection rule states from the url.
func parseRuleStates(r *http.Request) []enum.RuleState {
	strStates, _ := QueryParamList(r, QueryParamState)
	m := make(map[enum.RuleState]struct{}) // use map to eliminate duplicates
	for _, s := range strStates {
		if state, ok := enum.RuleState(s).Sanitize(); ok {
			m[state] = struct{}{}
		}
	}

	states := make([]enum.RuleState, 0, len(m))
	for s := range m {
		states = append(states, s)
	}

	return states
}

// parseRuleTypes extracts the rule types from the url.
func parseRuleTypes(r *http.Request) []enum.RuleType {
	strTypes, _ := QueryParamList(r, QueryParamType)
	m := make(map[enum.RuleType]struct{}) // use map to eliminate duplicates
	for _, s := range strTypes {
		if t, ok := enum.RuleType(s).Sanitize(); ok {
			m[t] = struct{}{}
		}
	}

	ruleTypes := make([]enum.RuleType, 0, len(m))
	for t := range m {
		ruleTypes = append(ruleTypes, t)
	}

	return ruleTypes
}

// GetRuleIdentifierFromPath extracts the protection rule identifier from the URL.
func GetRuleIdentifierFromPath(r *http.Request) (string, error) {
	return PathParamOrError(r, PathParamRuleIdentifier)
}

// parseRuleSort extracts the protection rule sort parameter from the URL.
func parseRuleSort(r *http.Request) enum.RuleSort {
	return enum.ParseRuleSortAttr(
		r.URL.Query().Get(QueryParamSort),
	)
}

// ParseBypassRulesFromQuery extracts the bypass rules parameter from the URL query.
func ParseBypassRulesFromQuery(r *http.Request) (bool, error) {
	return QueryParamAsBoolOrDefault(r, QueryParamBypassRules, false)
}

// ParseDryRunRulesFromQuery extracts the dry run rules parameter from the URL query.
func ParseDryRunRulesFromQuery(r *http.Request) (bool, error) {
	return QueryParamAsBoolOrDefault(r, QueryParamDryRunRules, false)
}
