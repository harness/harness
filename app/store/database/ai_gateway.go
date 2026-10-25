// Copyright 2026 Harness, Inc.
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
	"time"

	"github.com/harness/gitness/app/store"
	gitnessstore "github.com/harness/gitness/store"
	"github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/Masterminds/squirrel"
	"github.com/guregu/null"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

const (
	aiModelTable   = `ai_models`
	aiModelColumns = `
		aimodel_id,
		aimodel_kind,
		aimodel_context_length,
		aimodel_supports_tools,
		aimodel_status,
		aimodel_success_rate_by_task,
		aimodel_p95_latency_ms,
		aimodel_last_seen,
		aimodel_quarantined_reason,
		aimodel_created,
		aimodel_updated`

	aiDecisionTable   = `ai_decisions`
	aiDecisionColumns = `
		aidecision_task_id,
		aidecision_question_id,
		aidecision_model_build,
		aidecision_state_hash,
		aidecision_answer,
		aidecision_fallback,
		aidecision_outcome,
		aidecision_created`
	aiDecisionSelectColumns = `aidecision_id, ` + aiDecisionColumns

	aiQuotaTable = `ai_quota`

	aiUsageTable   = `ai_usage`
	aiUsageColumns = `
		aiusage_space_id,
		aiusage_repo_id,
		aiusage_feature,
		aiusage_model,
		aiusage_requests,
		aiusage_tokens_in,
		aiusage_tokens_out,
		aiusage_cost_usd,
		aiusage_created`
	aiUsageSelectColumns = `aiusage_id, ` + aiUsageColumns
)

// ---------------------------------------------------------------------------
// models

type aiModel struct {
	ID                string             `db:"aimodel_id"`
	Kind              enum.AIModelKind   `db:"aimodel_kind"`
	ContextLength     int                `db:"aimodel_context_length"`
	SupportsTools     bool               `db:"aimodel_supports_tools"`
	Status            enum.AIModelStatus `db:"aimodel_status"`
	SuccessRateByTask string             `db:"aimodel_success_rate_by_task"`
	P95LatencyMS      int                `db:"aimodel_p95_latency_ms"`
	LastSeen          int64              `db:"aimodel_last_seen"`
	QuarantinedReason string             `db:"aimodel_quarantined_reason"`
	Created           int64              `db:"aimodel_created"`
	Updated           int64              `db:"aimodel_updated"`
}

type aiModelStore struct {
	db *sqlx.DB
}

// NewAIModelStore returns a new AIModelStore.
func NewAIModelStore(db *sqlx.DB) store.AIModelStore {
	return &aiModelStore{db: db}
}

var _ store.AIModelStore = (*aiModelStore)(nil)

func (s *aiModelStore) Upsert(ctx context.Context, m *types.AIModel) error {
	rates, err := json.Marshal(m.SuccessRateByTask)
	if err != nil {
		return fmt.Errorf("failed to marshal success rates: %w", err)
	}
	if m.SuccessRateByTask == nil {
		rates = []byte("{}")
	}
	now := time.Now().UnixMilli()
	if m.Created == 0 {
		m.Created = now
	}
	m.Updated = now

	stmt := database.Builder.
		Insert(aiModelTable).
		Columns(aiModelColumns).
		Values(
			m.ID, m.Kind, m.ContextLength, m.SupportsTools, m.Status, string(rates),
			m.P95LatencyMS, m.LastSeen, m.QuarantinedReason, m.Created, m.Updated,
		).
		Suffix(`ON CONFLICT (aimodel_id) DO UPDATE SET
			aimodel_kind = EXCLUDED.aimodel_kind,
			aimodel_context_length = EXCLUDED.aimodel_context_length,
			aimodel_supports_tools = EXCLUDED.aimodel_supports_tools,
			aimodel_status = EXCLUDED.aimodel_status,
			aimodel_success_rate_by_task = EXCLUDED.aimodel_success_rate_by_task,
			aimodel_p95_latency_ms = EXCLUDED.aimodel_p95_latency_ms,
			aimodel_last_seen = EXCLUDED.aimodel_last_seen,
			aimodel_quarantined_reason = EXCLUDED.aimodel_quarantined_reason,
			aimodel_updated = EXCLUDED.aimodel_updated`)

	sql, args, err := stmt.ToSql()
	if err != nil {
		return errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}
	db := dbtx.GetAccessor(ctx, s.db)
	if _, err := db.ExecContext(ctx, sql, args...); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to upsert ai model %s", m.ID)
	}
	return nil
}

