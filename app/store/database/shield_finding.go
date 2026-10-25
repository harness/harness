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

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/Masterminds/squirrel"
	"github.com/guregu/null"
	"github.com/jmoiron/sqlx"
)

var _ store.ShieldFindingStore = (*ShieldFindingStore)(nil)

// NewShieldFindingStore returns a new ShieldFindingStore.
func NewShieldFindingStore(db *sqlx.DB) *ShieldFindingStore {
	return &ShieldFindingStore{db: db}
}

// ShieldFindingStore implements store.ShieldFindingStore backed by a relational database.
type ShieldFindingStore struct {
	db *sqlx.DB
}

type shieldFinding struct {
	ID               int64                   `db:"shield_finding_id"`
	RepoID           int64                   `db:"shield_finding_repo_id"`
	Fingerprint      string                  `db:"shield_finding_fingerprint"`
	Scanner          enum.ShieldScanner      `db:"shield_finding_scanner"`
	Kind             enum.ShieldFindingKind  `db:"shield_finding_kind"`
	Ecosystem        string                  `db:"shield_finding_ecosystem"`
	Package          string                  `db:"shield_finding_package"`
	InstalledVersion string                  `db:"shield_finding_installed_version"`
	ManifestPath     string                  `db:"shield_finding_manifest_path"`
	VulnID           string                  `db:"shield_finding_vuln_id"`
	Aliases          string                  `db:"shield_finding_aliases"`
	Summary          string                  `db:"shield_finding_summary"`
	CVSSVector       string                  `db:"shield_finding_cvss_vector"`
	CVSSScore        float64                 `db:"shield_finding_cvss_score"`
	EPSS             float64                 `db:"shield_finding_epss"`
	InKEV            bool                    `db:"shield_finding_in_kev"`
	Reachable        enum.ShieldReachability `db:"shield_finding_reachable"`
	RiskScore        int                     `db:"shield_finding_risk_score"`
	FixStrategy      null.String             `db:"shield_finding_fix_strategy"`
	MinFixedVersion  string                  `db:"shield_finding_min_fixed_version"`
	FixedVersions    string                  `db:"shield_finding_fixed_versions"`
	State            enum.ShieldFindingState `db:"shield_finding_state"`
	StateReason      string                  `db:"shield_finding_state_reason"`
	FirstSeenScanID  int64                   `db:"shield_finding_first_seen_scan_id"`
	LastSeenScanID   int64                   `db:"shield_finding_last_seen_scan_id"`
	FirstSeen        int64                   `db:"shield_finding_first_seen"`
	LastSeen         int64                   `db:"shield_finding_last_seen"`
	FixedAt          null.Int                `db:"shield_finding_fixed_at"`
	TaskID           null.Int                `db:"shield_finding_task_id"`
	AIDecisionIDs    string                  `db:"shield_finding_ai_decision_ids"`
	Created          int64                   `db:"shield_finding_created"`
	Updated          int64                   `db:"shield_finding_updated"`
	Version          int64                   `db:"shield_finding_version"`
}

const (
	shieldFindingColumns = `
		 shield_finding_id
		,shield_finding_repo_id
		,shield_finding_fingerprint
		,shield_finding_scanner
		,shield_finding_kind
		,shield_finding_ecosystem
		,shield_finding_package
		,shield_finding_installed_version
		,shield_finding_manifest_path
		,shield_finding_vuln_id
		,shield_finding_aliases
		,shield_finding_summary
		,shield_finding_cvss_vector
		,shield_finding_cvss_score
		,shield_finding_epss
		,shield_finding_in_kev
		,shield_finding_reachable
		,shield_finding_risk_score
		,shield_finding_fix_strategy
		,shield_finding_min_fixed_version
		,shield_finding_fixed_versions
		,shield_finding_state
		,shield_finding_state_reason
		,shield_finding_first_seen_scan_id
		,shield_finding_last_seen_scan_id
		,shield_finding_first_seen
		,shield_finding_last_seen
		,shield_finding_fixed_at
		,shield_finding_task_id
		,shield_finding_ai_decision_ids
		,shield_finding_created
		,shield_finding_updated
		,shield_finding_version`

	shieldFindingSelectBase = `SELECT` + shieldFindingColumns + `
		FROM shield_findings`
)

