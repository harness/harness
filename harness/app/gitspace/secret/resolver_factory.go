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
	"fmt"

	"github.com/harness/gitness/app/gitspace/secret/enum"
)

type ResolverFactory struct {
	resolvers map[enum.SecretType]Resolver
}

func NewFactoryWithProviders(resolvers ...Resolver) *ResolverFactory {
	resolversMap := make(map[enum.SecretType]Resolver)
	for _, r := range resolvers {
		resolversMap[r.Type()] = r
	}
	return &ResolverFactory{resolvers: resolversMap}
}

func (f *ResolverFactory) GetSecretResolver(secretType enum.SecretType) (Resolver, error) {
	val := f.resolvers[secretType]
	if val == nil {
		return nil, fmt.Errorf("unknown secret type: %s", secretType)
	}
	return val, nil
}