func (s *aiModelStore) Find(ctx context.Context, id string) (*types.AIModel, error) {
	stmt := database.Builder.
		Select(aiModelColumns).
		From(aiModelTable).
		Where("aimodel_id = ?", id)

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}
	dst := &aiModel{}
	db := dbtx.GetAccessor(ctx, s.db)
	if err := db.GetContext(ctx, dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find ai model %s", id)
	}
	return mapAIModel(dst)
}

func (s *aiModelStore) List(ctx context.Context, filter types.AIModelFilter) ([]*types.AIModel, error) {
	stmt := database.Builder.
		Select(aiModelColumns).
		From(aiModelTable).
		OrderBy("aimodel_id ASC")
	if filter.Kind != "" {
		stmt = stmt.Where("aimodel_kind = ?", filter.Kind)
	}
	if filter.Status != "" {
		stmt = stmt.Where("aimodel_status = ?", filter.Status)
	}

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}
	var dst []*aiModel
	db := dbtx.GetAccessor(ctx, s.db)
	if err := db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to list ai models")
	}
	out := make([]*types.AIModel, 0, len(dst))
	for _, d := range dst {
		m, err := mapAIModel(d)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, nil
}

func (s *aiModelStore) UpdateStatus(
	ctx context.Context,
	id string,
	status enum.AIModelStatus,
	reason string,
) error {
	stmt := database.Builder.
		Update(aiModelTable).
		Set("aimodel_status", status).
		Set("aimodel_quarantined_reason", reason).
		Set("aimodel_updated", time.Now().UnixMilli()).
		Where("aimodel_id = ?", id)

	sql, args, err := stmt.ToSql()
	if err != nil {
		return errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}
	db := dbtx.GetAccessor(ctx, s.db)
	res, err := db.ExecContext(ctx, sql, args...)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to update status of ai model %s", id)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to get rows affected")
	}
	if n == 0 {
		return gitnessstore.ErrResourceNotFound
	}
	return nil
}

func (s *aiModelStore) UpdateLatency(
	ctx context.Context,
	id string,
	p95LatencyMS int,
) error {
	stmt := database.Builder.
		Update(aiModelTable).
		Set("aimodel_p95_latency_ms", p95LatencyMS).
		Set("aimodel_updated", time.Now().UnixMilli()).
		Where("aimodel_id = ?", id)

	sql, args, err := stmt.ToSql()
	if err != nil {
		return errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}
	db := dbtx.GetAccessor(ctx, s.db)
	res, err := db.ExecContext(ctx, sql, args...)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to update latency of ai model %s", id)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to get rows affected")
	}
	if n == 0 {
		return gitnessstore.ErrResourceNotFound
	}
	return nil
}

func mapAIModel(m *aiModel) (*types.AIModel, error) {
	rates := map[string]float64{}
	if m.SuccessRateByTask != "" {
		if err := json.Unmarshal([]byte(m.SuccessRateByTask), &rates); err != nil {
			return nil, fmt.Errorf("failed to unmarshal success rates of model %s: %w", m.ID, err)
		}
	}
	return &types.AIModel{
		ID:                m.ID,
		Kind:              m.Kind,
		ContextLength:     m.ContextLength,
		SupportsTools:     m.SupportsTools,
		Status:            m.Status,
		SuccessRateByTask: rates,
		P95LatencyMS:      m.P95LatencyMS,
		LastSeen:          m.LastSeen,
		QuarantinedReason: m.QuarantinedReason,
		Created:           m.Created,
		Updated:           m.Updated,
	}, nil
}

// ---------------------------------------------------------------------------
// decisions

