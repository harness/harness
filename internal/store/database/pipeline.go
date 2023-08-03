package database

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/harness/gitness/internal/store"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

var _ store.PipelineStore = (*pipelineStore)(nil)

const (
	pipelineQueryBase = `
		SELECT
		pipeline_id,
		pipeline_description,
		pipeline_parent_id,
		pipeline_uid,
		pipeline_seq,
		pipeline_repo_id,
		pipeline_repo_type,
		pipeline_repo_name,
		pipeline_default_branch,
		pipeline_config_path,
		pipeline_created,
		pipeline_updated,
		pipeline_version
		FROM pipelines
		`

	pipelineColumns = `
	pipeline_id,
	pipeline_description,
	pipeline_parent_id,
	pipeline_uid,
	pipeline_seq,
	pipeline_repo_id,
	pipeline_repo_type,
	pipeline_repo_name,
	pipeline_default_branch,
	pipeline_config_path,
	pipeline_created,
	pipeline_updated,
	pipeline_version
	`

	pipelineInsertStmt = `
	INSERT INTO pipelines (
		pipeline_description,
		pipeline_parent_id,
		pipeline_uid,
		pipeline_seq,
		pipeline_repo_id,
		pipeline_repo_type,
		pipeline_repo_name,
		pipeline_default_branch,
		pipeline_config_path,
		pipeline_created,
		pipeline_updated,
		pipeline_version
	) VALUES (
		:pipeline_description,
		:pipeline_parent_id,
		:pipeline_uid,
		:pipeline_seq,
		:pipeline_repo_id,
		:pipeline_repo_type,
		:pipeline_repo_name,
		:pipeline_default_branch,
		:pipeline_config_path,
		:pipeline_created,
		:pipeline_updated,
		:pipeline_version
	) RETURNING pipeline_id`

	pipelineUpdateStmt = `
	UPDATE pipelines
	SET
		pipeline_description = :pipeline_description,
		pipeline_parent_id = :pipeline_parent_id,
		pipeline_uid = :pipeline_uid,
		pipeline_seq = :pipeline_seq,
		pipeline_repo_id = :pipeline_repo_id,
		pipeline_repo_type = :pipeline_repo_type,
		pipeline_repo_name = :pipeline_repo_name,
		pipeline_default_branch = :pipeline_default_branch,
		pipeline_config_path = :pipeline_config_path,
		pipeline_created = :pipeline_created,
		pipeline_updated = :pipeline_updated,
		pipeline_version = :pipeline_version
	WHERE pipeline_id = :pipeline_id AND pipeline_version = :pipeline_version - 1`
)

// NewPipelineStore returns a new PipelineStore.
func NewPipelineStore(db *sqlx.DB) *pipelineStore {
	return &pipelineStore{
		db: db,
	}
}

type pipelineStore struct {
	db *sqlx.DB
}

// Find returns a pipeline given a pipeline ID
func (s *pipelineStore) Find(ctx context.Context, id int64) (*types.Pipeline, error) {
	const findQueryStmt = pipelineQueryBase + `
		WHERE pipeline_id = $1`
	db := dbtx.GetAccessor(ctx, s.db)

	dst := new(types.Pipeline)
	if err := db.GetContext(ctx, dst, findQueryStmt, id); err != nil {
		return nil, database.ProcessSQLErrorf(err, "Failed to find pipeline")
	}
	return dst, nil
}

// FindByUID returns a pipeline in a given space with a given UID
func (s *pipelineStore) FindByUID(ctx context.Context, spaceID int64, uid string) (*types.Pipeline, error) {
	const findQueryStmt = pipelineQueryBase + `
		WHERE pipeline_parent_id = $1 AND pipeline_uid = $2`
	db := dbtx.GetAccessor(ctx, s.db)

	dst := new(types.Pipeline)
	if err := db.GetContext(ctx, dst, findQueryStmt, spaceID, uid); err != nil {
		return nil, database.ProcessSQLErrorf(err, "Failed to find pipeline")
	}
	return dst, nil
}

// Create creates a pipeline
func (s *pipelineStore) Create(ctx context.Context, pipeline *types.Pipeline) error {
	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(pipelineInsertStmt, pipeline)
	if err != nil {
		return database.ProcessSQLErrorf(err, "Failed to bind pipeline object")
	}

	if err = db.QueryRowContext(ctx, query, arg...).Scan(&pipeline.ID); err != nil {
		return database.ProcessSQLErrorf(err, "Pipeline query failed")
	}

	return nil
}

func (s *pipelineStore) Update(ctx context.Context, pipeline *types.Pipeline) (*types.Pipeline, error) {
	updatedAt := time.Now()

	pipeline.Version++
	pipeline.Updated = updatedAt.UnixMilli()

	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(pipelineUpdateStmt, pipeline)
	if err != nil {
		return nil, database.ProcessSQLErrorf(err, "Failed to bind pipeline object")
	}

	result, err := db.ExecContext(ctx, query, arg...)
	if err != nil {
		return nil, database.ProcessSQLErrorf(err, "Failed to update pipeline")
	}

	count, err := result.RowsAffected()
	if err != nil {
		return nil, database.ProcessSQLErrorf(err, "Failed to get number of updated rows")
	}

	if count == 0 {
		return nil, gitness_store.ErrVersionConflict
	}

	return s.Find(ctx, pipeline.ID)

}

// List lists all the pipelines present in a space
func (s *pipelineStore) List(ctx context.Context, parentID int64, opts *types.PipelineFilter) ([]*types.Pipeline, error) {
	stmt := database.Builder.
		Select(pipelineColumns).
		From("pipelines").
		Where("pipeline_parent_id = ?", fmt.Sprint(parentID))

	if opts.Query != "" {
		stmt = stmt.Where("LOWER(pipeline_uid) LIKE ?", fmt.Sprintf("%%%s%%", strings.ToLower(opts.Query)))
	}

	stmt = stmt.Limit(database.Limit(opts.Size))
	stmt = stmt.Offset(database.Offset(opts.Page, opts.Size))

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, s.db)

	dst := []*types.Pipeline{}
	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(err, "Failed executing custom list query")
	}

	return dst, nil
}

// Delete deletes a pipeline given a pipeline ID
func (s *pipelineStore) Delete(ctx context.Context, id int64) error {
	const pipelineDeleteStmt = `
		DELETE FROM pipelines
		WHERE pipeline_id = $1`

	db := dbtx.GetAccessor(ctx, s.db)

	if _, err := db.ExecContext(ctx, pipelineDeleteStmt, id); err != nil {
		return database.ProcessSQLErrorf(err, "Could not delete pipeline")
	}

	return nil
}
