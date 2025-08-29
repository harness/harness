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
	"fmt"
	"strconv"

	"github.com/harness/gitness/app/paths"
	"github.com/harness/gitness/store"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

func (c *Service) FindWithLatestInstanceWithSpacePath(
	ctx context.Context,
	spacePath string,
	identifier string,
) (*types.GitspaceConfig, error) {
	space, err := c.spaceFinder.FindByRef(ctx, spacePath)
	if err != nil {
		return nil, fmt.Errorf("failed to find space: %w", err)
	}
	return c.FindWithLatestInstance(ctx, space.ID, spacePath, identifier)
}

func (c *Service) FindWithLatestInstance(
	ctx context.Context,
	spaceID int64,
	spacePath string,
	identifier string,
) (*types.GitspaceConfig, error) {
	if spacePath == "" {
		space, err := c.spaceFinder.FindByID(ctx, spaceID)
		if err != nil {
			log.Warn().Err(err).Msgf("failed to find space path for id %d", spaceID)
		} else {
			spacePath = space.Path
		}
	}

	var gitspaceConfigResult *types.GitspaceConfig
	txErr := c.tx.WithTx(ctx, func(ctx context.Context) error {
		gitspaceConfig, err := c.gitspaceConfigStore.FindByIdentifier(ctx, spaceID, identifier)
		if err != nil {
			return fmt.Errorf("failed to find gitspace config: %w", err)
		}
		gitspaceConfig.SpacePath = spacePath
		latestInstance, err := c.findLatestInstance(ctx, gitspaceConfig)
		if err != nil {
			return err
		}

		configState, err := getGitspaceConfigState(latestInstance)
		if err != nil {
			return err
		}
		// update gitspace config parameters based on latest instance
		gitspaceConfig.GitspaceInstance = latestInstance
		gitspaceConfig.State = configState
		// store result in return variable
		gitspaceConfigResult = gitspaceConfig
		return nil
	}, dbtx.TxDefaultReadOnly)
	if txErr != nil {
		return nil, txErr
	}

	gitspaceConfigResult.BranchURL = c.GetBranchURL(ctx, gitspaceConfigResult)
	return gitspaceConfigResult, nil
}

// findLatestInstance return latest gitspace instance for given gitspace config.
// If no instance is found, it returns nil.
func (c *Service) findLatestInstance(
	ctx context.Context,
	gitspaceConfig *types.GitspaceConfig,
) (*types.GitspaceInstance, error) {
	instance, err := c.gitspaceInstanceStore.FindLatestByGitspaceConfigID(ctx, gitspaceConfig.ID)
	if err != nil && !errors.Is(err, store.ErrResourceNotFound) {
		return nil, err
	}

	if errors.Is(err, store.ErrResourceNotFound) {
		// nolint:nilnil // return value is based on no resource
		return nil, nil
	}

	// add or update various parameters of gitspace instance.
	return c.addOrUpdateInstanceParameters(ctx, instance, gitspaceConfig)
}

func (c *Service) getToken(
	ctx context.Context,
	gitspaceConfig *types.GitspaceConfig,
) (string, error) {
	if gitspaceConfig.IDE != enum.IDETypeVSCodeWeb {
		return "", nil
	}

	resourceSpace, err := c.spaceStore.FindByRef(ctx, gitspaceConfig.InfraProviderResource.SpacePath)
	if err != nil || resourceSpace == nil {
		return "", fmt.Errorf("failed to find space ref: %w", err)
	}
	infraProviderConfigIdentifier := gitspaceConfig.InfraProviderResource.InfraProviderConfigIdentifier
	infraProviderConfig, err := c.infraProviderSvc.Find(ctx, resourceSpace.Core(), infraProviderConfigIdentifier)
	if err != nil {
		log.Warn().Msgf(
			"Cannot get infraProviderConfig for resource : %s/%s",
			resourceSpace.Path, infraProviderConfigIdentifier)
		return "", err
	}

	return c.tokenGenerator.GenerateToken(
		ctx,
		gitspaceConfig,
		gitspaceConfig.GitspaceUser.Identifier,
		enum.PrincipalTypeUser,
		infraProviderConfig,
	)
}

func getProjectName(spacePath string) string {
	_, projectName, err := paths.DisectLeaf(spacePath)
	if err != nil {
		return ""
	}

	return projectName
}

func getGitspaceConfigState(instance *types.GitspaceInstance) (enum.GitspaceStateType, error) {
	if instance == nil {
		return enum.GitspaceStateUninitialized, nil
	}

	return instance.GetGitspaceState()
}

