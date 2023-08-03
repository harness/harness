package database

import (
	"github.com/harness/gitness/internal/store"
	"github.com/jmoiron/sqlx"
)

var _ store.ExecutionStore = (*executionStore)(nil)

// NewSpaceStore returns a new PathStore.
func NewExecutionStore(db *sqlx.DB) *executionStore {
	return &executionStore{
		db: db,
	}
}

type executionStore struct {
	db *sqlx.DB
}

const (
	executionColumns = `
		execution_id
		,execution_scm_type
		,execution_repo_id
		,execution_trigger
		,execution_number
		,execution_parent
		,execution_status
		,execution_error
		,execution_event
		,execution_action
		,execution_link
		,execution_timestamp
		,execution_title
		,execution_message
		,execution_before
		,execution_after
		,execution_ref
		,execution_source_repo
		,execution_source
		,execution_target
		,execution_author
		,execution_author_name
		,execution_author_email
		,execution_author_avatar
		,execution_sender
		,execution_params
		,execution_cron
		,execution_deploy
		,execution_deploy_id
		,execution_debug
		,execution_started
		,execution_finished
		,execution_created
		,execution_updated
		,execution_version
		`
)
