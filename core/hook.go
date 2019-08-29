// Copyright 2019 Drone IO, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package core

import (
	"context"
	"net/http"
)

// Hook action constants.
const (
	ActionOpen   = "open"
	ActionCreate = "create"
	ActionDelete = "delete"
	ActionSync   = "sync"
)

// Hook represents the payload of a post-commit hook.
type Hook struct {
	Parent       int64             `json:"parent"`
	Trigger      string            `json:"trigger"`
	Event        string            `json:"event"`
	Action       string            `json:"action"`
	Link         string            `json:"link"`
	Timestamp    int64             `json:"timestamp"`
	Title        string            `json:"title"`
	Message      string            `json:"message"`
	Before       string            `json:"before"`
	After        string            `json:"after"`
	Ref          string            `json:"ref"`
	Fork         string            `json:"hook"`
	Source       string            `json:"source"`
	Target       string            `json:"target"`
	Author       string            `json:"author_login"`
	AuthorName   string            `json:"author_name"`
	AuthorEmail  string            `json:"author_email"`
	AuthorAvatar string            `json:"author_avatar"`
	Deployment   string            `json:"deploy_to"`
	DeploymentID int64             `json:"deploy_id"`
	Cron         string            `json:"cron"`
	Sender       string            `json:"sender"`
	Params       map[string]string `json:"params"`
}

// HookService manages post-commit hooks in the external
// source code management service (e.g. GitHub).
type HookService interface {
	Create(ctx context.Context, user *User, repo *Repository) error
	Delete(ctx context.Context, user *User, repo *Repository) error
}

// HookParser parses a post-commit hook from the source
// code management system, and returns normalized data.
type HookParser interface {
	Parse(req *http.Request, secretFunc func(string) string) (*Hook, *Repository, error)
}
