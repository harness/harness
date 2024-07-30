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
	"github.com/harness/gitness/app/store"
	urlprovider "github.com/harness/gitness/app/url"
	"github.com/harness/gitness/git"

	"github.com/google/wire"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	ProvideGitnessSCM, ProvideGenericSCM, ProvideFactory, ProvideSCM,
)

func ProvideGitnessSCM(repoStore store.RepoStore,
	rpcClient git.Interface,
	tokenStore store.TokenStore,
	principalStore store.PrincipalStore,
	urlProvider urlprovider.Provider,
) *GitnessSCM {
	return NewGitnessSCM(repoStore, rpcClient, tokenStore, principalStore, urlProvider)
}

func ProvideGenericSCM() *GenericSCM {
	return NewGenericSCM()
}

func ProvideFactory(gitness *GitnessSCM, genericSCM *GenericSCM) Factory {
	return NewFactory(gitness, genericSCM)
}

func ProvideSCM(factory Factory) SCM {
	return NewSCM(factory)
}