type aiDecision struct {
	ID         int64       `db:"aidecision_id"`
	TaskID     null.Int    `db:"aidecision_task_id"`
	QuestionID string      `db:"aidecision_question_id"`
	ModelBuild string      `db:"aidecision_model_build"`
	StateHash  string      `db:"aidecision_state_hash"`
	Answer     string      `db:"aidecision_answer"`
	Fallback   bool        `db:"aidecision_fallback"`
	Outcome    null.String `db:"aidecision_outcome"`
	Created    int64       `db:"aidecision_created"`
}

type aiDecisionStore struct {
	db *sqlx.DB
}

// NewAIDecisionStore returns a new AIDecisionStore.
func NewAIDecisionStore(db *sqlx.DB) store.AIDecisionStore {
	return &aiDecisionStore{db: db}
}

var _ store.AIDecisionStore = (*aiDecisionStore)(nil)

func (s *aiDecisionStore) Create(ctx context.Context, d *types.AIDecision) error {
	answer, err := json.Marshal(d.Answer)
	if err != nil {
		return fmt.Errorf("failed to marshal decision answer: %w", err)
	}
	if d.Created == 0 {
		d.Created = time.Now().UnixMilli()
	}
	var outcome null.String
	if d.Outcome != nil {
		outcome = null.StringFrom(string(*d.Outcome))
	}

	stmt := database.Builder.
		Insert(aiDecisionTable).
		Columns(aiDecisionColumns).
		Values(
			null.IntFromPtr(d.TaskID), d.QuestionID, d.ModelBuild, d.StateHash,
			string(answer), d.Fallback, outcome, d.Created,
		).
		Suffix("RETURNING aidecision_id")

	sql, args, err := stmt.ToSql()
	if err != nil {
		return errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}
	db := dbtx.GetAccessor(ctx, s.db)
	if err := db.QueryRowContext(ctx, sql, args...).Scan(&d.ID); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to create ai decision for %s", d.QuestionID)
	}
	return nil
}

func (s *aiDecisionStore) Find(ctx context.Context, id int64) (*types.AIDecision, error) {
	stmt := database.Builder.
		Select(aiDecisionSelectColumns).
		From(aiDecisionTable).
		Where("aidecision_id = ?", id)

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}
	dst := &aiDecision{}
	db := dbtx.GetAccessor(ctx, s.db)
	if err := db.GetContext(ctx, dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find ai decision %d", id)
	}
	return mapAIDecision(dst)
}

func (s *aiDecisionStore) ListByTask(ctx context.Context, taskID int64) ([]*types.AIDecision, error) {
	stmt := database.Builder.
		Select(aiDecisionSelectColumns).
		From(aiDecisionTable).
		Where("aidecision_task_id = ?", taskID).
		OrderBy("aidecision_created ASC", "aidecision_id ASC")

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}
	var dst []*aiDecision
	db := dbtx.GetAccessor(ctx, s.db)
	if err := db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to list ai decisions of task %d", taskID)
	}
	out := make([]*types.AIDecision, 0, len(dst))
	for _, d := range dst {
		m, err := mapAIDecision(d)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, nil
}

func (s *aiDecisionStore) SetOutcome(ctx context.Context, id int64, outcome enum.AIDecisionOutcome) error {
	stmt := database.Builder.
		Update(aiDecisionTable).
		Set("aidecision_outcome", string(outcome)).
		Where("aidecision_id = ?", id)

	sql, args, err := stmt.ToSql()
	if err != nil {
		return errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}
	db := dbtx.GetAccessor(ctx, s.db)
	res, err := db.ExecContext(ctx, sql, args...)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to set outcome of ai decision %d", id)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to get rows affected")
	}
	if n == 0 {
		return gitnessstore.ErrResourceNotFound
	}
	return nil
}

