//  Copyright 2023 Harness, Inc.
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

package commons

import (
	"context"
	"fmt"

	"github.com/harness/gitness/app/paths"
	"github.com/harness/gitness/app/services/refcache"
	api "github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/secret"
	coretypes "github.com/harness/gitness/types"

	"github.com/rs/zerolog/log"
)

func GetCredentials(
	ctx context.Context, spaceFinder refcache.SpaceFinder, secretService secret.Service, reg types.UpstreamProxy,
) (accessKey string, secretKey string, isAnonymous bool, err error) {
	if api.AuthType(reg.RepoAuthType) == api.AuthTypeAnonymous {
		return "", "", true, nil
	}
	if api.AuthType(reg.RepoAuthType) == api.AuthTypeUserPassword {
		secretKey, err = getSecretValue(ctx, spaceFinder, secretService, reg.ParentID, reg.SecretSpaceID,
			reg.SecretIdentifier)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msgf("failed to get secret for registry: %s", reg.RepoKey)
			return "", "", false, fmt.Errorf("failed to get secret for registry: %s", reg.RepoKey)
		}
		return reg.UserName, secretKey, false, nil
	}
	if api.AuthType(reg.RepoAuthType) == api.AuthTypeAccessKeySecretKey {
		accessKey, err = getSecretValue(ctx, spaceFinder, secretService, reg.ParentID, reg.UserNameSecretSpaceID,
			reg.UserNameSecretIdentifier)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msgf("failed to get access secret for registry: %s", reg.RepoKey)
			return "", "", false, fmt.Errorf("failed to get access key for registry: %s", reg.RepoKey)
		}

		secretKey, err = getSecretValue(ctx, spaceFinder, secretService, reg.ParentID, reg.SecretSpaceID,
			reg.SecretIdentifier)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msgf("failed to get user secret for registry: %s", reg.RepoKey)
			return "", "", false, fmt.Errorf("failed to get secret key for registry: %s", reg.RepoKey)
		}
		return accessKey, secretKey, false, nil
	}
	return "", "", false, fmt.Errorf("unsupported auth type: %s", reg.RepoAuthType)
}

func getSecretValue(
	ctx context.Context, spaceFinder refcache.SpaceFinder, secretService secret.Service,
	registryParentID string, secretSpaceID int64, secretIdentifier string,
) (string, error) {
	if secretIdentifier == "" || secretSpaceID <= 0 {
		return "", nil
	}

	secretSpace, err := spaceFinder.FindByID(ctx, secretSpaceID)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("failed to find space path: %v", err)
		return "", err
	}

	if err := AssertSecretSameAccountAsRegistry(ctx, spaceFinder, registryParentID, secretSpace); err != nil {
		return "", err
	}

	decryptSecret, err := secretService.DecryptSecret(ctx, secretSpace.Path, secretIdentifier)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("failed to decrypt secret: %v", err)
		return "", err
	}
	return decryptSecret, nil
}

// spaceResolver is the subset of SpaceFinder needed to resolve the registry parent space.
type spaceResolver interface {
	FindByRef(ctx context.Context, ref string) (*coretypes.SpaceCore, error)
}

// AssertSecretSameAccountAsRegistry returns an error when secretSpace is not under the same
// account root as the registry parent identified by registryParentID.
func AssertSecretSameAccountAsRegistry(
	ctx context.Context, finder spaceResolver,
	registryParentID string, secretSpace *coretypes.SpaceCore,
) error {
	if secretSpace == nil {
		return fmt.Errorf("secret space is required for ownership check")
	}
	if registryParentID == "" {
		return fmt.Errorf("registry parent space is required for secret ownership check")
	}

	registrySpace, err := finder.FindByRef(ctx, registryParentID)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf(
			"failed to resolve registry parent space %s for secret ownership check", registryParentID)
		return fmt.Errorf("failed to resolve registry parent space for secret ownership check: %w", err)
	}

	same, err := SameAccount(registrySpace.Path, secretSpace.Path)
	if err != nil {
		return err
	}
	if !same {
		log.Ctx(ctx).Error().Msgf(
			"refusing to decrypt secret in space %q for registry parent %q: cross-account binding",
			secretSpace.Path, registrySpace.Path)
		return fmt.Errorf(
			"secret space %q is outside registry account %q", secretSpace.Path, registrySpace.Path)
	}
	return nil
}

// SameAccount reports whether two space paths share the same account root
// (first path segment). Empty paths are rejected.
func SameAccount(registrySpacePath, secretSpacePath string) (bool, error) {
	registryAccount, _, err := paths.DisectRoot(registrySpacePath)
	if err != nil {
		return false, fmt.Errorf("invalid registry space path %q: %w", registrySpacePath, err)
	}
	secretAccount, _, err := paths.DisectRoot(secretSpacePath)
	if err != nil {
		return false, fmt.Errorf("invalid secret space path %q: %w", secretSpacePath, err)
	}
	return registryAccount == secretAccount, nil
}