func (s *ShieldFindingStore) Upsert(ctx context.Context, f *types.ShieldFinding) error {
	const sqlQuery = `
		INSERT INTO shield_findings (
			 shield_finding_repo_id
			,shield_finding_fingerprint
			,shield_finding_scanner
			,shield_finding_kind
			,shield_finding_ecosystem
			,shield_finding_package
			,shield_finding_installed_version
			,shield_finding_manifest_path
			,shield_finding_vuln_id
			,shield_finding_aliases
			,shield_finding_summary
			,shield_finding_cvss_vector
			,shield_finding_cvss_score
			,shield_finding_epss
			,shield_finding_in_kev
			,shield_finding_reachable
			,shield_finding_risk_score
			,shield_finding_fix_strategy
			,shield_finding_min_fixed_version
			,shield_finding_fixed_versions
			,shield_finding_state
			,shield_finding_state_reason
			,shield_finding_first_seen_scan_id
			,shield_finding_last_seen_scan_id
			,shield_finding_first_seen
			,shield_finding_last_seen
			,shield_finding_fixed_at
			,shield_finding_task_id
			,shield_finding_ai_decision_ids
			,shield_finding_created
			,shield_finding_updated
			,shield_finding_version
		) VALUES (
			 :shield_finding_repo_id
			,:shield_finding_fingerprint
			,:shield_finding_scanner
			,:shield_finding_kind
			,:shield_finding_ecosystem
			,:shield_finding_package
			,:shield_finding_installed_version
			,:shield_finding_manifest_path
			,:shield_finding_vuln_id
			,:shield_finding_aliases
			,:shield_finding_summary
			,:shield_finding_cvss_vector
			,:shield_finding_cvss_score
			,:shield_finding_epss
			,:shield_finding_in_kev
			,:shield_finding_reachable
			,:shield_finding_risk_score
			,:shield_finding_fix_strategy
			,:shield_finding_min_fixed_version
			,:shield_finding_fixed_versions
			,:shield_finding_state
			,:shield_finding_state_reason
			,:shield_finding_first_seen_scan_id
			,:shield_finding_last_seen_scan_id
			,:shield_finding_first_seen
			,:shield_finding_last_seen
			,:shield_finding_fixed_at
			,:shield_finding_task_id
			,:shield_finding_ai_decision_ids
			,:shield_finding_created
			,:shield_finding_updated
			,0
		)
		ON CONFLICT (shield_finding_repo_id, shield_finding_fingerprint) DO UPDATE SET
			 shield_finding_scanner = excluded.shield_finding_scanner
			,shield_finding_kind = excluded.shield_finding_kind
			,shield_finding_ecosystem = excluded.shield_finding_ecosystem
			,shield_finding_package = excluded.shield_finding_package
			,shield_finding_installed_version = excluded.shield_finding_installed_version
			,shield_finding_manifest_path = excluded.shield_finding_manifest_path
			,shield_finding_vuln_id = excluded.shield_finding_vuln_id
			,shield_finding_aliases = excluded.shield_finding_aliases
			,shield_finding_summary = excluded.shield_finding_summary
			,shield_finding_cvss_vector = excluded.shield_finding_cvss_vector
			,shield_finding_cvss_score = excluded.shield_finding_cvss_score
			,shield_finding_epss = excluded.shield_finding_epss
			,shield_finding_in_kev = excluded.shield_finding_in_kev
			,shield_finding_reachable = CASE
				WHEN shield_findings.shield_finding_ai_decision_ids <> '[]'
				THEN shield_findings.shield_finding_reachable
				ELSE excluded.shield_finding_reachable END
			,shield_finding_risk_score = excluded.shield_finding_risk_score
			,shield_finding_fix_strategy = CASE
				WHEN shield_findings.shield_finding_ai_decision_ids <> '[]'
				THEN shield_findings.shield_finding_fix_strategy
				ELSE excluded.shield_finding_fix_strategy END
			,shield_finding_min_fixed_version = excluded.shield_finding_min_fixed_version
			,shield_finding_fixed_versions = excluded.shield_finding_fixed_versions
			,shield_finding_last_seen_scan_id = excluded.shield_finding_last_seen_scan_id
			,shield_finding_last_seen = excluded.shield_finding_last_seen
			,shield_finding_state = CASE
				WHEN shield_findings.shield_finding_state = 'fixed' THEN 'open'
				ELSE shield_findings.shield_finding_state END
			,shield_finding_state_reason = CASE
				WHEN shield_findings.shield_finding_state = 'fixed' THEN 'reappeared'
				ELSE shield_findings.shield_finding_state_reason END
			,shield_finding_fixed_at = CASE
				WHEN shield_findings.shield_finding_state = 'fixed' THEN NULL
				ELSE shield_findings.shield_finding_fixed_at END
			,shield_finding_updated = excluded.shield_finding_updated
			,shield_finding_version = shield_findings.shield_finding_version + 1
		RETURNING ` + shieldFindingColumns

	db := dbtx.GetAccessor(ctx, s.db)

	dbFinding, err := mapInternalShieldFinding(f)
	if err != nil {
		return err
	}

	query, arg, err := db.BindNamed(sqlQuery, dbFinding)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind shield finding object")
	}

	dst := &shieldFinding{}
	if err = db.GetContext(ctx, dst, query, arg...); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to upsert shield finding")
	}

	stored, err := mapShieldFinding(dst)
	if err != nil {
		return err
	}
	*f = *stored

	return nil
}

