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
	"fmt"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// NewManager creates new protection Manager.
func NewManager(ruleStore store.RuleStore) *Manager {
	return &Manager{
		defGenMap: make(map[types.RuleType]DefinitionGenerator),
		ruleStore: ruleStore,
	}
}

// DefinitionGenerator is the function that creates blank rules.
type DefinitionGenerator func() Definition

// Manager is used to enforce protection rules.
type Manager struct {
	defGenMap map[types.RuleType]DefinitionGenerator
	ruleStore store.RuleStore
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

	return toJSON(r)
}

func (m *Manager) Space(ctx context.Context, spaceID int64, ruleStates ...enum.RuleState) (Protection, error) {
	// TODO: Include the rules of the parent spaces if any.
	// TODO: Use some other function to fetch the rules, that returns just basic info about the rule

	rules, err := m.ruleStore.List(ctx, &spaceID, nil, &types.RuleFilter{
		ListQueryFilter: types.ListQueryFilter{
			Pagination: types.Pagination{
				Page: 1,
				Size: 1000,
			},
			Query: "",
		},
		States: ruleStates,
		Sort:   enum.RuleSortCreated,
		Order:  enum.OrderAsc,
	})
	if err != nil {
		return ruleSet{}, fmt.Errorf("failed to list rules for space: %w", err)
	}

	return ruleSet{
		rules:   rules,
		manager: m,
	}, nil
}

func (m *Manager) Repo(ctx context.Context, repoID int64, ruleStates ...enum.RuleState) (Protection, error) {
	// TODO: Include rules of the space and its parent spaces.
	// TODO: Use some other function to fetch the rules, that returns just basic info about the rule

	rules, err := m.ruleStore.List(ctx, nil, &repoID, &types.RuleFilter{
		ListQueryFilter: types.ListQueryFilter{
			Pagination: types.Pagination{
				Page: 1,
				Size: 1000,
			},
			Query: "",
		},
		States: ruleStates,
		Sort:   enum.RuleSortCreated,
		Order:  enum.OrderAsc,
	})
	if err != nil {
		return ruleSet{}, fmt.Errorf("failed to list rules for repository: %w", err)
	}

	return ruleSet{
		rules:   rules,
		manager: m,
	}, nil
}
