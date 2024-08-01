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

package scm

import (
	"context"
	"fmt"

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type Provider interface {
	ResolveCredentials(ctx context.Context, gitspaceConfig types.GitspaceConfig) (*ResolvedCredentials, error)
	GetFileContent(
		ctx context.Context,
		gitspaceConfig types.GitspaceConfig,
		filePath string,
	) ([]byte, error)
}

type Factory struct {
	providers map[enum.GitspaceCodeRepoType]Provider
}

func NewFactoryWithProviders(providers map[enum.GitspaceCodeRepoType]Provider) Factory {
	return Factory{providers: providers}
}

func NewFactory(gitnessProvider *GitnessSCM, genericSCM *GenericSCM) Factory {
	providers := make(map[enum.GitspaceCodeRepoType]Provider)
	providers[enum.CodeRepoTypeGitness] = gitnessProvider
	providers[enum.CodeRepoTypeUnknown] = genericSCM
	return Factory{providers: providers}
}

func (f *Factory) GetSCMProvider(providerType enum.GitspaceCodeRepoType) (Provider, error) {
	val := f.providers[providerType]
	if val == nil {
		return nil, fmt.Errorf("unknown scm provider type: %s", providerType)
	}
	return val, nil
}
