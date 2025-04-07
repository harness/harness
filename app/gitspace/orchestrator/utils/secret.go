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

package utils

import (
	"context"
	"fmt"

	"github.com/harness/gitness/app/gitspace/secret"
	"github.com/harness/gitness/app/gitspace/secret/enum"
	"github.com/harness/gitness/app/paths"
	"github.com/harness/gitness/types"
	gitnessenum "github.com/harness/gitness/types/enum"
)

func ResolveSecret(ctx context.Context, secretResolverFactory *secret.ResolverFactory, config types.GitspaceConfig) (
	*string,
	error,
) {
	rootSpaceID, _, err := paths.DisectRoot(config.SpacePath)
	if err != nil {
		return nil, fmt.Errorf("unable to find root space id from space path: %s", config.SpacePath)
	}

	secretType := GetSecretType(config.GitspaceInstance.AccessType)
	secretResolver, err := secretResolverFactory.GetSecretResolver(secretType)
	if err != nil {
		return nil, fmt.Errorf("could not find secret resolver for type: %s", config.GitspaceInstance.AccessType)
	}

	resolvedSecret, err := secretResolver.Resolve(ctx, secret.ResolutionContext{
		UserIdentifier:     config.GitspaceUser.Identifier,
		GitspaceIdentifier: config.Identifier,
		SecretRef:          *config.GitspaceInstance.AccessKeyRef,
		SpaceIdentifier:    rootSpaceID,
	})
	if err != nil {
		return nil, fmt.Errorf(
			"could not resolve secret type: %s, ref: %s : %w",
			config.GitspaceInstance.AccessType,
			*config.GitspaceInstance.AccessKeyRef,
			err,
		)
	}
	return &resolvedSecret.SecretValue, nil
}

func GetSecretType(accessType gitnessenum.GitspaceAccessType) enum.SecretType {
	secretType := enum.PasswordSecretType
	switch accessType {
	case gitnessenum.GitspaceAccessTypeUserCredentials:
		secretType = enum.PasswordSecretType
	case gitnessenum.GitspaceAccessTypeJWTToken:
		secretType = enum.JWTSecretType
	case gitnessenum.GitspaceAccessTypeSSHKey:
		secretType = enum.SSHSecretType
	}
	return secretType
}
