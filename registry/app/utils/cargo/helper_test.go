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
	"encoding/json"
	"testing"

	cargometadata "github.com/harness/gitness/registry/app/metadata/cargo"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMapVersionDependenciesToIndexDependencies_Rename(t *testing.T) {
	h := &registryHelper{}

	tests := []struct {
		name string
		dep  cargometadata.VersionDependency
		want map[string]any
	}{
		{
			name: "renamed dependency",
			dep: cargometadata.VersionDependency{
				Dependency: cargometadata.Dependency{
					Name:               "serde",
					ExplicitNameInToml: "my_serde",
					Kind:               cargometadata.DependencyKindTypeNormal,
				},
				VersionRequired: "^1",
			},
			want: map[string]any{"name": "my_serde", "package": "serde", "req": "^1", "kind": "normal"},
		},
		{
			name: "explicit name equal to name",
			dep: cargometadata.VersionDependency{
				Dependency: cargometadata.Dependency{
					Name:               "serde",
					ExplicitNameInToml: "serde",
					Kind:               cargometadata.DependencyKindTypeNormal,
				},
				VersionRequired: "^1",
			},
			want: map[string]any{"name": "serde", "req": "^1", "kind": "normal"},
		},
		{
			name: "no explicit name",
			dep: cargometadata.VersionDependency{
				Dependency: cargometadata.Dependency{
					Name: "serde",
					Kind: cargometadata.DependencyKindTypeNormal,
				},
				VersionRequired: "^1",
			},
			want: map[string]any{"name": "serde", "req": "^1", "kind": "normal"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := h.mapVersionDependenciesToIndexDependencies(
				[]cargometadata.VersionDependency{tt.dep},
			)
			require.Len(t, deps, 1)

			raw, err := json.Marshal(deps[0])
			require.NoError(t, err)
			var got map[string]any
			require.NoError(t, json.Unmarshal(raw, &got))

			assert.Equal(t, tt.want, got)
			assert.NotContains(t, got, "explicit_name_in_toml")
		})
	}
}
