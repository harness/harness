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
	"time"

	"github.com/harness/gitness/app/store"
	gitnessstore "github.com/harness/gitness/store"
	"github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

const (
	agentAttemptTable   = `agent_attempts`
	agentAttemptColumns = `
		agent_attempt_repo_id,
		agent_attempt_task_key,
		agent_attempt_model,
		agent_attempt_state,
		agent_attempt_started,
		agent_attempt_finished,
		agent_attempt_error`
	agentAttemptSelectColumns = `agent_attempt_id, ` + agentAttemptColumns

	// agentAttemptLockNamespace separates the advisory locks of the attempts from other advisory locks.
	agentAttemptLockNamespace = int64(0x4147) << 40
)

type agentAttempt struct {
	ID       int64                   `db:"agent_attempt_id"`
	RepoID   int64                   `db:"agent_attempt_repo_id"`
	TaskKey  string                  `db:"agent_attempt_task_key"`
	Model    string                  `db:"agent_attempt_model"`
	State    types.AgentAttemptState `db:"agent_attempt_state"`
	Started  int64                   `db:"agent_attempt_started"`
	Finished *int64                  `db:"agent_attempt_finished"`
	Error    string                  `db:"agent_attempt_error"`
}

type agentAttemptStore struct {
	db *sqlx.DB
	tx dbtx.Transactor
}

// NewAgentAttemptStore returns a new AgentAttemptStore.
func NewAgentAttemptStore(db *sqlx.DB, tx dbtx.Transactor) store.AgentAttemptStore {
	return &agentAttemptStore{db: db, tx: tx}
}

var _ store.AgentAttemptStore = (*agentAttemptStore)(nil)

func (s *agentAttemptStore) isSQLite() bool {
	return strings.HasPrefix(s.db.DriverName(), "sqlite")
}

// TryStart checks failed count, cooldown and max active and inserts the running attempt in one
// transaction. Concurrent reservations of a repo are serialized: by the single writer of sqlite or
// by a transaction scoped advisory lock on the repo in postgres.
func (s *agentAttemptStore) TryStart(
	ctx context.Context,
	in types.AgentAttemptStart,
) (int64, string, error) {
	var (
		id   int64
		deny string
	)
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		id, deny = 0, ""
		db := dbtx.GetAccessor(ctx, s.db)

		if !s.isSQLite() {
			if _, err := db.ExecContext(ctx, "SELECT pg_advisory_xact_lock($1)",
				agentAttemptLockNamespace|(in.RepoID&(1<<40-1))); err != nil {
				return database.ProcessSQLErrorf(ctx, err, "Failed to lock agent attempts of repo %d", in.RepoID)
			}
		}

		var stats struct {
			Failed      int   `db:"failed"`
			LastStarted int64 `db:"last_started"`
		}
		const taskSQL = `SELECT
				COALESCE(SUM(CASE WHEN agent_attempt_state = 'failed' THEN 1 ELSE 0 END), 0) AS failed,
				COALESCE(MAX(agent_attempt_started), 0) AS last_started
			FROM agent_attempts WHERE agent_attempt_task_key = $1`
		if err := db.GetContext(ctx, &stats, taskSQL, in.TaskKey); err != nil {
			return database.ProcessSQLErrorf(ctx, err, "Failed to load attempts of task %q", in.TaskKey)
		}
		if in.MaxFailed > 0 && stats.Failed >= in.MaxFailed {
			deny = types.AgentAttemptDenyTooManyFailures
			return nil
		}
		if stats.LastStarted != 0 && in.At.UnixMilli()-stats.LastStarted < in.Cooldown.Milliseconds() {
			deny = types.AgentAttemptDenyCooldown
			return nil
		}

		var active int
		const activeSQL = `SELECT COUNT(*) FROM agent_attempts
			WHERE agent_attempt_repo_id = $1 AND agent_attempt_state = 'running'`
		if err := db.GetContext(ctx, &active, activeSQL, in.RepoID); err != nil {
			return database.ProcessSQLErrorf(ctx, err, "Failed to count active attempts of repo %d", in.RepoID)
		}
		if active >= in.MaxActive {
			deny = types.AgentAttemptDenyMaxActive
			return nil
		}

		stmt := database.Builder.
			Insert(agentAttemptTable).
			Columns(agentAttemptColumns).
			Values(in.RepoID, in.TaskKey, in.Model, types.AgentAttemptRunning, in.At.UnixMilli(), nil, "").
			Suffix("RETURNING agent_attempt_id")
		sql, args, err := stmt.ToSql()
		if err != nil {
			return errors.Wrap(err, "Failed to convert squirrel builder to sql")
		}
		if err = db.QueryRowContext(ctx, sql, args...).Scan(&id); err != nil {
			return database.ProcessSQLErrorf(ctx, err, "Failed to create agent attempt for task %q", in.TaskKey)
		}
		return nil
	})
	if err != nil {
		return 0, "", err
	}
	return id, deny, nil
}

func (s *agentAttemptStore) Finish(
	ctx context.Context,
	attemptID int64,
	state types.AgentAttemptState,
	errMsg string,
) error {
	if state != types.AgentAttemptSucceeded && state != types.AgentAttemptFailed {
		return fmt.Errorf("invalid final agent attempt state %q", state)
	}
	stmt := database.Builder.
		Update(agentAttemptTable).
		Set("agent_attempt_state", state).
		Set("agent_attempt_finished", time.Now().UnixMilli()).
		Set("agent_attempt_error", errMsg).
		Where("agent_attempt_id = ?", attemptID)
	sql, args, err := stmt.ToSql()
	if err != nil {
		return errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}
	db := dbtx.GetAccessor(ctx, s.db)
	res, err := db.ExecContext(ctx, sql, args...)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to finish agent attempt %d", attemptID)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to finish agent attempt %d", attemptID)
	}
	if n == 0 {
		return gitnessstore.ErrResourceNotFound
	}
	return nil
}

func (s *agentAttemptStore) ListByTask(ctx context.Context, taskKey string) ([]*types.AgentAttempt, error) {
	stmt := database.Builder.
		Select(agentAttemptSelectColumns).
		From(agentAttemptTable).
		Where("agent_attempt_task_key = ?", taskKey).
		OrderBy("agent_attempt_started ASC", "agent_attempt_id ASC")
	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}
	var dst []*agentAttempt
	db := dbtx.GetAccessor(ctx, s.db)
	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to list attempts of task %q", taskKey)
	}
	out := make([]*types.AgentAttempt, len(dst))
	for i, a := range dst {
		out[i] = &types.AgentAttempt{
			ID: a.ID, RepoID: a.RepoID, TaskKey: a.TaskKey, Model: a.Model, State: a.State,
			Started: a.Started, Finished: a.Finished, Error: a.Error,
		}
	}
	return out, nil
}

func (s *agentAttemptStore) CountActive(ctx context.Context, repoID int64) (int, error) {
	const q = `SELECT COUNT(*) FROM agent_attempts WHERE agent_attempt_repo_id = $1 AND agent_attempt_state = 'running'`
	var n int
	db := dbtx.GetAccessor(ctx, s.db)
	if err := db.GetContext(ctx, &n, q, repoID); err != nil {
		return 0, database.ProcessSQLErrorf(ctx, err, "Failed to count active attempts of repo %d", repoID)
	}
	return n, nil
}
