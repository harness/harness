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
	"fmt"
	"strings"

	"github.com/harness/gitness/app/store"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/Masterminds/squirrel"
	"github.com/guregu/null"
	"github.com/jmoiron/sqlx"
)

var (
	_ store.InsightsProjectStore = (*InsightsProjectStore)(nil)
	_ store.InsightsIssueStore   = (*InsightsIssueStore)(nil)
	_ store.InsightsEventStore   = (*InsightsEventStore)(nil)
)

// ---------------------------------------------------------------------------
// projects
// ---------------------------------------------------------------------------

// NewInsightsProjectStore returns a new InsightsProjectStore.
func NewInsightsProjectStore(db *sqlx.DB) *InsightsProjectStore {
	return &InsightsProjectStore{db: db}
}

// InsightsProjectStore implements store.InsightsProjectStore backed by a relational database.
type InsightsProjectStore struct {
	db *sqlx.DB
}

type insightsProject struct {
	ID                 int64   `db:"insights_project_id"`
	RepoID             int64   `db:"insights_project_repo_id"`
	Identifier         string  `db:"insights_project_identifier"`
	PublicKey          string  `db:"insights_project_public_key"`
	Enabled            bool    `db:"insights_project_enabled"`
	RateLimitPerMinute int     `db:"insights_project_rate_limit_per_minute"`
	SampleRate         float64 `db:"insights_project_sample_rate"`
	CreatedBy          int64   `db:"insights_project_created_by"`
	Created            int64   `db:"insights_project_created"`
	Updated            int64   `db:"insights_project_updated"`
}

const (
	insightsProjectColumns = `
		 insights_project_id
		,insights_project_repo_id
		,insights_project_identifier
		,insights_project_public_key
		,insights_project_enabled
		,insights_project_rate_limit_per_minute
		,insights_project_sample_rate
		,insights_project_created_by
		,insights_project_created
		,insights_project_updated`

	insightsProjectSelectBase = `SELECT` + insightsProjectColumns + `
		FROM insights_projects`
)

func (s *InsightsProjectStore) Create(ctx context.Context, p *types.InsightsProject) error {
	const sqlQuery = `
		INSERT INTO insights_projects (
			 insights_project_repo_id
			,insights_project_identifier
			,insights_project_public_key
			,insights_project_enabled
			,insights_project_rate_limit_per_minute
			,insights_project_sample_rate
			,insights_project_created_by
			,insights_project_created
			,insights_project_updated
		) VALUES (
			 :insights_project_repo_id
			,:insights_project_identifier
			,:insights_project_public_key
			,:insights_project_enabled
			,:insights_project_rate_limit_per_minute
			,:insights_project_sample_rate
			,:insights_project_created_by
			,:insights_project_created
			,:insights_project_updated
		) RETURNING insights_project_id`

	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(sqlQuery, mapInternalInsightsProject(p))
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind insights project object")
	}

	if err = db.QueryRowContext(ctx, query, arg...).Scan(&p.ID); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to insert insights project")
	}

	return nil
}

func (s *InsightsProjectStore) Find(ctx context.Context, id int64) (*types.InsightsProject, error) {
	const sqlQuery = insightsProjectSelectBase + ` WHERE insights_project_id = $1`

	db := dbtx.GetAccessor(ctx, s.db)

	dst := &insightsProject{}
	if err := db.GetContext(ctx, dst, sqlQuery, id); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find insights project")
	}

	return mapInsightsProject(dst), nil
}

func (s *InsightsProjectStore) FindByIdentifier(
	ctx context.Context,
	repoID int64,
	identifier string,
) (*types.InsightsProject, error) {
	const sqlQuery = insightsProjectSelectBase +
		` WHERE insights_project_repo_id = $1 AND insights_project_identifier = $2`

	db := dbtx.GetAccessor(ctx, s.db)

	dst := &insightsProject{}
	if err := db.GetContext(ctx, dst, sqlQuery, repoID, identifier); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find insights project by identifier")
	}

	return mapInsightsProject(dst), nil
}

func (s *InsightsProjectStore) List(ctx context.Context, repoID int64) ([]*types.InsightsProject, error) {
	const sqlQuery = insightsProjectSelectBase + `
		WHERE insights_project_repo_id = $1
		ORDER BY insights_project_identifier ASC`

	db := dbtx.GetAccessor(ctx, s.db)

	dst := []*insightsProject{}
	if err := db.SelectContext(ctx, &dst, sqlQuery, repoID); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to list insights projects")
	}

	out := make([]*types.InsightsProject, len(dst))
	for i := range dst {
		out[i] = mapInsightsProject(dst[i])
	}

	return out, nil
}