func (s *ShieldFindingStore) Find(ctx context.Context, id int64) (*types.ShieldFinding, error) {
	const sqlQuery = shieldFindingSelectBase + ` WHERE shield_finding_id = $1`

	db := dbtx.GetAccessor(ctx, s.db)

	dst := &shieldFinding{}
	if err := db.GetContext(ctx, dst, sqlQuery, id); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find shield finding")
	}

	return mapShieldFinding(dst)
}

func (s *ShieldFindingStore) List(
	ctx context.Context,
	repoID int64,
	filter *types.ShieldFindingFilter,
) ([]*types.ShieldFinding, error) {
	stmt := database.Builder.
		Select(shieldFindingColumns).
		From("shield_findings").
		Where("shield_finding_repo_id = ?", repoID)

	stmt = applyShieldFindingFilter(stmt, filter)

	page, size := 0, 0
	if filter != nil {
		page, size = filter.Page, filter.Size
	}

	stmt = stmt.
		OrderBy("shield_finding_risk_score DESC", "shield_finding_id ASC").
		Limit(database.Limit(size)).
		Offset(database.Offset(page, size))

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to convert shield finding list query to sql: %w", err)
	}

	db := dbtx.GetAccessor(ctx, s.db)

	dst := []*shieldFinding{}
	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed executing shield finding list query")
	}

	out := make([]*types.ShieldFinding, len(dst))
	for i, d := range dst {
		if out[i], err = mapShieldFinding(d); err != nil {
			return nil, err
		}
	}

	return out, nil
}

func (s *ShieldFindingStore) Count(
	ctx context.Context,
	repoID int64,
	filter *types.ShieldFindingFilter,
) (int64, error) {
	stmt := database.Builder.
		Select("COUNT(*)").
		From("shield_findings").
		Where("shield_finding_repo_id = ?", repoID)

	stmt = applyShieldFindingFilter(stmt, filter)

	sql, args, err := stmt.ToSql()
	if err != nil {
		return 0, fmt.Errorf("failed to convert shield finding count query to sql: %w", err)
	}

	db := dbtx.GetAccessor(ctx, s.db)

	var count int64
	if err = db.QueryRowContext(ctx, sql, args...).Scan(&count); err != nil {
		return 0, database.ProcessSQLErrorf(ctx, err, "Failed executing shield finding count query")
	}

	return count, nil
}

func (s *ShieldFindingStore) MarkFixedNotSeenIn(ctx context.Context, repoID, scanID, now int64) (int64, error) {
	const sqlQuery = `
		UPDATE shield_findings
		SET
			 shield_finding_state = 'fixed'
			,shield_finding_state_reason = 'not_seen_in_scan'
			,shield_finding_fixed_at = $1
			,shield_finding_updated = $1
			,shield_finding_version = shield_finding_version + 1
		WHERE shield_finding_repo_id = $2
			AND shield_finding_state = 'open'
			AND shield_finding_last_seen_scan_id <> $3`

	db := dbtx.GetAccessor(ctx, s.db)

	result, err := db.ExecContext(ctx, sqlQuery, now, repoID, scanID)
	if err != nil {
		return 0, database.ProcessSQLErrorf(ctx, err, "Failed to mark shield findings as fixed")
	}

	count, err := result.RowsAffected()
	if err != nil {
		return 0, database.ProcessSQLErrorf(ctx, err, "Failed to get number of fixed shield findings")
	}

	return count, nil
}

