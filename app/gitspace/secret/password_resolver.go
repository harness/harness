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

package secret

import (
	"context"
	"fmt"

	"github.com/harness/gitness/app/gitspace/secret/enum"
	"github.com/harness/gitness/secret"

	"github.com/dchest/uniuri"
)

const defaultPasswordRef = "harness_password"

type PasswordResolver struct {
	secretService secret.Service
}

func NewPasswordResolver(secretService secret.Service) *PasswordResolver {
	return &PasswordResolver{
		secretService: secretService,
	}
}

// Resolve implements Resolver.
func (p *PasswordResolver) Resolve(ctx context.Context, resolutionContext ResolutionContext) (ResolvedSecret, error) {
	if resolutionContext.SecretRef != "" && resolutionContext.SecretRef != defaultPasswordRef {
		if p.secretService == nil {
			return ResolvedSecret{}, fmt.Errorf(
				"secret service unavailable to resolve secret %q",
				resolutionContext.SecretRef,
			)
		}
		val, err := p.secretService.DecryptSecret(ctx, resolutionContext.SpaceIdentifier, resolutionContext.SecretRef)
		if err != nil {
			return ResolvedSecret{}, fmt.Errorf(
				"failed to resolve secret %q for gitspace %q: %w",
				resolutionContext.SecretRef,
				resolutionContext.GitspaceIdentifier,
				err,
			)
		}
		return ResolvedSecret{
			SecretValue: val,
		}, nil
	}

	// If the default password ref is provided and exists in the secret store, use it.
	if resolutionContext.SecretRef == defaultPasswordRef && p.secretService != nil {
		val, err := p.secretService.DecryptSecret(ctx, resolutionContext.SpaceIdentifier, resolutionContext.SecretRef)
		if err == nil && val != "" {
			return ResolvedSecret{
				SecretValue: val,
			}, nil
		}
	}

	// Generate a secure random password per gitspace instance when no specific secret is configured.
	return ResolvedSecret{
		SecretValue: uniuri.NewLen(24),
	}, nil
}

func (p *PasswordResolver) Type() enum.SecretType {
	return enum.PasswordSecretType
}
