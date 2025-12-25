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

package repo

import (
	"context"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types"
)

type CheckInput struct {
	ParentRef     string `json:"parent_ref"`
	Identifier    string `json:"identifier"`
	DefaultBranch string `json:"default_branch"`
	Description   string `json:"description"`
	IsPublic      bool   `json:"is_public"`

	IsFork       bool   `json:"is_fork,omitempty"`
	UpstreamPath string `json:"upstream_path,omitempty"`

	CreateFileOptions
}

// Check defines the interface for adding extra checks during repository operations.
type Check interface {
	// Create allows adding extra check during create repo operations
	Create(ctx context.Context, session *auth.Session, in *CheckInput) error
	LifecycleRestriction(ctx context.Context, session *auth.Session, repo *types.RepositoryCore) error
}
