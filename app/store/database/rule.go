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

package database

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/harness/gitness/app/store"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/Masterminds/squirrel"
	"github.com/guregu/null"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"
)

var _ store.RuleStore = (*RuleStore)(nil)

// NewRuleStore returns a new RuleStore.
func NewRuleStore(
	db *sqlx.DB,
	pCache store.PrincipalInfoCache,
) *RuleStore {
	return &RuleStore{
		pCache: pCache,
		db:     db,
	}
}

// RuleStore implements a store.RuleStore backed by a relational database.
type RuleStore struct {
	db     *sqlx.DB
	pCache store.PrincipalInfoCache
}

type rule struct {
	ID      int64 `db:"rule_id"`
	Version int64 `db:"rule_version"`

	CreatedBy int64 `db:"rule_created_by"`
	Created   int64 `db:"rule_created"`
	Updated   int64 `db:"rule_updated"`

	SpaceID null.Int `db:"rule_space_id"`
	RepoID  null.Int `db:"rule_repo_id"`

	Identifier  string `db:"rule_uid"`
	Description string `db:"rule_description"`

	Type  types.RuleType `db:"rule_type"`
	State enum.RuleState `db:"rule_state"`

	Pattern    string `db:"rule_pattern"`
	Definition string `db:"rule_definition"`
}

const (
	ruleColumns = `
		 rule_id
		,rule_version
		,rule_created_by
		,rule_created
		,rule_updated
		,rule_space_id
		,rule_repo_id
		,rule_uid
		,rule_description
		,rule_type
		,rule_state
		,rule_pattern
		,rule_definition`

	ruleSelectBase = `
		SELECT` + ruleColumns + `
		FROM rules`
)

// Find finds the rule by id.
func (s *RuleStore) Find(ctx context.Context, id int64) (*types.Rule, error) {
	const sqlQuery = ruleSelectBase + `
		WHERE rule_id = $1`

	db := dbtx.GetAccessor(ctx, s.db)

	dst := &rule{}
	if err := db.GetContext(ctx, dst, sqlQuery, id); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find rule")
	}

	r := s.mapToRule(ctx, dst)

	return &r, nil
}

func (s *RuleStore) FindByIdentifier(
	ctx context.Context,
	parentType enum.RuleParent,
	parentID int64,
	identifier string,
) (*types.Rule, error) {
	stmt := database.Builder.
		Select(ruleColumns).
		From("rules").
		Where("LOWER(rule_uid) = ?", strings.ToLower(identifier))

	switch parentType {
	case enum.RuleParentRepo:
		stmt = stmt.Where("rule_repo_id = ?", parentID)
	case enum.RuleParentSpace:
		stmt = stmt.Where("rule_space_id = ?", parentID)
	default:
		return nil, fmt.Errorf("rule parent type '%s' is not supported", parentType)
	}

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to convert find rule by Identifier to sql: %w", err)
	}

	db := dbtx.GetAccessor(ctx, s.db)

	dst := &rule{}
	if err = db.GetContext(ctx, dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed executing find rule by identifier query")
	}

	r := s.mapToRule(ctx, dst)

	return &r, nil
}

// Create creates a new protection rule.
func (s *RuleStore) Create(ctx context.Context, rule *types.Rule) error {
	const sqlQuery = `
		INSERT INTO rules (
			 rule_version
			,rule_created_by
			,rule_created
			,rule_updated
			,rule_space_id
			,rule_repo_id
			,rule_uid
			,rule_description
			,rule_type
			,rule_state
			,rule_pattern
			,rule_definition
		) values (
			 :rule_version
			,:rule_created_by
			,:rule_created
			,:rule_updated
			,:rule_space_id
			,:rule_repo_id
			,:rule_uid
			,:rule_description
			,:rule_type
			,:rule_state
			,:rule_pattern
			,:rule_definition
		) RETURNING rule_id`

	db := dbtx.GetAccessor(ctx, s.db)

	dbRule := mapToInternalRule(rule)

	query, arg, err := db.BindNamed(sqlQuery, &dbRule)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind rule object")
	}

	if err = db.QueryRowContext(ctx, query, arg...).Scan(&dbRule.ID); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Insert rule query failed")
	}

	r := s.mapToRule(ctx, &dbRule)

	*rule = r

	return nil
}

// Update updates the protection rule details.
func (s *RuleStore) Update(ctx context.Context, rule *types.Rule) error {
	const sqlQuery = `
		UPDATE rules
		SET
			 rule_version = :rule_version
			,rule_updated = :rule_updated
			,rule_uid = :rule_uid
			,rule_description = :rule_description
			,rule_state = :rule_state
			,rule_pattern = :rule_pattern
			,rule_definition = :rule_definition
		WHERE rule_id = :rule_id AND rule_version = :rule_version - 1`

	dbRule := mapToInternalRule(rule)
	dbRule.Version++
	dbRule.Updated = time.Now().UnixMilli()

	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(sqlQuery, dbRule)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind rule object")
	}

	result, err := db.ExecContext(ctx, query, arg...)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to update rule")
	}

	count, err := result.RowsAffected()
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to get number of updated rule rows")
	}

	if count == 0 {
		return gitness_store.ErrVersionConflict
	}

	rule.Version = dbRule.Version
	rule.Updated = dbRule.Updated

	return nil
}

