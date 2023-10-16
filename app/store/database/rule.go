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

	UID         string `db:"rule_uid"`
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
		return nil, database.ProcessSQLErrorf(err, "Failed to find rule")
	}

	r := s.mapToRule(ctx, dst)

	return &r, nil
}

func (s *RuleStore) FindByUID(ctx context.Context, spaceID, repoID *int64, uid string) (*types.Rule, error) {
	stmt := database.Builder.
		Select(ruleColumns).
		From("rules").
		Where("LOWER(rule_uid) = ?", strings.ToLower(uid))
	stmt = s.applyParentID(stmt, spaceID, repoID)

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to convert find rule by UID to sql: %w", err)
	}

	db := dbtx.GetAccessor(ctx, s.db)

	dst := &rule{}
	if err = db.GetContext(ctx, dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(err, "Failed executing find rule by uid query")
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
		return database.ProcessSQLErrorf(err, "Failed to bind rule object")
	}

	if err = db.QueryRowContext(ctx, query, arg...).Scan(&dbRule.ID); err != nil {
		return database.ProcessSQLErrorf(err, "Insert rule query failed")
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
		return database.ProcessSQLErrorf(err, "Failed to bind rule object")
	}

	result, err := db.ExecContext(ctx, query, arg...)
	if err != nil {
		return database.ProcessSQLErrorf(err, "Failed to update rule")
	}

	count, err := result.RowsAffected()
	if err != nil {
		return database.ProcessSQLErrorf(err, "Failed to get number of updated rule rows")
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
		return database.ProcessSQLErrorf(err, "the delete rule query failed")
	}

	return nil
}

func (s *RuleStore) DeleteByUID(ctx context.Context, spaceID, repoID *int64, uid string) error {
	stmt := database.Builder.
		Delete("rules").
		Where("LOWER(rule_uid) = ?", strings.ToLower(uid))

	if spaceID != nil {
		stmt = stmt.Where("rule_space_id = ?", *spaceID)
	}

	if repoID != nil {
		stmt = stmt.Where("rule_repo_id = ?", *repoID)
	}

	sql, args, err := stmt.ToSql()
	if err != nil {
		return fmt.Errorf("failed to convert delete rule by UID to sql: %w", err)
	}

	db := dbtx.GetAccessor(ctx, s.db)

	if _, err = db.ExecContext(ctx, sql, args...); err != nil {
		return database.ProcessSQLErrorf(err, "Failed executing delete rule by uid query")
	}

	return nil
}

// Count returns count of protection rules matching the provided criteria.
func (s *RuleStore) Count(ctx context.Context, spaceID, repoID *int64, filter *types.RuleFilter) (int64, error) {
	stmt := database.Builder.
		Select("count(*)").
		From("rules")

	stmt = s.applyParentID(stmt, spaceID, repoID)
	stmt = s.applyFilter(stmt, filter)

	sql, args, err := stmt.ToSql()
	if err != nil {
		return 0, fmt.Errorf("failed to convert count rules query to sql: %w", err)
	}

	db := dbtx.GetAccessor(ctx, s.db)

	var count int64

	err = db.QueryRowContext(ctx, sql, args...).Scan(&count)
	if err != nil {
		return 0, database.ProcessSQLErrorf(err, "Failed executing count rules query")
	}

	return count, nil
}

// List returns a list of protection rules of a repository or a space.
func (s *RuleStore) List(ctx context.Context, spaceID, repoID *int64, filter *types.RuleFilter) ([]types.Rule, error) {
	stmt := database.Builder.
		Select(ruleColumns).
		From("rules")

	stmt = s.applyParentID(stmt, spaceID, repoID)
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
	case enum.RuleSortUID:
		stmt = stmt.OrderBy("LOWER(rule_uid) " + order.String())
	}

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to convert query to sql: %w", err)
	}

	db := dbtx.GetAccessor(ctx, s.db)

	dst := make([]rule, 0)
	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(err, "Failed executing custom list query")
	}

	return s.mapToRules(ctx, dst), nil
}

func (*RuleStore) applyParentID(
	stmt squirrel.SelectBuilder,
	spaceID, repoID *int64,
) squirrel.SelectBuilder {
	if spaceID != nil {
		stmt = stmt.Where("rule_space_id = ?", *spaceID)
	}

	if repoID != nil {
		stmt = stmt.Where("rule_repo_id = ?", *repoID)
	}

	return stmt
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
		stmt = stmt.Where("LOWER(rule_uid) LIKE ?", fmt.Sprintf("%%%s%%", strings.ToLower(filter.Query)))
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
		UID:         in.UID,
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
		UID:         in.UID,
		Description: in.Description,
		Type:        in.Type,
		State:       in.State,
		Pattern:     string(in.Pattern),
		Definition:  string(in.Definition),
	}
}
