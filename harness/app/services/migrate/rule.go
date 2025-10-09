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
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/errors"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"

	migratetypes "github.com/harness/harness-migrate/types"
	"github.com/rs/zerolog/log"
)

type Rule struct {
	ruleStore      store.RuleStore
	principalStore store.PrincipalStore
	tx             dbtx.Transactor

	DefDeserializationMap     map[migratetypes.RuleType]definitionDeserializer
	PatternDeserializationMap map[migratetypes.RuleType]patternDeserializer
}

func NewRule(
	ruleStore store.RuleStore,
	tx dbtx.Transactor,
	principalStore store.PrincipalStore,
) *Rule {
	rule := &Rule{
		ruleStore:                 ruleStore,
		principalStore:            principalStore,
		tx:                        tx,
		DefDeserializationMap:     make(map[ExternalRuleType]definitionDeserializer),
		PatternDeserializationMap: make(map[ExternalRuleType]patternDeserializer),
	}

	rule.registerDeserializers(principalStore)

	return rule
}

func (migrate Rule) Import(
	ctx context.Context,
	migrator types.Principal,
	repo *types.RepositoryCore,
	typ ExternalRuleType,
	extRules []*ExternalRule,
) ([]*types.Rule, error) {
	rules := make([]*types.Rule, len(extRules))
	for i, extRule := range extRules {
		if err := check.Identifier(extRule.Identifier); err != nil {
			return nil, fmt.Errorf("branch rule identifier '%s' is invalid: %w", extRule.Identifier, err)
		}

		def, err := migrate.DefDeserializationMap[typ](ctx, string(extRule.Definition))
		if err != nil {
			return nil, fmt.Errorf("failed to deserialize rule definition: %w", err)
		}

		if err = def.Sanitize(); err != nil {
			return nil, fmt.Errorf("provided rule definition is invalid: %w", err)
		}

		definitionJSON, err := json.Marshal(def)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal rule definition: %w", err)
		}

		pattern, err := migrate.PatternDeserializationMap[typ](ctx, string(extRule.Pattern))
		if err != nil {
			return nil, fmt.Errorf("failed to deserialize rule pattern: %w", err)
		}

		if err = pattern.Validate(); err != nil {
			return nil, fmt.Errorf("provided rule pattern is invalid: %w", err)
		}

		now := time.Now().UnixMilli()
		r := &types.Rule{
			CreatedBy:  migrator.ID,
			Created:    now,
			Updated:    now,
			RepoID:     &repo.ID,
			SpaceID:    nil,
			Type:       protection.TypeBranch,
			State:      enum.RuleState(extRule.State),
			Identifier: extRule.Identifier,
			Pattern:    pattern.JSON(),
			Definition: json.RawMessage(definitionJSON),
		}
		rules[i] = r
	}

	err := migrate.tx.WithTx(ctx, func(ctx context.Context) error {
		for _, rule := range rules {
			err := migrate.ruleStore.Create(ctx, rule)
			if err != nil {
				return fmt.Errorf("failed to create branch rule: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to store external branch rules: %w", err)
	}

	return rules, nil
}

func mapToBranchRules(
	ctx context.Context,
	rule ExternalDefinition,
	principalStore store.PrincipalStore,
) (*protection.Branch, error) {
	// map users
	var userIDs []int64
	for _, email := range rule.Bypass.UserEmails {
		principal, err := principalStore.FindByEmail(ctx, email)
		if err != nil && !errors.Is(err, gitness_store.ErrResourceNotFound) {
			return nil, fmt.Errorf("failed to find principal by email for '%s': %w", email, err)
		}

		if errors.Is(err, gitness_store.ErrResourceNotFound) {
			log.Ctx(ctx).Warn().Msgf("skipping principal '%s' on bypass list", email)
			continue
		}

		userIDs = append(userIDs, principal.ID)
	}

	return &protection.Branch{
		Bypass: protection.DefBypass{
			UserIDs:    userIDs,
			RepoOwners: rule.Bypass.RepoOwners,
		},

		PullReq: protection.DefPullReq{
			Approvals: protection.DefApprovals{
				RequireCodeOwners:                  rule.PullReq.Approvals.RequireCodeOwners,
				RequireMinimumCount:                rule.PullReq.Approvals.RequireMinimumCount,
				RequireLatestCommit:                rule.PullReq.Approvals.RequireLatestCommit,
				RequireNoChangeRequest:             rule.PullReq.Approvals.RequireNoChangeRequest,
				RequireMinimumDefaultReviewerCount: 0,
			},
			Comments: protection.DefComments(rule.PullReq.Comments),
			Merge: protection.DefMerge{
				StrategiesAllowed: convertMergeMethods(rule.PullReq.Merge.StrategiesAllowed),
				DeleteBranch:      rule.PullReq.Merge.DeleteBranch,
				Block:             rule.PullReq.Merge.Block,
			},
		},

		Lifecycle: protection.DefBranchLifecycle{
			DefLifecycle: protection.DefLifecycle{
				CreateForbidden:      rule.Lifecycle.CreateForbidden,
				DeleteForbidden:      rule.Lifecycle.DeleteForbidden,
				UpdateForceForbidden: rule.Lifecycle.UpdateForceForbidden,
			},
			UpdateForbidden: rule.Lifecycle.UpdateForbidden,
		},
	}, nil
}

func convertMergeMethods(vals []string) []enum.MergeMethod {
	res := make([]enum.MergeMethod, len(vals))
	for i := range vals {
		res[i] = enum.MergeMethod(vals[i])
	}
	return res
}
