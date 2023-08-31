// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package database

import (
	"database/sql"

	"github.com/harness/gitness/types"
)

// pipelineExecutionjoin struct represents a joined row between pipelines and executions
type pipelineExecutionJoin struct {
	*types.Pipeline
	ID           sql.NullInt64  `db:"execution_id"`
	PipelineID   sql.NullInt64  `db:"execution_pipeline_id"`
	RepoID       sql.NullInt64  `db:"execution_repo_id"`
	Trigger      sql.NullString `db:"execution_trigger"`
	Number       sql.NullInt64  `db:"execution_number"`
	Status       sql.NullString `db:"execution_status"`
	Error        sql.NullString `db:"execution_error"`
	Link         sql.NullString `db:"execution_link"`
	Timestamp    sql.NullInt64  `db:"execution_timestamp"`
	Title        sql.NullString `db:"execution_title"`
	Fork         sql.NullString `db:"execution_source_repo"`
	Source       sql.NullString `db:"execution_source"`
	Target       sql.NullString `db:"execution_target"`
	Author       sql.NullString `db:"execution_author"`
	AuthorName   sql.NullString `db:"execution_author_name"`
	AuthorEmail  sql.NullString `db:"execution_author_email"`
	AuthorAvatar sql.NullString `db:"execution_author_avatar"`
	Started      sql.NullInt64  `db:"execution_started"`
	Finished     sql.NullInt64  `db:"execution_finished"`
	Created      sql.NullInt64  `db:"execution_created"`
	Updated      sql.NullInt64  `db:"execution_updated"`
}

func convert(rows []*pipelineExecutionJoin) []*types.Pipeline {
	pipelines := []*types.Pipeline{}
	for _, k := range rows {
		pipeline := convertPipelineJoin(k)
		pipelines = append(pipelines, pipeline)
	}
	return pipelines
}

func convertPipelineJoin(join *pipelineExecutionJoin) *types.Pipeline {
	ret := join.Pipeline
	if !join.ID.Valid {
		return ret
	}
	ret.Execution = &types.Execution{
		ID:           join.ID.Int64,
		PipelineID:   join.PipelineID.Int64,
		RepoID:       join.RepoID.Int64,
		Trigger:      join.Trigger.String,
		Number:       join.Number.Int64,
		Status:       join.Status.String,
		Error:        join.Error.String,
		Link:         join.Link.String,
		Timestamp:    join.Timestamp.Int64,
		Title:        join.Title.String,
		Fork:         join.Fork.String,
		Source:       join.Source.String,
		Target:       join.Target.String,
		Author:       join.Author.String,
		AuthorName:   join.AuthorName.String,
		AuthorEmail:  join.AuthorEmail.String,
		AuthorAvatar: join.AuthorAvatar.String,
		Started:      join.Started.Int64,
		Finished:     join.Finished.Int64,
		Created:      join.Created.Int64,
		Updated:      join.Updated.Int64,
	}
	return ret
}