func (s *InsightsProjectStore) Update(ctx context.Context, p *types.InsightsProject) error {
	const sqlQuery = `
		UPDATE insights_projects
		SET
			 insights_project_public_key = :insights_project_public_key
			,insights_project_enabled = :insights_project_enabled
			,insights_project_rate_limit_per_minute = :insights_project_rate_limit_per_minute
			,insights_project_sample_rate = :insights_project_sample_rate
			,insights_project_updated = :insights_project_updated
		WHERE insights_project_id = :insights_project_id`

	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(sqlQuery, mapInternalInsightsProject(p))
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind insights project object")
	}

	result, err := db.ExecContext(ctx, query, arg...)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to update insights project")
	}

	count, err := result.RowsAffected()
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to get number of updated insights project rows")
	}
	if count == 0 {
		return gitness_store.ErrResourceNotFound
	}

	return nil
}

func (s *InsightsProjectStore) Delete(ctx context.Context, id int64) error {
	const sqlQuery = `DELETE FROM insights_projects WHERE insights_project_id = $1`

	db := dbtx.GetAccessor(ctx, s.db)

	if _, err := db.ExecContext(ctx, sqlQuery, id); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to delete insights project")
	}

	return nil
}

func mapInsightsProject(p *insightsProject) *types.InsightsProject {
	return &types.InsightsProject{
		ID:                 p.ID,
		RepoID:             p.RepoID,
		Identifier:         p.Identifier,
		PublicKey:          p.PublicKey,
		Enabled:            p.Enabled,
		RateLimitPerMinute: p.RateLimitPerMinute,
		SampleRate:         p.SampleRate,
		CreatedBy:          p.CreatedBy,
		Created:            p.Created,
		Updated:            p.Updated,
	}
}

func mapInternalInsightsProject(p *types.InsightsProject) *insightsProject {
	return &insightsProject{
		ID:                 p.ID,
		RepoID:             p.RepoID,
		Identifier:         p.Identifier,
		PublicKey:          p.PublicKey,
		Enabled:            p.Enabled,
		RateLimitPerMinute: p.RateLimitPerMinute,
		SampleRate:         p.SampleRate,
		CreatedBy:          p.CreatedBy,
		Created:            p.Created,
		Updated:            p.Updated,
	}
}

// ---------------------------------------------------------------------------
// issues
// ---------------------------------------------------------------------------

// NewInsightsIssueStore returns a new InsightsIssueStore.
func NewInsightsIssueStore(db *sqlx.DB) *InsightsIssueStore {
	return &InsightsIssueStore{db: db}
}

// InsightsIssueStore implements store.InsightsIssueStore backed by a relational database.
type InsightsIssueStore struct {
	db *sqlx.DB
}

type insightsIssue struct {
	ID                int64                    `db:"insights_issue_id"`
	ProjectID         int64                    `db:"insights_issue_project_id"`
	RepoID            int64                    `db:"insights_issue_repo_id"`
	Fingerprint       string                   `db:"insights_issue_fingerprint"`
	Title             string                   `db:"insights_issue_title"`
	Culprit           string                   `db:"insights_issue_culprit"`
	Level             string                   `db:"insights_issue_level"`
	Platform          string                   `db:"insights_issue_platform"`
	FirstSeen         int64                    `db:"insights_issue_first_seen"`
	LastSeen          int64                    `db:"insights_issue_last_seen"`
	TimesSeen         int64                    `db:"insights_issue_times_seen"`
	UsersSeen         int64                    `db:"insights_issue_users_seen"`
	LastUser          string                   `db:"insights_issue_last_user"`
	FirstRelease      string                   `db:"insights_issue_first_release"`
	LastRelease       string                   `db:"insights_issue_last_release"`
	Status            enum.InsightsIssueStatus `db:"insights_issue_status"`
	ResolvedInRelease string                   `db:"insights_issue_resolved_in_release"`
	TaskID            null.Int                 `db:"insights_issue_task_id"`
	Created           int64                    `db:"insights_issue_created"`
	Updated           int64                    `db:"insights_issue_updated"`
	Version           int64                    `db:"insights_issue_version"`
}

