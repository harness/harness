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

	"github.com/harness/gitness/app/services/refcache"
	api "github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/secret"

	"github.com/rs/zerolog/log"
)

func GetCredentials(
	ctx context.Context, spaceFinder refcache.SpaceFinder, secretService secret.Service, reg types.UpstreamProxy,
) (accessKey string, secretKey string, isAnonymous bool, err error) {
	if api.AuthType(reg.RepoAuthType) == api.AuthTypeAnonymous {
		return "", "", true, nil
	}
	if api.AuthType(reg.RepoAuthType) == api.AuthTypeUserPassword {
		secretKey, err = getSecretValue(ctx, spaceFinder, secretService, reg.SecretSpaceID,
			reg.SecretIdentifier)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msgf("failed to get secret for registry: %s", reg.RepoKey)
			return "", "", false, fmt.Errorf("failed to get secret for registry: %s", reg.RepoKey)
		}
		return reg.UserName, secretKey, false, nil
	}
	if api.AuthType(reg.RepoAuthType) == api.AuthTypeAccessKeySecretKey {
		accessKey, err = getSecretValue(ctx, spaceFinder, secretService, reg.UserNameSecretSpaceID,
			reg.UserNameSecretIdentifier)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msgf("failed to get access secret for registry: %s", reg.RepoKey)
			return "", "", false, fmt.Errorf("failed to get access key for registry: %s", reg.RepoKey)
		}

		secretKey, err = getSecretValue(ctx, spaceFinder, secretService, reg.SecretSpaceID,
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
	secretSpaceID int64, secretSpacePath string,
) (string, error) {
	spacePath, err := spaceFinder.FindByID(ctx, secretSpaceID)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("failed to find space path: %v", err)
		return "", err
	}
	decryptSecret, err := secretService.DecryptSecret(ctx, spacePath.Path, secretSpacePath)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("failed to decrypt secret: %v", err)
		return "", err
	}
	return decryptSecret, nil
}
