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
const defaultResourceIdentifier = "default"
const infraProviderResourceMissingErr = "Failed to find infraProviderResource: resource not found"

var (
	// errSecretRequiresParent if the user tries to create a secret without a parent space.
	ErrGitspaceRequiresParent = usererror.BadRequest(
		"Parent space required - standalone gitspace are not supported.")
)

// CreateInput is the input used for create operations.
type CreateInput struct {
	Identifier         string                    `json:"identifier"`
	Name               string                    `json:"name"`
	SpaceRef           string                    `json:"space_ref"` // Ref of the parent space
	IDE                enum.IDEType              `json:"ide"`
	ResourceIdentifier string                    `json:"resource_identifier"`
	ResourceSpaceRef   string                    `json:"resource_space_ref"`
	CodeRepoURL        string                    `json:"code_repo_url"`
	CodeRepoType       enum.GitspaceCodeRepoType `json:"code_repo_type"`
	CodeRepoRef        *string                   `json:"code_repo_ref"`
	Branch             string                    `json:"branch"`
	DevcontainerPath   *string                   `json:"devcontainer_path"`
	Metadata           map[string]string         `json:"metadata"`
}

// Create creates a new gitspace.
func (c *Controller) Create(
	ctx context.Context,
	session *auth.Session,
	in *CreateInput,
) (*types.GitspaceConfig, error) {
	space, err := c.spaceStore.FindByRef(ctx, in.SpaceRef)
	if err != nil {
		return nil, fmt.Errorf("failed to find parent by ref: %w", err)
	}
	if err = apiauth.CheckGitspace(
		ctx,
		c.authorizer,
		session,
		space.Path,
		"",
		enum.PermissionGitspaceEdit); err != nil {
		return nil, err
	}
	// check if it's an internal repo
	if in.CodeRepoType == enum.CodeRepoTypeGitness && *in.CodeRepoRef != "" {
		repo, err := c.repoStore.FindByRef(ctx, *in.CodeRepoRef)
		if err != nil {
			return nil, fmt.Errorf("couldn't fetch repo for the user: %w", err)
		}
		if err = apiauth.CheckRepo(
			ctx,
			c.authorizer,
			session,
			repo,
			enum.PermissionRepoView); err != nil {
			return nil, err
		}
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
	var gitspaceConfig *types.GitspaceConfig
	resourceIdentifier := in.ResourceIdentifier
	// assume resource to be in same space if it's not explicitly specified.
	if in.ResourceSpaceRef == "" {
		in.ResourceSpaceRef = in.SpaceRef
	}
	resourceSpace, err := c.spaceStore.FindByRef(ctx, in.ResourceSpaceRef)
	if err != nil {
		return nil, fmt.Errorf("failed to find parent by ref: %w", err)
	}
	if err = apiauth.CheckInfraProvider(
		ctx,
		c.authorizer,
		session,
		resourceSpace.Path,
		resourceIdentifier,
		enum.PermissionInfraProviderAccess); err != nil {
		return nil, err
	}
	err = c.createOrFindInfraProviderResource(ctx, resourceSpace, resourceIdentifier, now)
	if err != nil {
		return nil, err
	}
	err = c.tx.WithTx(ctx, func(ctx context.Context) error {
		infraProviderResource, err := c.infraProviderSvc.FindResourceByIdentifier(
			ctx,
			resourceSpace.ID,
			resourceIdentifier)
		if err != nil {
			return fmt.Errorf("could not find infra provider resource : %q %w", resourceIdentifier, err)
		}
		gitspaceConfig = &types.GitspaceConfig{
			Identifier:                      identifier,
			Name:                            in.Name,
			IDE:                             in.IDE,
			InfraProviderResourceID:         infraProviderResource.ID,
			InfraProviderResourceIdentifier: infraProviderResource.Identifier,
			CodeRepoType:                    in.CodeRepoType,
			State:                           enum.GitspaceStateUninitialized,
			CodeRepoURL:                     in.CodeRepoURL,
			CodeRepoRef:                     in.CodeRepoRef,
			Branch:                          in.Branch,
			DevcontainerPath:                in.DevcontainerPath,
			UserID:                          session.Principal.UID,
			SpaceID:                         space.ID,
			SpacePath:                       space.Path,
			Created:                         now,
			Updated:                         now,
		}
		err = c.gitspaceConfigStore.Create(ctx, gitspaceConfig)
		if err != nil {
			return fmt.Errorf("failed to create gitspace config for : %q %w", identifier, err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return gitspaceConfig, nil
}

func (c *Controller) createOrFindInfraProviderResource(
	ctx context.Context,
	parentSpace *types.Space,
	resourceIdentifier string,
	now int64,
) error {
	_, err := c.infraProviderSvc.FindResourceByIdentifier(
		ctx,
		parentSpace.ID,
		resourceIdentifier)
	if err != nil &&
		err.Error() == infraProviderResourceMissingErr &&
		resourceIdentifier == defaultResourceIdentifier {
		err = c.autoCreateDefaultResource(ctx, parentSpace, now)
		if err != nil {
			return err
		}
	} else if err != nil {
		return fmt.Errorf("could not find infra provider resource : %q %w", resourceIdentifier, err)
	}
	return err
}

func (c *Controller) autoCreateDefaultResource(ctx context.Context, parentSpace *types.Space, now int64) error {
	defaultDockerConfig := &types.InfraProviderConfig{
		Identifier: defaultResourceIdentifier,
		Name:       "default docker infrastructure",
		Type:       enum.InfraProviderTypeDocker,
		SpaceID:    parentSpace.ID,
		SpacePath:  parentSpace.Path,
		Created:    now,
		Updated:    now,
	}
	defaultResource := types.InfraProviderResource{
		Identifier:                    defaultResourceIdentifier,
		Name:                          "Standard Docker Resource",
		InfraProviderConfigIdentifier: defaultDockerConfig.Identifier,
		InfraProviderType:             enum.InfraProviderTypeDocker,
		CPU:                           wrapString("any"),
		Memory:                        wrapString("any"),
		Disk:                          wrapString("any"),
		Network:                       wrapString("standard"),
		SpaceID:                       parentSpace.ID,
		SpacePath:                     parentSpace.Path,
		Created:                       now,
		Updated:                       now,
	}
	defaultDockerConfig.Resources = []types.InfraProviderResource{defaultResource}
	err := c.infraProviderSvc.CreateInfraProvider(ctx, defaultDockerConfig)
	if err != nil {
		return fmt.Errorf("could not auto-create the infra provider: %w", err)
	}
	return nil
}

func wrapString(str string) *string {
	return &str
}

func (c *Controller) sanitizeCreateInput(in *CreateInput) error {
	if err := check.Identifier(in.Identifier); err != nil {
		return err
	}
	if err := check.Identifier(in.ResourceIdentifier); err != nil {
		return err
	}
	parentRefAsID, err := strconv.ParseInt(in.SpaceRef, 10, 64)
	if (err == nil && parentRefAsID <= 0) || (len(strings.TrimSpace(in.SpaceRef)) == 0) {
		return ErrGitspaceRequiresParent
	}

	return nil
}