const (
	insightsIssueColumns = `
		 insights_issue_id
		,insights_issue_project_id
		,insights_issue_repo_id
		,insights_issue_fingerprint
		,insights_issue_title
		,insights_issue_culprit
		,insights_issue_level
		,insights_issue_platform
		,insights_issue_first_seen
		,insights_issue_last_seen
		,insights_issue_times_seen
		,insights_issue_users_seen
		,insights_issue_last_user
		,insights_issue_first_release
		,insights_issue_last_release
		,insights_issue_status
		,insights_issue_resolved_in_release
		,insights_issue_task_id
		,insights_issue_created
		,insights_issue_updated
		,insights_issue_version`

	insightsIssueSelectBase = `SELECT` + insightsIssueColumns + `
		FROM insights_issues`
)

func (s *InsightsIssueStore) Create(ctx context.Context, i *types.InsightsIssue) error {
	const sqlQuery = `
		INSERT INTO insights_issues (
			 insights_issue_project_id
			,insights_issue_repo_id
			,insights_issue_fingerprint
			,insights_issue_title
			,insights_issue_culprit
			,insights_issue_level
			,insights_issue_platform
			,insights_issue_first_seen
			,insights_issue_last_seen
			,insights_issue_times_seen
			,insights_issue_users_seen
			,insights_issue_last_user
			,insights_issue_first_release
			,insights_issue_last_release
			,insights_issue_status
			,insights_issue_resolved_in_release
			,insights_issue_task_id
			,insights_issue_created
			,insights_issue_updated
			,insights_issue_version
		) VALUES (
			 :insights_issue_project_id
			,:insights_issue_repo_id
			,:insights_issue_fingerprint
			,:insights_issue_title
			,:insights_issue_culprit
			,:insights_issue_level
			,:insights_issue_platform
			,:insights_issue_first_seen
			,:insights_issue_last_seen
			,:insights_issue_times_seen
			,:insights_issue_users_seen
			,:insights_issue_last_user
			,:insights_issue_first_release
			,:insights_issue_last_release
			,:insights_issue_status
			,:insights_issue_resolved_in_release
			,:insights_issue_task_id
			,:insights_issue_created
			,:insights_issue_updated
			,:insights_issue_version
		) RETURNING insights_issue_id`

	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(sqlQuery, mapInternalInsightsIssue(i))
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind insights issue object")
	}

	if err = db.QueryRowContext(ctx, query, arg...).Scan(&i.ID); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to insert insights issue")
	}

	return nil
}

func (s *InsightsIssueStore) Find(ctx context.Context, id int64) (*types.InsightsIssue, error) {
	const sqlQuery = insightsIssueSelectBase + ` WHERE insights_issue_id = $1`

	db := dbtx.GetAccessor(ctx, s.db)

	dst := &insightsIssue{}
	if err := db.GetContext(ctx, dst, sqlQuery, id); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find insights issue")
	}

	return mapInsightsIssue(dst), nil
}

func (s *InsightsIssueStore) FindByFingerprint(
	ctx context.Context,
	projectID int64,
	fingerprint string,
) (*types.InsightsIssue, error) {
	const sqlQuery = insightsIssueSelectBase +
		` WHERE insights_issue_project_id = $1 AND insights_issue_fingerprint = $2`

	db := dbtx.GetAccessor(ctx, s.db)

	dst := &insightsIssue{}
	if err := db.GetContext(ctx, dst, sqlQuery, projectID, fingerprint); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find insights issue by fingerprint")
	}

	return mapInsightsIssue(dst), nil
}

// Update updates the issue using optimistic locking.
func (s *InsightsIssueStore) Update(ctx context.Context, i *types.InsightsIssue) error {
	const sqlQuery = `
		UPDATE insights_issues
		SET
			 insights_issue_version = :insights_issue_version + 1
			,insights_issue_title = :insights_issue_title
			,insights_issue_culprit = :insights_issue_culprit
			,insights_issue_level = :insights_issue_level
			,insights_issue_platform = :insights_issue_platform
			,insights_issue_last_seen = :insights_issue_last_seen
			,insights_issue_times_seen = :insights_issue_times_seen
			,insights_issue_users_seen = :insights_issue_users_seen
			,insights_issue_last_user = :insights_issue_last_user
			,insights_issue_last_release = :insights_issue_last_release
			,insights_issue_status = :insights_issue_status
			,insights_issue_resolved_in_release = :insights_issue_resolved_in_release
			,insights_issue_task_id = :insights_issue_task_id
			,insights_issue_updated = :insights_issue_updated
		WHERE insights_issue_id = :insights_issue_id AND insights_issue_version = :insights_issue_version`

	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(sqlQuery, mapInternalInsightsIssue(i))
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind insights issue object")
	}

	result, err := db.ExecContext(ctx, query, arg...)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to update insights issue")
	}

	count, err := result.RowsAffected()
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to get number of updated insights issue rows")
	}
	if count == 0 {
		return gitness_store.ErrVersionConflict
	}

	i.Version++

	return nil
}

