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

package enum

import (
	"strings"
)

// RuleType represents rule type.
type RuleType string

// RuleType enumeration.
const (
	RuleTypeBranch RuleType = "branch"
	RuleTypeTag    RuleType = "tag"
	RuleTypePush   RuleType = "push"
)

var ruleTypes = sortEnum([]RuleType{
	RuleTypeBranch,
	RuleTypeTag,
	RuleTypePush,
})

func (RuleType) Enum() []interface{} { return toInterfaceSlice(ruleTypes) }
func (s RuleType) Sanitize() (RuleType, bool) {
	return Sanitize(s, GetAllRuleTypes)
}
func GetAllRuleTypes() ([]RuleType, RuleType) {
	return ruleTypes, ""
}

// RuleState represents rule state.
type RuleState string

// RuleState enumeration.
const (
	RuleStateActive   RuleState = "active"
	RuleStateMonitor  RuleState = "monitor"
	RuleStateDisabled RuleState = "disabled"
)

var ruleStates = sortEnum([]RuleState{
	RuleStateActive,
	RuleStateMonitor,
	RuleStateDisabled,
})

func (RuleState) Enum() []interface{} { return toInterfaceSlice(ruleStates) }
func (s RuleState) Sanitize() (RuleState, bool) {
	return Sanitize(s, GetAllRuleStates)
}
func GetAllRuleStates() ([]RuleState, RuleState) {
	return ruleStates, RuleStateActive
}

// RuleSort contains protection rule sorting options.
type RuleSort string

const (
	// TODO [CODE-1363]: remove after identifier migration.
	RuleSortUID        RuleSort = uid
	RuleSortIdentifier RuleSort = identifier
	RuleSortCreated    RuleSort = createdAt
	RuleSortUpdated    RuleSort = updatedAt
)

var ruleSorts = sortEnum([]RuleSort{
	// TODO [CODE-1363]: remove after identifier migration.
	RuleSortUID,
	RuleSortIdentifier,
	RuleSortCreated,
	RuleSortUpdated,
})

func (RuleSort) Enum() []interface{} { return toInterfaceSlice(ruleSorts) }
func (s RuleSort) Sanitize() (RuleSort, bool) {
	return Sanitize(s, GetAllRuleSorts)
}
func GetAllRuleSorts() ([]RuleSort, RuleSort) {
	return ruleSorts, RuleSortCreated
}

// ParseRuleSortAttr parses the protection rule sorting option.
func ParseRuleSortAttr(s string) RuleSort {
	switch strings.ToLower(s) {
	// TODO [CODE-1363]: remove after identifier migration.
	case uid:
		return RuleSortUID
	case created, createdAt:
		return RuleSortCreated
	case updated, updatedAt:
		return RuleSortUpdated
	}

	return RuleSortIdentifier
}

// RuleParent defines different types of parents of a rule.
type RuleParent string

func (RuleParent) Enum() []interface{} { return toInterfaceSlice(RuleParents) }

const (
	// RuleParentRepo describes a repo as Rule owner.
	RuleParentRepo RuleParent = "repo"

	// RuleParentSpace describes a space as Rule owner.
	RuleParentSpace RuleParent = "space"
)

var RuleParents = sortEnum([]RuleParent{
	RuleParentRepo,
	RuleParentSpace,
})
