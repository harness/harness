// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package types

import "github.com/harness/gitness/types/enum"

// Execution represents an instance of a pipeline execution.
type Execution struct {
	ID           int64             `json:"-"`
	PipelineID   int64             `json:"pipeline_id"`
	CreatedBy    int64             `json:"created_by"`
	RepoID       int64             `json:"repo_id"`
	Trigger      string            `json:"trigger,omitempty"`
	Number       int64             `json:"number"`
	Parent       int64             `json:"parent,omitempty"`
	Status       enum.CIStatus     `json:"status"`
	Error        string            `json:"error,omitempty"`
	Event        string            `json:"event,omitempty"`
	Action       string            `json:"action,omitempty"`
	Link         string            `json:"link,omitempty"`
	Timestamp    int64             `json:"timestamp,omitempty"`
	Title        string            `json:"title,omitempty"`
	Message      string            `json:"message,omitempty"`
	Before       string            `json:"before,omitempty"`
	After        string            `json:"after,omitempty"`
	Ref          string            `json:"ref,omitempty"`
	Fork         string            `json:"source_repo,omitempty"`
	Source       string            `json:"source,omitempty"`
	Target       string            `json:"target,omitempty"`
	Author       string            `json:"author_login,omitempty"`
	AuthorName   string            `json:"author_name,omitempty"`
	AuthorEmail  string            `json:"author_email,omitempty"`
	AuthorAvatar string            `json:"author_avatar,omitempty"`
	Sender       string            `json:"sender,omitempty"`
	Params       map[string]string `json:"params,omitempty"`
	Cron         string            `json:"cron,omitempty"`
	Deploy       string            `json:"deploy_to,omitempty"`
	DeployID     int64             `json:"deploy_id,omitempty"`
	Debug        bool              `json:"debug,omitempty"`
	Started      int64             `json:"started,omitempty"`
	Finished     int64             `json:"finished,omitempty"`
	Created      int64             `json:"created"`
	Updated      int64             `json:"updated"`
	Version      int64             `json:"-"`
	Stages       []*Stage          `json:"stages,omitempty"`
}
