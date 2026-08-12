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

package checkreq

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

const (
	// CodeStatusChecksRequired is reported when a required check has not succeeded.
	CodeStatusChecksRequired = "pullreq.status_checks.external_required_identifiers"

	// RuleIdentifier is the synthetic rule identity carried by these violations.
	// They do not originate from a protection rule, but the merge API models all
	// blockers as types.RuleViolations.
	RuleIdentifier = "external_status_checks"
)

// Resolved is the requirement set of a pull request, indexed for lookup.
type Resolved struct {
	required   map[string]struct{}
	bypassable map[string]struct{}
}

// Resolve asks the provider for the requirements of the given pull request.
func Resolve(
	ctx context.Context,
	provider Provider,
	repo *types.RepositoryCore,
	pr *types.PullReq,
) (Resolved, error) {
	if provider == nil {
		return Resolved{}, nil
	}

	requirements, err := provider.RequiredChecks(ctx, repo, pr)
	if err != nil {
		return Resolved{}, fmt.Errorf("failed to resolve required checks: %w", err)
	}

	out := Resolved{
		required:   make(map[string]struct{}, len(requirements.Required)),
		bypassable: make(map[string]struct{}, len(requirements.Bypassable)),
	}
	for _, identifier := range requirements.Required {
		out.required[identifier] = struct{}{}
	}
	for _, identifier := range requirements.Bypassable {
		// A check declared both required and bypassable is treated as
		// non-bypassable - the stricter of the two declarations wins.
		if _, ok := out.required[identifier]; ok {
			continue
		}
		out.bypassable[identifier] = struct{}{}
	}

	return out, nil
}

// Identifiers returns every required check identifier.
func (r Resolved) Identifiers() []string {
	out := make([]string, 0, len(r.required)+len(r.bypassable))
	for identifier := range r.required {
		out = append(out, identifier)
	}
	for identifier := range r.bypassable {
		out = append(out, identifier)
	}
	sort.Strings(out)
	return out
}

// Requirement reports how the given check identifier is required, if at all.
func (r Resolved) Requirement(identifier string) (required, bypassable bool) {
	if _, ok := r.required[identifier]; ok {
		return true, false
	}
	if _, ok := r.bypassable[identifier]; ok {
		return true, true
	}
	return false, false
}

// Unsatisfied returns the required checks that the given results do not satisfy,
// sorted by identifier. A required check with no result at all is unsatisfied.
func (r Resolved) Unsatisfied(results []types.CheckResult) []string {
	if len(r.required) == 0 && len(r.bypassable) == 0 {
		return nil
	}

	byIdentifier := make(map[string]types.CheckResult, len(results))
	for i := range results {
		byIdentifier[results[i].Identifier] = results[i]
	}

	var unsatisfied []string
	for identifier := range r.required {
		result, ok := byIdentifier[identifier]
		// An explicit bypass on the check does not satisfy a non-bypassable
		// requirement.
		if !ok || !result.Status.IsSuccess() {
			unsatisfied = append(unsatisfied, identifier)
		}
	}
	for identifier := range r.bypassable {
		result, ok := byIdentifier[identifier]
		if !ok || (!result.Status.IsSuccess() && result.BypassedBy == nil) {
			unsatisfied = append(unsatisfied, identifier)
		}
	}

	sort.Strings(unsatisfied)
	return unsatisfied
}

// Violations reports the unsatisfied required checks as merge blockers.
func Violations(unsatisfied []string) []types.RuleViolations {
	if len(unsatisfied) == 0 {
		return nil
	}

	violations := types.RuleViolations{
		Rule: types.RuleInfo{
			Identifier: RuleIdentifier,
			State:      enum.RuleStateActive,
		},
		Bypassable: false,
		Bypassed:   false,
	}
	violations.Addf(
		CodeStatusChecksRequired,
		"The following status checks are required to be completed successfully: %s",
		strings.Join(unsatisfied, ", "),
	)

	return []types.RuleViolations{violations}
}