func (c *Service) addOrUpdateInstanceParameters(
	ctx context.Context,
	instance *types.GitspaceInstance,
	gitspaceConfig *types.GitspaceConfig,
) (*types.GitspaceInstance, error) {
	if instance == nil || gitspaceConfig == nil {
		// nolint:nilnil // return value is based on nil pointers
		return nil, nil
	}
	// add or update various parameters of gitspace instance.
	instance.SpacePath = gitspaceConfig.SpacePath

	ideSvc, err := c.ideFactory.GetIDE(gitspaceConfig.IDE)
	if err != nil {
		return nil, err
	}

	projectName := getProjectName(gitspaceConfig.SpacePath)
	pluginURL := ideSvc.GeneratePluginURL(projectName, instance.Identifier)
	if pluginURL != "" {
		instance.PluginURL = &pluginURL
	}

	if instance.URL != nil && gitspaceConfig.IDE == enum.IDETypeVSCodeWeb {
		// token is jwt token issue by cde-manager which is validated in cde-gateway when accessing vscode web.
		gitspaceConfig.GitspaceInstance = instance
		token, err := c.getToken(ctx, gitspaceConfig)
		if err != nil {
			return nil, fmt.Errorf("unable to generate JWT token for vscode web: %w", err)
		}

		if token != "" {
			urlWithToken := fmt.Sprintf("%s&token=%s", *instance.URL, token)
			instance.URL = &urlWithToken
		}
	}

	return instance, nil
}

func (c *Service) FindWithLatestInstanceByID(
	ctx context.Context,
	id int64,
	includeDeleted bool,
) (*types.GitspaceConfig, error) {
	var gitspaceConfigResult *types.GitspaceConfig
	txErr := c.tx.WithTx(ctx, func(ctx context.Context) error {
		gitspaceConfig, err := c.gitspaceConfigStore.Find(ctx, id, includeDeleted)
		if err != nil {
			return fmt.Errorf("failed to find gitspace config: %w", err)
		}

		space, err := c.spaceFinder.FindByID(ctx, gitspaceConfig.SpaceID)
		if err != nil {
			return fmt.Errorf("failed to find space: %w", err)
		}
		gitspaceConfig.SpacePath = space.Path

		latestInstance, err := c.findLatestInstance(ctx, gitspaceConfig)
		if err != nil {
			return err
		}

		configState, err := getGitspaceConfigState(latestInstance)
		if err != nil {
			return err
		}
		// update gitspace config parameters based on latest instance
		gitspaceConfig.GitspaceInstance = latestInstance
		gitspaceConfig.State = configState
		// store result in return variable
		gitspaceConfigResult = gitspaceConfig

		return nil
	}, dbtx.TxDefaultReadOnly)
	if txErr != nil {
		return nil, txErr
	}
	return gitspaceConfigResult, nil
}

func (c *Service) FindAll(
	ctx context.Context,
	ids []int64,
) ([]*types.GitspaceConfig, error) {
	var gitspaceConfigResult []*types.GitspaceConfig
	txErr := c.tx.WithTx(ctx, func(ctx context.Context) error {
		// TODO join and set gitspace instance, space from cache
		gitspaceConfigs, err := c.gitspaceConfigStore.FindAll(ctx, ids)
		if err != nil {
			return fmt.Errorf("failed to find gitspace config: %w", err)
		}
		for _, gitspaceConfig := range gitspaceConfigs {
			// FindByRef method is backed by cache as opposed to Find
			space, err := c.spaceFinder.FindByRef(ctx, strconv.FormatInt(gitspaceConfig.SpaceID, 10))
			if err != nil {
				return fmt.Errorf("failed to find space: %w", err)
			}
			gitspaceConfig.SpacePath = space.Path
			gitspaceConfigResult = append(gitspaceConfigResult, gitspaceConfig)
		}
		return nil
	}, dbtx.TxDefaultReadOnly)
	if txErr != nil {
		return nil, txErr
	}
	return gitspaceConfigResult, nil
}

func (c *Service) FindAllByIdentifier(
	ctx context.Context,
	spaceID int64,
	identifiers []string,
) ([]types.GitspaceConfig, error) {
	var gitspaceConfigResult []types.GitspaceConfig
	txErr := c.tx.WithTx(ctx, func(ctx context.Context) error {
		gitspaceConfigs, err := c.gitspaceConfigStore.FindAllByIdentifier(ctx, spaceID, identifiers)
		if err != nil {
			return fmt.Errorf("failed to find gitspace config: %w", err)
		}

		gitspaceConfigResult = gitspaceConfigs
		return nil
	}, dbtx.TxDefaultReadOnly)
	if txErr != nil {
		return nil, txErr
	}
	return gitspaceConfigResult, nil
}

func (c *Service) FindInstanceByIdentifier(
	ctx context.Context,
	identifier string,
) (*types.GitspaceInstance, error) {
	var gitspaceInstanceResult *types.GitspaceInstance
	txErr := c.tx.WithTx(ctx, func(ctx context.Context) error {
		gitspaceInstance, err := c.gitspaceInstanceStore.FindByIdentifier(ctx, identifier)
		if err != nil {
			return fmt.Errorf("failed to find gitspace instance: %w", err)
		}
		gitspaceInstanceResult = gitspaceInstance
		return nil
	}, dbtx.TxDefaultReadOnly)
	if txErr != nil {
		return nil, txErr
	}

	gitspaceConfig, err := c.gitspaceConfigStore.Find(ctx, gitspaceInstanceResult.GitSpaceConfigID, true)
	if err != nil {
		return nil, fmt.Errorf("could not find gitspace config: %w", err)
	}

	return c.addOrUpdateInstanceParameters(ctx, gitspaceInstanceResult, gitspaceConfig)
}