func (s *ShieldFindingStore) SetTask(ctx context.Context, id int64, taskID *int64, now int64) error {
	const sqlQuery = `
		UPDATE shield_findings
		SET
			 shield_finding_task_id = $1
			,shield_finding_updated = $2
			,shield_finding_version = shield_finding_version + 1
		WHERE shield_finding_id = $3`

	db := dbtx.GetAccessor(ctx, s.db)

	if _, err := db.ExecContext(ctx, sqlQuery, null.IntFromPtr(taskID), now, id); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to set shield finding task")
	}

	return nil
}

func (s *ShieldFindingStore) SetFixStrategy(
	ctx context.Context,
	id int64,
	strategy enum.ShieldFixStrategy,
	now int64,
) error {
	const sqlQuery = `
		UPDATE shield_findings
		SET
			 shield_finding_fix_strategy = $1
			,shield_finding_updated = $2
			,shield_finding_version = shield_finding_version + 1
		WHERE shield_finding_id = $3`

	db := dbtx.GetAccessor(ctx, s.db)

	if _, err := db.ExecContext(ctx, sqlQuery, string(strategy), now, id); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to set shield finding fix strategy")
	}

	return nil
}

func (s *ShieldFindingStore) UpdateState(
	ctx context.Context,
	id int64,
	state enum.ShieldFindingState,
	reason string,
	now int64,
) error {
	const sqlQuery = `
		UPDATE shield_findings
		SET
			 shield_finding_state = $1
			,shield_finding_state_reason = $2
			,shield_finding_fixed_at = $3
			,shield_finding_updated = $4
			,shield_finding_version = shield_finding_version + 1
		WHERE shield_finding_id = $5`

	db := dbtx.GetAccessor(ctx, s.db)

	var fixedAt null.Int
	if state == enum.ShieldFindingStateFixed {
		fixedAt = null.IntFrom(now)
	}

	if _, err := db.ExecContext(ctx, sqlQuery, state, reason, fixedAt, now, id); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to update shield finding state")
	}

	return nil
}

func (s *ShieldFindingStore) UpdateTriage(
	ctx context.Context,
	id int64,
	in *types.ShieldFindingTriage,
	now int64,
) error {
	ids, err := marshalJSONList(in.AIDecisionIDs)
	if err != nil {
		return err
	}
	const sqlQuery = `
		UPDATE shield_findings
		SET
			 shield_finding_risk_score = $1
			,shield_finding_reachable = $2
			,shield_finding_fix_strategy = COALESCE($3, shield_finding_fix_strategy)
			,shield_finding_ai_decision_ids = $4
			,shield_finding_updated = $5
			,shield_finding_version = shield_finding_version + 1
		WHERE shield_finding_id = $6`

	db := dbtx.GetAccessor(ctx, s.db)

	var strategy null.String
	if in.FixStrategy != nil {
		strategy = null.StringFrom(string(*in.FixStrategy))
	}
	if _, err = db.ExecContext(ctx, sqlQuery, in.RiskScore, in.Reachable, strategy, ids, now, id); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to update shield finding triage")
	}

	return nil
}

func applyShieldFindingFilter(
	stmt squirrel.SelectBuilder,
	filter *types.ShieldFindingFilter,
) squirrel.SelectBuilder {
	if filter == nil {
		return stmt
	}

	if len(filter.States) > 0 {
		stmt = stmt.Where(squirrel.Eq{"shield_finding_state": filter.States})
	}
	if len(filter.Kinds) > 0 {
		stmt = stmt.Where(squirrel.Eq{"shield_finding_kind": filter.Kinds})
	}
	if filter.MinRiskScore > 0 {
		stmt = stmt.Where("shield_finding_risk_score >= ?", filter.MinRiskScore)
	}

	return stmt
}

func marshalJSONList[T any](in []T) (string, error) {
	if in == nil {
		return "[]", nil
	}
	b, err := json.Marshal(in)
	if err != nil {
		return "", fmt.Errorf("failed to marshal list: %w", err)
	}
	return string(b), nil
}