func (s *InsightsIssueStore) List(
	ctx context.Context,
	repoID int64,
	filter *types.InsightsIssueFilter,
) ([]*types.InsightsIssue, error) {
	stmt := database.Builder.
		Select(insightsIssueColumns).
		From("insights_issues").
		Where("insights_issue_repo_id = ?", repoID)

	stmt = applyInsightsIssueFilter(stmt, filter)

	var size, page int
	if filter != nil {
		size, page = filter.Size, filter.Page
	}

	stmt = stmt.
		OrderBy("insights_issue_last_seen DESC", "insights_issue_id DESC").
		Limit(database.Limit(size)).
		Offset(database.Offset(page, size))

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to convert insights issue list query to sql: %w", err)
	}

	db := dbtx.GetAccessor(ctx, s.db)

	dst := []*insightsIssue{}
	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed executing insights issue list query")
	}

	out := make([]*types.InsightsIssue, len(dst))
	for i := range dst {
		out[i] = mapInsightsIssue(dst[i])
	}

	return out, nil
}

func (s *InsightsIssueStore) Count(
	ctx context.Context,
	repoID int64,
	filter *types.InsightsIssueFilter,
) (int64, error) {
	stmt := database.Builder.
		Select("COUNT(*)").
		From("insights_issues").
		Where("insights_issue_repo_id = ?", repoID)

	stmt = applyInsightsIssueFilter(stmt, filter)

	sql, args, err := stmt.ToSql()
	if err != nil {
		return 0, fmt.Errorf("failed to convert insights issue count query to sql: %w", err)
	}

	db := dbtx.GetAccessor(ctx, s.db)

	var count int64
	if err = db.QueryRowContext(ctx, sql, args...).Scan(&count); err != nil {
		return 0, database.ProcessSQLErrorf(ctx, err, "Failed executing insights issue count query")
	}

	return count, nil
}

func applyInsightsIssueFilter(
	stmt squirrel.SelectBuilder,
	filter *types.InsightsIssueFilter,
) squirrel.SelectBuilder {
	if filter == nil {
		return stmt
	}
	if len(filter.Statuses) > 0 {
		stmt = stmt.Where(squirrel.Eq{"insights_issue_status": filter.Statuses})
	}
	if filter.Query != "" {
		stmt = stmt.Where("LOWER(insights_issue_title) LIKE ?", "%"+strings.ToLower(filter.Query)+"%")
	}
	return stmt
}

func mapInsightsIssue(i *insightsIssue) *types.InsightsIssue {
	out := &types.InsightsIssue{
		ID:                i.ID,
		Version:           i.Version,
		ProjectID:         i.ProjectID,
		RepoID:            i.RepoID,
		Fingerprint:       i.Fingerprint,
		Title:             i.Title,
		Culprit:           i.Culprit,
		Level:             i.Level,
		Platform:          i.Platform,
		FirstSeen:         i.FirstSeen,
		LastSeen:          i.LastSeen,
		TimesSeen:         i.TimesSeen,
		UsersSeen:         i.UsersSeen,
		LastUser:          i.LastUser,
		FirstRelease:      i.FirstRelease,
		LastRelease:       i.LastRelease,
		Status:            i.Status,
		ResolvedInRelease: i.ResolvedInRelease,
		Created:           i.Created,
		Updated:           i.Updated,
	}
	if i.TaskID.Valid {
		v := i.TaskID.Int64
		out.TaskID = &v
	}
	return out
}

func mapInternalInsightsIssue(i *types.InsightsIssue) *insightsIssue {
	out := &insightsIssue{
		ID:                i.ID,
		Version:           i.Version,
		ProjectID:         i.ProjectID,
		RepoID:            i.RepoID,
		Fingerprint:       i.Fingerprint,
		Title:             i.Title,
		Culprit:           i.Culprit,
		Level:             i.Level,
		Platform:          i.Platform,
		FirstSeen:         i.FirstSeen,
		LastSeen:          i.LastSeen,
		TimesSeen:         i.TimesSeen,
		UsersSeen:         i.UsersSeen,
		LastUser:          i.LastUser,
		FirstRelease:      i.FirstRelease,
		LastRelease:       i.LastRelease,
		Status:            i.Status,
		ResolvedInRelease: i.ResolvedInRelease,
		Created:           i.Created,
		Updated:           i.Updated,
	}
	if i.TaskID != nil {
		out.TaskID = null.IntFrom(*i.TaskID)
	}
	return out
}

