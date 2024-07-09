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
	"fmt"
	"strconv"
	"strings"
	"time"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"

	gonanoid "github.com/matoous/go-nanoid"
)

const allowedUIDAlphabet = "abcdefghijklmnopqrstuvwxyz0123456789"

var (
	// errSecretRequiresParent if the user tries to create a secret without a parent space.
	ErrGitspaceRequiresParent = usererror.BadRequest(
		"Parent space required - standalone gitspace are not supported.")
)

// CreateInput is the input used for create operations.
type CreateInput struct {
	Identifier         string            `json:"identifier"`
	Name               string            `json:"name"`
	SpaceRef           string            `json:"space_ref"` // Ref of the parent space
	IDE                enum.IDEType      `json:"ide"`
	ResourceIdentifier string            `json:"resource_identifier"`
	CodeRepoURL        string            `json:"code_repo_url"`
	Branch             string            `json:"branch"`
	DevcontainerPath   *string           `json:"devcontainer_path"`
	Metadata           map[string]string `json:"metadata"`
}

// Create creates a new gitspace.
func (c *Controller) Create(
	ctx context.Context,
	session *auth.Session,
	in *CreateInput,
) (*types.GitspaceConfig, error) {
	parentSpace, err := c.spaceStore.FindByRef(ctx, in.SpaceRef)
	if err != nil {
		return nil, fmt.Errorf("failed to find parent by ref: %w", err)
	}
	if err = apiauth.CheckGitspace(
		ctx,
		c.authorizer,
		session,
		parentSpace.Path,
		"",
		enum.PermissionGitspaceEdit); err != nil {
		return nil, err
	}
	suffixUID, err := gonanoid.Generate(allowedUIDAlphabet, 6)
	if err != nil {
		return nil, fmt.Errorf("could not generate UID for gitspace config : %q %w", in.Identifier, err)
	}
	identifier := strings.ToLower(in.Identifier + "-" + suffixUID)
	if err = c.sanitizeCreateInput(in); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}
	now := time.Now().UnixMilli()
	infraProviderResource, err := c.infraProviderResourceStore.FindByIdentifier(
		ctx,
		parentSpace.ID,
		in.ResourceIdentifier)
	if err != nil {
		return nil, fmt.Errorf("could not find infra provider resource : %q %w", in.ResourceIdentifier, err)
	}
	gitspaceConfig := &types.GitspaceConfig{
		Identifier:                      identifier,
		Name:                            in.Name,
		IDE:                             in.IDE,
		InfraProviderResourceID:         infraProviderResource.ID,
		InfraProviderResourceIdentifier: infraProviderResource.Identifier,
		CodeRepoType:                    enum.CodeRepoTypeUnknown, // TODO fix this
		State:                           enum.GitspaceStateUninitialized,
		CodeRepoURL:                     in.CodeRepoURL,
		Branch:                          in.Branch,
		DevcontainerPath:                in.DevcontainerPath,
		UserID:                          session.Principal.UID,
		SpaceID:                         parentSpace.ID,
		SpacePath:                       parentSpace.Path,
		Created:                         now,
		Updated:                         now,
	}
	err = c.gitspaceConfigStore.Create(ctx, gitspaceConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create gitspace config for : %q %w", identifier, err)
	}
	return gitspaceConfig, nil
}

func (c *Controller) sanitizeCreateInput(in *CreateInput) error {
	if err := check.Identifier(in.Identifier); err != nil {
		return err
	}
	parentRefAsID, err := strconv.ParseInt(in.SpaceRef, 10, 64)
	if (err == nil && parentRefAsID <= 0) || (len(strings.TrimSpace(in.SpaceRef)) == 0) {
		return ErrGitspaceRequiresParent
	}
	return nil
}