// Delete the protection rule.
func (s *RuleStore) Delete(ctx context.Context, id int64) error {
	const sqlQuery = `
		DELETE FROM rules
		WHERE rule_id = $1`

	db := dbtx.GetAccessor(ctx, s.db)

	if _, err := db.ExecContext(ctx, sqlQuery, id); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "the delete rule query failed")
	}

	return nil
}

// Count returns count of protection rules matching the provided criteria.
func (s *RuleStore) Count(
	ctx context.Context,
	parents []types.RuleParentInfo,
	filter *types.RuleFilter,
) (int64, error) {
	stmt := database.Builder.
		Select("count(*)").
		From("rules")

	err := selectRuleParents(parents, &stmt)
	if err != nil {
		return 0, fmt.Errorf("failed to select rule parents: %w", err)
	}

	stmt = s.applyFilter(stmt, filter)

	sql, args, err := stmt.ToSql()
	if err != nil {
		return 0, fmt.Errorf("failed to convert count rules query to sql: %w", err)
	}

	db := dbtx.GetAccessor(ctx, s.db)

	var count int64

	err = db.QueryRowContext(ctx, sql, args...).Scan(&count)
	if err != nil {
		return 0, database.ProcessSQLErrorf(ctx, err, "Failed executing count rules query")
	}

	return count, nil
}

// List returns a list of protection rules of a repository or a space.
func (s *RuleStore) List(
	ctx context.Context,
	parents []types.RuleParentInfo,
	filter *types.RuleFilter,
) ([]types.Rule, error) {
	stmt := database.Builder.
		Select(ruleColumns).
		From("rules")

	err := selectRuleParents(parents, &stmt)
	if err != nil {
		return nil, fmt.Errorf("failed to select rule parents: %w", err)
	}

	stmt = s.applyFilter(stmt, filter)

	stmt = stmt.Limit(database.Limit(filter.Size))
	stmt = stmt.Offset(database.Offset(filter.Page, filter.Size))

	order := filter.Order
	if order == enum.OrderDefault {
		order = enum.OrderAsc
	}

	switch filter.Sort {
	case enum.RuleSortCreated:
		stmt = stmt.OrderBy("rule_created " + order.String())
	case enum.RuleSortUpdated:
		stmt = stmt.OrderBy("rule_updated " + order.String())
		// TODO [CODE-1363]: remove after identifier migration.
	case enum.RuleSortUID, enum.RuleSortIdentifier:
		stmt = stmt.OrderBy("LOWER(rule_uid) " + order.String())
	}

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to convert query to sql: %w", err)
	}

	db := dbtx.GetAccessor(ctx, s.db)

	dst := make([]rule, 0)
	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed executing custom list query")
	}

	return s.mapToRules(ctx, dst), nil
}

type ruleInfo struct {
	SpacePath  string         `db:"space_path"`
	RepoPath   string         `db:"repo_path"`
	ID         int64          `db:"rule_id"`
	Identifier string         `db:"rule_uid"`
	Type       types.RuleType `db:"rule_type"`
	State      enum.RuleState `db:"rule_state"`
	Pattern    string         `db:"rule_pattern"`
	Definition string         `db:"rule_definition"`
}

// ListAllRepoRules returns a list of all protection rules that can be applied on a repository.
// This includes the rules defined directly on the repository and all those defined on the parent spaces.
func (s *RuleStore) ListAllRepoRules(ctx context.Context, repoID int64) ([]types.RuleInfoInternal, error) {
	const query = `
		WITH RECURSIVE
			repo_info(repo_id, repo_uid, repo_space_id) AS (
				SELECT repo_id, repo_uid, repo_parent_id
				FROM repositories
				WHERE repo_id = $1
			),
			space_parents(space_id, space_uid, space_parent_id) AS (
				SELECT space_id, space_uid, space_parent_id
				FROM spaces
				INNER JOIN repo_info ON repo_info.repo_space_id = spaces.space_id
				UNION ALL
				SELECT spaces.space_id, spaces.space_uid, spaces.space_parent_id
				FROM spaces
				INNER JOIN space_parents ON space_parents.space_parent_id = spaces.space_id
			),
			spaces_with_path(space_id, space_parent_id, space_uid, space_full_path) AS (
				SELECT space_id, space_parent_id, space_uid, space_uid
				FROM space_parents
				WHERE space_parent_id IS NULL
				UNION ALL
				SELECT
					space_parents.space_id,
					space_parents.space_parent_id,
					space_parents.space_uid,
					spaces_with_path.space_full_path || '/' || space_parents.space_uid
				FROM space_parents
				INNER JOIN spaces_with_path ON spaces_with_path.space_id = space_parents.space_parent_id
			)
		SELECT
			 space_full_path AS "space_path"
			,'' as "repo_path"
			,rule_id
			,rule_uid
			,rule_type
			,rule_state
			,rule_pattern
			,rule_definition
		FROM spaces_with_path
		INNER JOIN rules ON rules.rule_space_id = spaces_with_path.space_id
		WHERE rule_state IN ('active', 'monitor')
		UNION ALL
		SELECT
			 '' as "space_path"
			,space_full_path || '/' || repo_info.repo_uid AS "repo_path"
			,rule_id
			,rule_uid
			,rule_type
			,rule_state
			,rule_pattern
			,rule_definition
		FROM rules
		INNER JOIN repo_info ON repo_info.repo_id = rules.rule_repo_id
		INNER JOIN spaces_with_path ON spaces_with_path.space_id = repo_info.repo_space_id
		WHERE rule_state IN ('active', 'monitor')`

	db := dbtx.GetAccessor(ctx, s.db)

	result := make([]ruleInfo, 0)
	if err := db.SelectContext(ctx, &result, query, repoID); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed executing custom list query")
	}

	return s.mapToRuleInfos(result), nil
}