// ---------------------------------------------------------------------------
// events
// ---------------------------------------------------------------------------

// NewInsightsEventStore returns a new InsightsEventStore.
func NewInsightsEventStore(db *sqlx.DB) *InsightsEventStore {
	return &InsightsEventStore{db: db}
}

// InsightsEventStore implements store.InsightsEventStore backed by a relational database.
type InsightsEventStore struct {
	db *sqlx.DB
}

type insightsEvent struct {
	ID          int64  `db:"insights_event_id"`
	IssueID     int64  `db:"insights_event_issue_id"`
	ProjectID   int64  `db:"insights_event_project_id"`
	EventID     string `db:"insights_event_event_id"`
	Timestamp   int64  `db:"insights_event_timestamp"`
	Release     string `db:"insights_event_release"`
	Environment string `db:"insights_event_environment"`
	Level       string `db:"insights_event_level"`
	Message     string `db:"insights_event_message"`
	Payload     string `db:"insights_event_payload"`
	Created     int64  `db:"insights_event_created"`
}

const insightsEventColumns = `
	 insights_event_id
	,insights_event_issue_id
	,insights_event_project_id
	,insights_event_event_id
	,insights_event_timestamp
	,insights_event_release
	,insights_event_environment
	,insights_event_level
	,insights_event_message
	,insights_event_payload
	,insights_event_created`

func (s *InsightsEventStore) Create(ctx context.Context, e *types.InsightsEvent) error {
	const sqlQuery = `
		INSERT INTO insights_events (
			 insights_event_issue_id
			,insights_event_project_id
			,insights_event_event_id
			,insights_event_timestamp
			,insights_event_release
			,insights_event_environment
			,insights_event_level
			,insights_event_message
			,insights_event_payload
			,insights_event_created
		) VALUES (
			 :insights_event_issue_id
			,:insights_event_project_id
			,:insights_event_event_id
			,:insights_event_timestamp
			,:insights_event_release
			,:insights_event_environment
			,:insights_event_level
			,:insights_event_message
			,:insights_event_payload
			,:insights_event_created
		) RETURNING insights_event_id`

	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(sqlQuery, mapInternalInsightsEvent(e))
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind insights event object")
	}

	if err = db.QueryRowContext(ctx, query, arg...).Scan(&e.ID); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to insert insights event")
	}

	return nil
}

func (s *InsightsEventStore) Exists(ctx context.Context, projectID int64, eventID string) (bool, error) {
	const sqlQuery = `
		SELECT COUNT(*) FROM insights_events
		WHERE insights_event_project_id = $1 AND insights_event_event_id = $2`

	db := dbtx.GetAccessor(ctx, s.db)

	var count int64
	if err := db.QueryRowContext(ctx, sqlQuery, projectID, eventID).Scan(&count); err != nil {
		return false, database.ProcessSQLErrorf(ctx, err, "Failed to check insights event existence")
	}

	return count > 0, nil
}

func (s *InsightsEventStore) ListByIssue(
	ctx context.Context,
	issueID int64,
	limit int,
) ([]*types.InsightsEvent, error) {
	const sqlQuery = `SELECT` + insightsEventColumns + `
		FROM insights_events
		WHERE insights_event_issue_id = $1
		ORDER BY insights_event_timestamp DESC, insights_event_id DESC
		LIMIT $2`

	db := dbtx.GetAccessor(ctx, s.db)

	dst := []*insightsEvent{}
	if err := db.SelectContext(ctx, &dst, sqlQuery, issueID, database.Limit(limit)); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to list insights events")
	}

	out := make([]*types.InsightsEvent, len(dst))
	for i, e := range dst {
		out[i] = &types.InsightsEvent{
			ID: e.ID, IssueID: e.IssueID, ProjectID: e.ProjectID, EventID: e.EventID,
			Timestamp: e.Timestamp, Release: e.Release, Environment: e.Environment,
			Level: e.Level, Message: e.Message, Payload: e.Payload, Created: e.Created,
		}
	}

	return out, nil
}

func mapInternalInsightsEvent(e *types.InsightsEvent) *insightsEvent {
	return &insightsEvent{
		ID: e.ID, IssueID: e.IssueID, ProjectID: e.ProjectID, EventID: e.EventID,
		Timestamp: e.Timestamp, Release: e.Release, Environment: e.Environment,
		Level: e.Level, Message: e.Message, Payload: e.Payload, Created: e.Created,
	}
}
