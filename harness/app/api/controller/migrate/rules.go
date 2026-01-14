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
	"fmt"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/services/migrate"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type RulesInput struct {
	Rules []*migrate.ExternalRule  `json:"rules"`
	Type  migrate.ExternalRuleType `json:"type"`
}

func (c *Controller) Rules(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	in *RulesInput,
) ([]*types.Rule, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoEdit)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire access to repo: %w", err)
	}

	if repo.State != enum.RepoStateMigrateDataImport {
		return nil, fmt.Errorf("repo state is %s want %s: %w",
			repo.State, enum.RepoStateMigrateDataImport, errInvalidRepoState)
	}

	rulesOut, err := c.ruleImporter.Import(ctx, session.Principal, repo, in.Type, in.Rules)
	if err != nil {
		return nil, fmt.Errorf("failed to import rules: %w", err)
	}

	return rulesOut, nil
}