func (*RuleStore) applyFilter(
	stmt squirrel.SelectBuilder,
	filter *types.RuleFilter,
) squirrel.SelectBuilder {
	if len(filter.States) == 1 {
		stmt = stmt.Where("rule_state = ?", filter.States[0])
	} else if len(filter.States) > 1 {
		stmt = stmt.Where(squirrel.Eq{"rule_state": filter.States})
	}

	if filter.Query != "" {
		stmt = stmt.Where(PartialMatch("rule_uid", filter.Query))
	}

	return stmt
}

func (s *RuleStore) mapToRule(
	ctx context.Context,
	in *rule,
) types.Rule {
	r := types.Rule{
		ID:          in.ID,
		Version:     in.Version,
		CreatedBy:   in.CreatedBy,
		Created:     in.Created,
		Updated:     in.Updated,
		SpaceID:     in.SpaceID.Ptr(),
		RepoID:      in.RepoID.Ptr(),
		Identifier:  in.Identifier,
		Description: in.Description,
		Type:        in.Type,
		State:       in.State,
		Pattern:     json.RawMessage(in.Pattern),
		Definition:  json.RawMessage(in.Definition),
	}

	createdBy, err := s.pCache.Get(ctx, in.CreatedBy)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("failed to load rule creator")
	}

	if createdBy != nil {
		r.CreatedByInfo = *createdBy
	}

	return r
}

func (s *RuleStore) mapToRules(
	ctx context.Context,
	rules []rule,
) []types.Rule {
	res := make([]types.Rule, len(rules))
	for i := 0; i < len(rules); i++ {
		res[i] = s.mapToRule(ctx, &rules[i])
	}
	return res
}

func mapToInternalRule(in *types.Rule) rule {
	return rule{
		ID:          in.ID,
		Version:     in.Version,
		CreatedBy:   in.CreatedBy,
		Created:     in.Created,
		Updated:     in.Updated,
		SpaceID:     null.IntFromPtr(in.SpaceID),
		RepoID:      null.IntFromPtr(in.RepoID),
		Identifier:  in.Identifier,
		Description: in.Description,
		Type:        in.Type,
		State:       in.State,
		Pattern:     string(in.Pattern),
		Definition:  string(in.Definition),
	}
}

func (*RuleStore) mapToRuleInfo(in *ruleInfo) types.RuleInfoInternal {
	return types.RuleInfoInternal{
		RuleInfo: types.RuleInfo{
			SpacePath:  in.SpacePath,
			RepoPath:   in.RepoPath,
			ID:         in.ID,
			Identifier: in.Identifier,
			Type:       in.Type,
			State:      in.State,
		},
		Pattern:    json.RawMessage(in.Pattern),
		Definition: json.RawMessage(in.Definition),
	}
}

func (s *RuleStore) mapToRuleInfos(
	ruleInfos []ruleInfo,
) []types.RuleInfoInternal {
	res := make([]types.RuleInfoInternal, len(ruleInfos))
	for i := 0; i < len(ruleInfos); i++ {
		res[i] = s.mapToRuleInfo(&ruleInfos[i])
	}
	return res
}

func selectRuleParents(
	parents []types.RuleParentInfo,
	stmt *squirrel.SelectBuilder,
) error {
	var parentSelector squirrel.Or
	for _, parent := range parents {
		switch parent.Type {
		case enum.RuleParentRepo:
			parentSelector = append(parentSelector, squirrel.Eq{
				"rule_repo_id": parent.ID,
			})
		case enum.RuleParentSpace:
			parentSelector = append(parentSelector, squirrel.Eq{
				"rule_space_id": parent.ID,
			})
		default:
			return fmt.Errorf("rule parent type '%s' is not supported", parent.Type)
		}
	}

	*stmt = stmt.Where(parentSelector)

	return nil
}
