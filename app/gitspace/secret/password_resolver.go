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
	"errors"
	"fmt"

	"github.com/harness/gitness/app/gitspace/secret/enum"
	"github.com/harness/gitness/secret"
	"github.com/harness/gitness/store"
)

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
	val, err := p.secretService.DecryptSecret(ctx, resolutionContext.SpaceIdentifier, resolutionContext.SecretRef)
	if errors.Is(err, store.ErrResourceNotFound) {
		return ResolvedSecret{}, fmt.Errorf(
			"gitspace %q has no SSH key and secret %q does not exist in space %q, configure one of them: %w",
			resolutionContext.GitspaceIdentifier,
			resolutionContext.SecretRef,
			resolutionContext.SpaceIdentifier,
			err,
		)
	}
	if err != nil {
		return ResolvedSecret{}, fmt.Errorf(
			"failed to resolve secret %q for gitspace %q: %w",
			resolutionContext.SecretRef,
			resolutionContext.GitspaceIdentifier,
			err,
		)
	}
	// An empty password combined with PermitEmptyPasswords in the gitspace sshd config would allow open login.
	if val == "" {
		return ResolvedSecret{}, fmt.Errorf("secret %q for gitspace %q is empty",
			resolutionContext.SecretRef, resolutionContext.GitspaceIdentifier)
	}
	return ResolvedSecret{
		SecretValue: val,
	}, nil
}

func (p *PasswordResolver) Type() enum.SecretType {
	return enum.PasswordSecretType
}