func mapAIDecision(d *aiDecision) (*types.AIDecision, error) {
	out := &types.AIDecision{
		ID:         d.ID,
		TaskID:     d.TaskID.Ptr(),
		QuestionID: d.QuestionID,
		ModelBuild: d.ModelBuild,
		StateHash:  d.StateHash,
		Fallback:   d.Fallback,
		Created:    d.Created,
	}
	if d.Answer != "" {
		if err := json.Unmarshal([]byte(d.Answer), &out.Answer); err != nil {
			return nil, fmt.Errorf("failed to unmarshal answer of decision %d: %w", d.ID, err)
		}
	}
	if d.Outcome.Valid {
		o := enum.AIDecisionOutcome(d.Outcome.String)
		out.Outcome = &o
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// quota

type aiQuota struct {
	Day      string `db:"aiquota_day"`
	Class    string `db:"aiquota_class"`
	Used     int    `db:"aiquota_used"`
	Reserved int    `db:"aiquota_reserved"`
}

type aiQuotaStore struct {
	db *sqlx.DB
}

// NewAIQuotaStore returns a new AIQuotaStore.
func NewAIQuotaStore(db *sqlx.DB) store.AIQuotaStore {
	return &aiQuotaStore{db: db}
}

var _ store.AIQuotaStore = (*aiQuotaStore)(nil)

func (s *aiQuotaStore) Find(ctx context.Context, day, class string) (*types.AIQuota, error) {
	stmt := database.Builder.
		Select("aiquota_day, aiquota_class, aiquota_used, aiquota_reserved").
		From(aiQuotaTable).
		Where("aiquota_day = ?", day).
		Where("aiquota_class = ?", class)

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}
	dst := &aiQuota{}
	db := dbtx.GetAccessor(ctx, s.db)
	if err := db.GetContext(ctx, dst, sql, args...); err != nil {
		err = database.ProcessSQLErrorf(ctx, err, "Failed to find ai quota %s/%s", day, class)
		if errors.Is(err, gitnessstore.ErrResourceNotFound) {
			return &types.AIQuota{Day: day, Class: class}, nil
		}
		return nil, err
	}
	return &types.AIQuota{Day: dst.Day, Class: dst.Class, Used: dst.Used, Reserved: dst.Reserved}, nil
}

// IncrementQuota is a single upsert statement, so the check and the increment are atomic
// on both postgres and sqlite. The conflict WHERE clause leaves the row untouched (zero
// rows affected) once used has reached the limit.
func (s *aiQuotaStore) IncrementQuota(ctx context.Context, day, class string, limit int) (bool, error) {
	if limit <= 0 {
		return false, nil
	}
	stmt := database.Builder.
		Insert(aiQuotaTable).
		Columns("aiquota_day", "aiquota_class", "aiquota_used", "aiquota_reserved").
		Values(day, class, 1, 0).
		Suffix(`ON CONFLICT (aiquota_day, aiquota_class) DO UPDATE SET
			aiquota_used = ai_quota.aiquota_used + 1
			WHERE ai_quota.aiquota_used < ?`, limit)

	sql, args, err := stmt.ToSql()
	if err != nil {
		return false, errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}
	db := dbtx.GetAccessor(ctx, s.db)
	res, err := db.ExecContext(ctx, sql, args...)
	if err != nil {
		return false, database.ProcessSQLErrorf(ctx, err, "Failed to increment ai quota %s/%s", day, class)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, database.ProcessSQLErrorf(ctx, err, "Failed to get rows affected")
	}
	return n > 0, nil
}

// DecrementQuota gives back one unit of used; it never goes below zero and is a no-op for a missing row.
func (s *aiQuotaStore) DecrementQuota(ctx context.Context, day, class string) error {
	stmt := database.Builder.
		Update(aiQuotaTable).
		Set("aiquota_used", squirrel.Expr("aiquota_used - 1")).
		Where("aiquota_day = ?", day).
		Where("aiquota_class = ?", class).
		Where("aiquota_used > 0")

	sql, args, err := stmt.ToSql()
	if err != nil {
		return errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}
	db := dbtx.GetAccessor(ctx, s.db)
	if _, err := db.ExecContext(ctx, sql, args...); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to decrement ai quota %s/%s", day, class)
	}
	return nil
}

// RaiseQuota is a single upsert: used becomes max(used, value).
func (s *aiQuotaStore) RaiseQuota(ctx context.Context, day, class string, used int) error {
	stmt := database.Builder.
		Insert(aiQuotaTable).
		Columns("aiquota_day", "aiquota_class", "aiquota_used", "aiquota_reserved").
		Values(day, class, used, 0).
		Suffix(`ON CONFLICT (aiquota_day, aiquota_class) DO UPDATE SET
			aiquota_used = ?
			WHERE ai_quota.aiquota_used < ?`, used, used)

	sql, args, err := stmt.ToSql()
	if err != nil {
		return errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}
	db := dbtx.GetAccessor(ctx, s.db)
	if _, err := db.ExecContext(ctx, sql, args...); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to raise ai quota %s/%s", day, class)
	}
	return nil
}

