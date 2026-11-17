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

package cargo

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDependencies_RenamedDependency(t *testing.T) {
	manifest, err := parseCargoToml([]byte(`
[package]
name = "app"
version = "0.1.0"

[dependencies.my_serde]
version = "1"
package = "serde"

[dependencies.log]
version = "0.4"
`))
	require.NoError(t, err)

	deps := *parseDependencies(manifest)
	require.Len(t, deps, 2)

	byAlias := map[string]string{}
	for _, d := range deps {
		byAlias[d.ExplicitNameInToml] = d.Name
	}
	// renamed: Name is the real crate, ExplicitNameInToml keeps the alias
	assert.Equal(t, "serde", byAlias["my_serde"])
	// not renamed: both are the same
	assert.Equal(t, "log", byAlias["log"])
}
