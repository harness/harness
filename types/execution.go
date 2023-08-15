// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package types

// Execution represents an instance of a pipeline execution.
type Execution struct {
	ID           int64    `db:"execution_id"             json:"id"`
	PipelineID   int64    `db:"execution_pipeline_id"    json:"pipeline_id"`
	RepoID       int64    `db:"execution_repo_id"        json:"repo_id"`
	Trigger      string   `db:"execution_trigger"        json:"trigger"`
	Number       int64    `db:"execution_number"         json:"number"`
	Parent       int64    `db:"execution_parent"         json:"parent,omitempty"`
	Status       string   `db:"execution_status"         json:"status"`
	Error        string   `db:"execution_error"          json:"error,omitempty"`
	Event        string   `db:"execution_event"          json:"event"`
	Action       string   `db:"execution_action"         json:"action"`
	Link         string   `db:"execution_link"           json:"link"`
	Timestamp    int64    `db:"execution_timestamp"      json:"timestamp"`
	Title        string   `db:"execution_title"          json:"title,omitempty"`
	Message      string   `db:"execution_message"        json:"message"`
	Before       string   `db:"execution_before"         json:"before"`
	After        string   `db:"execution_after"          json:"after"`
	Ref          string   `db:"execution_ref"            json:"ref"`
	Fork         string   `db:"execution_source_repo"    json:"source_repo"`
	Source       string   `db:"execution_source"         json:"source"`
	Target       string   `db:"execution_target"         json:"target"`
	Author       string   `db:"execution_author"         json:"author_login"`
	AuthorName   string   `db:"execution_author_name"    json:"author_name"`
	AuthorEmail  string   `db:"execution_author_email"   json:"author_email"`
	AuthorAvatar string   `db:"execution_author_avatar"  json:"author_avatar"`
	Sender       string   `db:"execution_sender"         json:"sender"`
	Params       string   `db:"execution_params"         json:"params,omitempty"`
	Cron         string   `db:"execution_cron"           json:"cron,omitempty"`
	Deploy       string   `db:"execution_deploy"         json:"deploy_to,omitempty"`
	DeployID     int64    `db:"execution_deploy_id"      json:"deploy_id,omitempty"`
	Debug        bool     `db:"execution_debug"          json:"debug,omitempty"`
	Started      int64    `db:"execution_started"        json:"started"`
	Finished     int64    `db:"execution_finished"       json:"finished"`
	Created      int64    `db:"execution_created"        json:"created"`
	Updated      int64    `db:"execution_updated"        json:"updated"`
	Version      int64    `db:"execution_version"        json:"version"`
	Stages       []*Stage `db:"-"                        json:"stages,omitempty"`
}
