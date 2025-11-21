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
	"database/sql"

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// pipelineExecutionjoin struct represents a joined row between pipelines and executions.
type pipelineExecutionJoin struct {
	*types.Pipeline
	ID           sql.NullInt64  `db:"execution_id"`
	PipelineID   sql.NullInt64  `db:"execution_pipeline_id"`
	Action       sql.NullString `db:"execution_action"`
	Message      sql.NullString `db:"execution_message"`
	After        sql.NullString `db:"execution_after"`
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
		Action:       enum.TriggerAction(join.Action.String),
		Trigger:      join.Trigger.String,
		Number:       join.Number.Int64,
		After:        join.After.String,
		Message:      join.Message.String,
		Status:       enum.ParseCIStatus(join.Status.String),
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

type pipelineRepoJoin struct {
	*types.Pipeline
	RepoID  sql.NullInt64  `db:"repo_id"`
	RepoUID sql.NullString `db:"repo_uid"`
}

func convertPipelineRepoJoins(rows []*pipelineRepoJoin) []*types.Pipeline {
	pipelines := []*types.Pipeline{}
	for _, k := range rows {
		pipeline := convertPipelineRepoJoin(k)
		pipelines = append(pipelines, pipeline)
	}
	return pipelines
}

func convertPipelineRepoJoin(join *pipelineRepoJoin) *types.Pipeline {
	ret := join.Pipeline
	if !join.RepoID.Valid {
		return ret
	}
	ret.RepoUID = join.RepoUID.String
	return ret
}
