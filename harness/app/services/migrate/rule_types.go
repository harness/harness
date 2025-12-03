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

package migrate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/app/store"

	migratetypes "github.com/harness/harness-migrate/types"
)

const ExternalRuleTypeBranch = migratetypes.RuleTypeBranch

type (
	ExternalRuleType = migratetypes.RuleType
	ExternalRule     = migratetypes.Rule

	ExternalDefinition    = migratetypes.Definition
	ExternalBranchPattern = migratetypes.BranchPattern

	definitionDeserializer func(context.Context, string) (protection.Definition, error)
	patternDeserializer    func(context.Context, string) (*protection.Pattern, error)
)

func (migrate *Rule) registerDeserializers(principalStore store.PrincipalStore) {
	// banch rules definition deserializer
	migrate.DefDeserializationMap[ExternalRuleTypeBranch] = func(
		ctx context.Context,
		rawDef string,
	) (protection.Definition, error) {
		// deserialize string into external branch rule type
		var extrDef ExternalDefinition

		decoder := json.NewDecoder(bytes.NewReader([]byte(rawDef)))
		if err := decoder.Decode(&extrDef); err != nil {
			return nil, fmt.Errorf("failed to decode external branch rule definition: %w", err)
		}

		rule, err := mapToBranchRules(ctx, extrDef, principalStore)
		if err != nil {
			return nil, fmt.Errorf("failed to map external branch rule definition to internal: %w", err)
		}

		return rule, nil
	}

	// branch rules pattern deserializer
	migrate.PatternDeserializationMap[ExternalRuleTypeBranch] = func(
		_ context.Context,
		rawDef string,
	) (*protection.Pattern, error) {
		var extrPattern ExternalBranchPattern

		decoder := json.NewDecoder(bytes.NewReader([]byte(rawDef)))
		if err := decoder.Decode(&extrPattern); err != nil {
			return nil, fmt.Errorf("failed to decode external branch rule pattern: %w", err)
		}

		return &protection.Pattern{
			Default: extrPattern.Default,
			Include: extrPattern.Include,
			Exclude: extrPattern.Exclude,
		}, nil
	}
}
