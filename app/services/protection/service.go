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

package protection

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type (
	Sanitizer interface {
		// Sanitize validates if the definition is valid and automatically corrects minor issues.
		Sanitize() error
	}

	Protection interface {
		MergeVerifier
		RefChangeVerifier
		CreatePullReqVerifier
		UserIDs() ([]int64, error)
		UserGroupIDs() ([]int64, error)
	}

	Definition interface {
		Sanitizer
		Protection
	}

	// DefinitionGenerator is the function that creates blank rules.
	DefinitionGenerator func() Definition

	// Manager is used to enforce protection rules.
	Manager struct {
		defGenMap map[types.RuleType]DefinitionGenerator
		ruleStore store.RuleStore
	}
)

var (
	ErrUnrecognizedType       = errors.New("unrecognized protection type")
	ErrAlreadyRegistered      = errors.New("protection type already registered")
	ErrPatternEmpty           = errors.New("name pattern can't be empty")
	ErrInvalidGlobstarPattern = errors.New("invalid globstar pattern")
)

func IsCritical(violations []types.RuleViolations) bool {
	for i := range violations {
		if violations[i].IsCritical() {
			return true
		}
	}
	return false
}

func IsBypassed(violations []types.RuleViolations) bool {
	for i := range violations {
		if violations[i].IsBypassed() {
			return true
		}
	}
	return false
}

// NewManager creates new protection Manager.
func NewManager(ruleStore store.RuleStore) *Manager {
	return &Manager{
		defGenMap: make(map[types.RuleType]DefinitionGenerator),
		ruleStore: ruleStore,
	}
}

// Register registers new types.RuleType.
func (m *Manager) Register(ruleType types.RuleType, gen DefinitionGenerator) error {
	_, ok := m.defGenMap[ruleType]
	if ok {
		return ErrAlreadyRegistered
	}

	m.defGenMap[ruleType] = gen

	return nil
}

func (m *Manager) FromJSON(ruleType types.RuleType, message json.RawMessage, strict bool) (Protection, error) {
	gen := m.defGenMap[ruleType]
	if gen == nil {
		return nil, ErrUnrecognizedType
	}

	decoder := json.NewDecoder(bytes.NewReader(message))

	if strict {
		decoder.DisallowUnknownFields()
	}

	r := gen()

	if err := decoder.Decode(&r); err != nil {
		return nil, err
	}

	if err := r.Sanitize(); err != nil {
		return nil, err
	}

	return r, nil
}

func (m *Manager) SanitizeJSON(ruleType types.RuleType, message json.RawMessage) (json.RawMessage, error) {
	r, err := m.FromJSON(ruleType, message, true)
	if err != nil {
		return nil, err
	}

	return ToJSON(r)
}

func (m *Manager) ForRepository(ctx context.Context, repoID int64) (Protection, error) {
	ruleInfos, err := m.ruleStore.ListAllRepoRules(ctx, repoID)
	if err != nil {
		return nil, fmt.Errorf("failed to list rules for repository: %w", err)
	}

	return ruleSet{
		rules:   ruleInfos,
		manager: m,
	}, nil
}

// GenerateErrorMessageForBlockingViolations generates an error message for a given slice of rule violations.
// It simply takes the first blocking rule that has a violation and prints that, with indication if further
// rules were violated.
func GenerateErrorMessageForBlockingViolations(ruleViolations []types.RuleViolations) string {
	printRuleScope := func(r types.RuleInfo) string {
		switch {
		case r.RepoPath != "":
			return fmt.Sprintf("repository %q", r.RepoPath)
		case r.SpacePath != "":
			return fmt.Sprintf("space %q", r.SpacePath)
		default:
			return "unknown scope"
		}
	}

	selectedIDX := -1
	blockingRuleViolationCnt := 0
	for i := range ruleViolations {
		// we don't care about bypassed or non-active rules
		if ruleViolations[i].Bypassed || ruleViolations[i].Rule.State != enum.RuleStateActive {
			continue
		}

		blockingRuleViolationCnt++

		// We take the first blocking rule violation we find, unless a later one has additional details.
		if selectedIDX >= 0 &&
			(len(ruleViolations[selectedIDX].Violations) > 0 || len(ruleViolations[i].Violations) == 0) {
			continue
		}
		selectedIDX = i
	}

	if blockingRuleViolationCnt == 0 {
		return "No blocking rule violations found."
	}

	var msg string
	if blockingRuleViolationCnt == 1 {
		msg = fmt.Sprintf(
			"Operation violates %s protection rule %q in %s",
			ruleViolations[selectedIDX].Rule.Type,
			ruleViolations[selectedIDX].Rule.Identifier,
			printRuleScope(ruleViolations[selectedIDX].Rule),
		)
	} else {
		msg = fmt.Sprintf(
			"Operation violates %d protection rules, including %s protection rule %q in %s",
			blockingRuleViolationCnt,
			ruleViolations[selectedIDX].Rule.Type,
			ruleViolations[selectedIDX].Rule.Identifier,
			printRuleScope(ruleViolations[selectedIDX].Rule),
		)
	}

	if len(ruleViolations[selectedIDX].Violations) > 0 {
		msg += " with violation: " + ruleViolations[selectedIDX].Violations[0].Message
	}

	return msg
}