// ---------------------------------------------------------------------------
// usage

type aiUsage struct {
	ID        int64    `db:"aiusage_id"`
	SpaceID   null.Int `db:"aiusage_space_id"`
	RepoID    null.Int `db:"aiusage_repo_id"`
	Feature   string   `db:"aiusage_feature"`
	Model     string   `db:"aiusage_model"`
	Requests  int      `db:"aiusage_requests"`
	TokensIn  int64    `db:"aiusage_tokens_in"`
	TokensOut int64    `db:"aiusage_tokens_out"`
	CostUSD   float64  `db:"aiusage_cost_usd"`
	Created   int64    `db:"aiusage_created"`
}

type aiUsageStore struct {
	db *sqlx.DB
}

// NewAIUsageStore returns a new AIUsageStore.
func NewAIUsageStore(db *sqlx.DB) store.AIUsageStore {
	return &aiUsageStore{db: db}
}

var _ store.AIUsageStore = (*aiUsageStore)(nil)

func (s *aiUsageStore) Create(ctx context.Context, u *types.AIUsage) error {
	if u.Created == 0 {
		u.Created = time.Now().UnixMilli()
	}
	stmt := database.Builder.
		Insert(aiUsageTable).
		Columns(aiUsageColumns).
		Values(
			null.IntFromPtr(u.SpaceID), null.IntFromPtr(u.RepoID), u.Feature, u.Model,
			u.Requests, u.TokensIn, u.TokensOut, u.CostUSD, u.Created,
		).
		Suffix("RETURNING aiusage_id")

	sql, args, err := stmt.ToSql()
	if err != nil {
		return errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}
	db := dbtx.GetAccessor(ctx, s.db)
	if err := db.QueryRowContext(ctx, sql, args...).Scan(&u.ID); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to create ai usage for %s", u.Feature)
	}
	return nil
}

func (s *aiUsageStore) ListSince(ctx context.Context, since int64) ([]*types.AIUsage, error) {
	stmt := database.Builder.
		Select(aiUsageSelectColumns).
		From(aiUsageTable).
		Where("aiusage_created >= ?", since).
		OrderBy("aiusage_created ASC", "aiusage_id ASC")

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}
	var dst []*aiUsage
	db := dbtx.GetAccessor(ctx, s.db)
	if err := db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to list ai usage")
	}
	out := make([]*types.AIUsage, 0, len(dst))
	for _, d := range dst {
		out = append(out, &types.AIUsage{
			ID:        d.ID,
			SpaceID:   d.SpaceID.Ptr(),
			RepoID:    d.RepoID.Ptr(),
			Feature:   d.Feature,
			Model:     d.Model,
			Requests:  d.Requests,
			TokensIn:  d.TokensIn,
			TokensOut: d.TokensOut,
			CostUSD:   d.CostUSD,
			Created:   d.Created,
		})
	}
	return out, nil
}

func (s *aiUsageStore) SumRequestsSince(
	ctx context.Context,
	spaceID int64,
	repoIDs []int64,
	since int64,
) (int, error) {
	scope := squirrel.Or{squirrel.Eq{"aiusage_space_id": spaceID}}
	if len(repoIDs) > 0 {
		scope = append(scope, squirrel.Eq{"aiusage_repo_id": repoIDs})
	}
	stmt := database.Builder.
		Select("COALESCE(SUM(aiusage_requests), 0)").
		From(aiUsageTable).
		Where("aiusage_created >= ?", since).
		Where(scope)

	sql, args, err := stmt.ToSql()
	if err != nil {
		return 0, errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}
	var sum int
	db := dbtx.GetAccessor(ctx, s.db)
	if err := db.QueryRowContext(ctx, sql, args...).Scan(&sum); err != nil {
		return 0, database.ProcessSQLErrorf(ctx, err, "Failed to sum ai usage requests")
	}
	return sum, nil
}
