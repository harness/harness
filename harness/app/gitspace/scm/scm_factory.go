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
	"fmt"

	"github.com/harness/gitness/types/enum"
)

type Factory struct {
	listingProviders     map[enum.GitspaceCodeRepoType]ListingProvider
	authAndFileProviders map[enum.GitspaceCodeRepoType]AuthAndFileContentProvider
}

func NewFactoryWithProviders(
	providers map[enum.GitspaceCodeRepoType]ListingProvider,
	authProviders map[enum.GitspaceCodeRepoType]AuthAndFileContentProvider) Factory {
	return Factory{listingProviders: providers,
		authAndFileProviders: authProviders}
}

func NewFactory(gitnessProvider *GitnessSCM, genericSCM *GenericSCM) Factory {
	listingProviders := make(map[enum.GitspaceCodeRepoType]ListingProvider)
	listingProviders[enum.CodeRepoTypeGitness] = gitnessProvider
	listingProviders[enum.CodeRepoTypeUnknown] = genericSCM
	authAndFileContentProviders := make(map[enum.GitspaceCodeRepoType]AuthAndFileContentProvider)
	authAndFileContentProviders[enum.CodeRepoTypeGitness] = gitnessProvider
	authAndFileContentProviders[enum.CodeRepoTypeUnknown] = genericSCM
	return Factory{
		listingProviders:     listingProviders,
		authAndFileProviders: authAndFileContentProviders,
	}
}

func (f *Factory) GetSCMProvider(providerType enum.GitspaceCodeRepoType) (ListingProvider, error) {
	val := f.listingProviders[providerType]
	if val == nil {
		return nil, fmt.Errorf("unknown scm provider type: %s", providerType)
	}
	return val, nil
}

func (f *Factory) GetSCMAuthAndFileProvider(providerType enum.GitspaceCodeRepoType) (
	AuthAndFileContentProvider, error) {
	val := f.authAndFileProviders[providerType]
	if val == nil {
		return nil, fmt.Errorf("unknown scm provider type: %s", providerType)
	}
	return val, nil
}
