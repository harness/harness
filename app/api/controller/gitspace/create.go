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
	"regexp"
	"strconv"
	"strings"
	"time"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/paths"
	"github.com/harness/gitness/app/services/gitspace"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"

	gonanoid "github.com/matoous/go-nanoid"
)

const (
	defaultResourceIdentifier               = "default"
	maxGitspaceConfigIdentifierPrefixLength = 50
)

var (
	// ErrGitspaceRequiresParent if the user tries to create a secret without a parent space.
	ErrGitspaceRequiresParent = usererror.BadRequest(
		"Parent space required - standalone gitspace are not supported.")
)

// CreateInput is the input used for create operations.
type CreateInput struct {
	Identifier                    string                    `json:"identifier"`
	Name                          string                    `json:"name"`
	SpaceRef                      string                    `json:"space_ref"` // Ref of the parent space
	IDE                           enum.IDEType              `json:"ide"`
	InfraProviderConfigIdentifier string                    `json:"infra_provider_config_identifier"`
	ResourceIdentifier            string                    `json:"resource_identifier"`
	ResourceSpaceRef              string                    `json:"resource_space_ref"`
	CodeRepoURL                   string                    `json:"code_repo_url"`
	CodeRepoType                  enum.GitspaceCodeRepoType `json:"code_repo_type"`
	CodeRepoRef                   *string                   `json:"code_repo_ref"`
	Branch                        string                    `json:"branch"`
	DevcontainerPath              *string                   `json:"devcontainer_path"`
	Metadata                      map[string]string         `json:"metadata"`
	SSHTokenIdentifier            string                    `json:"ssh_token_identifier"`
}