func unmarshalJSONList[T any](in string) ([]T, error) {
	out := []T{}
	if in == "" {
		return out, nil
	}
	if err := json.Unmarshal([]byte(in), &out); err != nil {
		return nil, fmt.Errorf("failed to unmarshal list: %w", err)
	}
	return out, nil
}

func mapInternalShieldFinding(f *types.ShieldFinding) (*shieldFinding, error) {
	aliases, err := marshalJSONList(f.Aliases)
	if err != nil {
		return nil, err
	}
	fixedVersions, err := marshalJSONList(f.FixedVersions)
	if err != nil {
		return nil, err
	}
	decisions, err := marshalJSONList(f.AIDecisionIDs)
	if err != nil {
		return nil, err
	}

	var strategy null.String
	if f.FixStrategy != nil {
		strategy = null.StringFrom(string(*f.FixStrategy))
	}

	return &shieldFinding{
		ID:               f.ID,
		RepoID:           f.RepoID,
		Fingerprint:      f.Fingerprint,
		Scanner:          f.Scanner,
		Kind:             f.Kind,
		Ecosystem:        f.Ecosystem,
		Package:          f.Package,
		InstalledVersion: f.InstalledVersion,
		ManifestPath:     f.ManifestPath,
		VulnID:           f.VulnID,
		Aliases:          aliases,
		Summary:          f.Summary,
		CVSSVector:       f.CVSSVector,
		CVSSScore:        f.CVSSScore,
		EPSS:             f.EPSS,
		InKEV:            f.InKEV,
		Reachable:        f.Reachable,
		RiskScore:        f.RiskScore,
		FixStrategy:      strategy,
		MinFixedVersion:  f.MinFixedVersion,
		FixedVersions:    fixedVersions,
		State:            f.State,
		StateReason:      f.StateReason,
		FirstSeenScanID:  f.FirstSeenScanID,
		LastSeenScanID:   f.LastSeenScanID,
		FirstSeen:        f.FirstSeen,
		LastSeen:         f.LastSeen,
		FixedAt:          null.IntFromPtr(f.FixedAt),
		TaskID:           null.IntFromPtr(f.TaskID),
		AIDecisionIDs:    decisions,
		Created:          f.Created,
		Updated:          f.Updated,
		Version:          f.Version,
	}, nil
}

func mapShieldFinding(f *shieldFinding) (*types.ShieldFinding, error) {
	aliases, err := unmarshalJSONList[string](f.Aliases)
	if err != nil {
		return nil, err
	}
	fixedVersions, err := unmarshalJSONList[string](f.FixedVersions)
	if err != nil {
		return nil, err
	}
	decisions, err := unmarshalJSONList[int64](f.AIDecisionIDs)
	if err != nil {
		return nil, err
	}

	var strategy *enum.ShieldFixStrategy
	if f.FixStrategy.Valid {
		v := enum.ShieldFixStrategy(f.FixStrategy.String)
		strategy = &v
	}

	return &types.ShieldFinding{
		ID:               f.ID,
		Version:          f.Version,
		RepoID:           f.RepoID,
		Fingerprint:      f.Fingerprint,
		Scanner:          f.Scanner,
		Kind:             f.Kind,
		Ecosystem:        f.Ecosystem,
		Package:          f.Package,
		InstalledVersion: f.InstalledVersion,
		ManifestPath:     f.ManifestPath,
		VulnID:           f.VulnID,
		Aliases:          aliases,
		Summary:          f.Summary,
		CVSSVector:       f.CVSSVector,
		CVSSScore:        f.CVSSScore,
		EPSS:             f.EPSS,
		InKEV:            f.InKEV,
		Reachable:        f.Reachable,
		RiskScore:        f.RiskScore,
		FixStrategy:      strategy,
		MinFixedVersion:  f.MinFixedVersion,
		FixedVersions:    fixedVersions,
		State:            f.State,
		StateReason:      f.StateReason,
		FirstSeenScanID:  f.FirstSeenScanID,
		LastSeenScanID:   f.LastSeenScanID,
		FirstSeen:        f.FirstSeen,
		LastSeen:         f.LastSeen,
		FixedAt:          f.FixedAt.Ptr(),
		TaskID:           f.TaskID.Ptr(),
		AIDecisionIDs:    decisions,
		Created:          f.Created,
		Updated:          f.Updated,
	}, nil
}
