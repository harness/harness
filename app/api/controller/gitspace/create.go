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

package gitspace

import (
	"context"
	"errors"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// TODO Stubbed Impl
// CreateInput is the input used for create operations.
type CreateInput struct {
	Identifier              string            `json:"identifier"`
	Name                    string            `json:"name"`
	SpaceRef                string            `json:"space_ref"` // Ref of the parent space
	IDE                     enum.IDEType      `json:"ide"`
	InfraProviderResourceID string            `json:"resource_identifier"`
	CodeRepoURL             string            `json:"code_repo_url"`
	Branch                  string            `json:"branch"`
	DevcontainerPath        string            `json:"devcontainer_path"`
	Metadata                map[string]string `json:"metadata"`
}

// Create creates a new gitspace.
func (c *Controller) Create(
	_ context.Context,
	_ *auth.Session,
	_ *CreateInput,
) (*types.GitspaceConfig, error) {
	return nil, errors.New("unimplemented")
}