// Create creates a new gitspace.
func (c *Controller) Create(
	ctx context.Context,
	session *auth.Session,
	in *CreateInput,
) (*types.GitspaceConfig, error) {
	space, err := c.spaceFinder.FindByRef(ctx, in.SpaceRef)
	if err != nil {
		return nil, fmt.Errorf("failed to find parent by ref: %w", err)
	}
	if err = c.sanitizeCreateInput(in); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
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

	err = c.gitspaceLimiter.Usage(ctx, space.ID)
	if err != nil {
		return nil, err
	}

	// check if it's an internal repo
	if in.CodeRepoType == enum.CodeRepoTypeGitness && *in.CodeRepoRef != "" {
		repo, err := c.repoFinder.FindByRef(ctx, *in.CodeRepoRef)
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
	identifier, err := buildIdentifier(in.Identifier)
	if err != nil {
		return nil, fmt.Errorf("could not generate identrifier for gitspace config : %q %w", in.Identifier, err)
	}
	now := time.Now().UnixMilli()
	var gitspaceConfig *types.GitspaceConfig
	// assume resource to be in same space if it's not explicitly specified.
	if in.ResourceSpaceRef == "" {
		rootSpaceRef, _, err := paths.DisectRoot(in.SpaceRef)
		if err != nil {
			return nil, fmt.Errorf("unable to find root space path for %s: %w", in.SpaceRef, err)
		}
		in.ResourceSpaceRef = rootSpaceRef
	}
	resourceIdentifier := in.ResourceIdentifier
	resourceSpace, err := c.spaceFinder.FindByRef(ctx, in.ResourceSpaceRef)
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

	// TODO: Temp fix to ensure the gitspace creation doesnt fail. Once the FE starts sending this field in the
	// request, remove this.
	if in.InfraProviderConfigIdentifier == "" {
		in.InfraProviderConfigIdentifier = defaultResourceIdentifier
	}

	infraProviderResource, err := c.createOrFindInfraProviderResource(ctx, resourceSpace, resourceIdentifier,
		in.InfraProviderConfigIdentifier, now)
	if err != nil {
		return nil, err
	}
	err = c.tx.WithTx(ctx, func(ctx context.Context) error {
		codeRepo := types.CodeRepo{
			URL:              in.CodeRepoURL,
			Ref:              in.CodeRepoRef,
			Type:             in.CodeRepoType,
			Branch:           in.Branch,
			DevcontainerPath: in.DevcontainerPath,
		}

		principal := session.Principal
		principalID := principal.ID
		user := types.GitspaceUser{
			Identifier:  principal.UID,
			Email:       principal.Email,
			DisplayName: principal.DisplayName,
			ID:          &principalID}
		gitspaceConfig = &types.GitspaceConfig{
			Identifier:         identifier,
			Name:               in.Name,
			IDE:                in.IDE,
			State:              enum.GitspaceStateUninitialized,
			SpaceID:            space.ID,
			SpacePath:          space.Path,
			Created:            now,
			Updated:            now,
			SSHTokenIdentifier: in.SSHTokenIdentifier,
			CodeRepo:           codeRepo,
			GitspaceUser:       user,
		}
		gitspaceConfig.InfraProviderResource = *infraProviderResource
		err = c.gitspaceSvc.Create(ctx, gitspaceConfig)
		if err != nil {
			return fmt.Errorf("failed to create gitspace config for : %q %w", identifier, err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	gitspaceConfig.BranchURL = c.gitspaceSvc.GetBranchURL(ctx, gitspaceConfig)
	return gitspaceConfig, nil
}

func (c *Controller) createOrFindInfraProviderResource(
	ctx context.Context,
	resourceSpace *types.SpaceCore,
	resourceIdentifier string,
	infraProviderConfigIdentifier string,
	now int64,
) (*types.InfraProviderResource, error) {
	var resource *types.InfraProviderResource
	var err error

	resource, err = c.infraProviderSvc.FindResourceByConfigAndIdentifier(ctx, resourceSpace.ID,
		infraProviderConfigIdentifier, resourceIdentifier)
	if ((err != nil && errors.Is(err, store.ErrResourceNotFound)) || resource == nil) &&
		resourceIdentifier == defaultResourceIdentifier {
		resource, err = c.autoCreateDefaultResource(ctx, resourceSpace, now)
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, fmt.Errorf("could not find infra provider resource : %q %w", resourceIdentifier, err)
	}

	return resource, err
}

func (c *Controller) autoCreateDefaultResource(
	ctx context.Context,
	currentSpace *types.SpaceCore,
	now int64,
) (*types.InfraProviderResource, error) {
	rootSpace, err := c.spaceStore.GetRootSpace(ctx, currentSpace.ID)
	if err != nil {
		return nil, fmt.Errorf("could not get root space for space %s while autocreating default docker "+
			"resource: %w", currentSpace.Path, err)
	}

	defaultDockerConfig := &types.InfraProviderConfig{
		Identifier: defaultResourceIdentifier,
		Name:       "default docker infrastructure",
		Type:       enum.InfraProviderTypeDocker,
		SpaceID:    rootSpace.ID,
		SpacePath:  rootSpace.Path,
		Created:    now,
		Updated:    now,
	}
	defaultResource := types.InfraProviderResource{
		UID:                           defaultResourceIdentifier,
		Name:                          "Standard Docker Resource",
		InfraProviderConfigIdentifier: defaultDockerConfig.Identifier,
		InfraProviderType:             enum.InfraProviderTypeDocker,
		CPU:                           wrapString("any"),
		Memory:                        wrapString("any"),
		Disk:                          wrapString("any"),
		Network:                       wrapString("standard"),
		SpaceID:                       rootSpace.ID,
		SpacePath:                     rootSpace.Path,
		Created:                       now,
		Updated:                       now,
	}
	defaultDockerConfig.Resources = []types.InfraProviderResource{defaultResource}

	err = c.infraProviderSvc.CreateConfigAndResources(ctx, defaultDockerConfig)
	if err != nil {
		return nil, fmt.Errorf("could not auto-create the infra provider: %w", err)
	}

	resource, err := c.infraProviderSvc.FindResourceByConfigAndIdentifier(ctx, rootSpace.ID,
		defaultDockerConfig.Identifier, defaultResourceIdentifier)
	if err != nil {
		return nil, fmt.Errorf("could not find infra provider resource : %q %w", defaultResourceIdentifier, err)
	}

	return resource, nil
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

func buildIdentifier(identifier string) (string, error) {
	const suffixLen = 6

	suffixUID, err := gonanoid.Generate(gitspace.AllowedUIDAlphabet, suffixLen)
	if err != nil {
		return "", fmt.Errorf("could not generate UID for gitspace config: %q %w", identifier, err)
	}

	sanitized := sanitizeIdentifier(strings.ToLower(identifier))
	return sanitized + "-" + suffixUID, nil
}

func sanitizeIdentifier(identifier string) string {
	// Replace invalid characters with hyphen (keep existing hyphens)
	identifier = regexp.MustCompile(`[^a-z0-9-]`).ReplaceAllString(identifier, "-")

	// Trim leading non-letters
	for len(identifier) > 0 && (identifier[0] < 'a' || identifier[0] > 'z') {
		identifier = identifier[1:]
	}

	// Truncate to 50 characters
	if len(identifier) > maxGitspaceConfigIdentifierPrefixLength {
		identifier = identifier[:maxGitspaceConfigIdentifierPrefixLength]
	}

	return identifier
}
