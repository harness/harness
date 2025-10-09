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

package url

import (
	"github.com/harness/gitness/types"

	"github.com/google/wire"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(ProvideURLProvider)

func ProvideURLProvider(config *types.Config) (Provider, error) {
	return NewProvider(
		config.URL.Internal,
		config.URL.Container,
		config.URL.API,
		config.URL.Git,
		config.URL.GitSSH,
		config.SSH.DefaultUser,
		config.SSH.Enable,
		config.URL.UI,
		config.URL.Registry,
	)
}
