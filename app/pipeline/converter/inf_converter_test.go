// Copyright 2026 Harness, Inc.
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

package converter

import (
	"context"
	"errors"
	"testing"

	"github.com/harness/gitness/app/pipeline/file"
	"github.com/harness/gitness/app/services/publicaccess"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

type infPublicAccess struct {
	publicaccess.Service

	public bool
	err    error
	gotRes string
}

func (p *infPublicAccess) Get(_ context.Context, _ enum.PublicResourceType, path string) (bool, error) {
	p.gotRes = path
	return p.public, p.err
}

type infFileService struct {
	files map[string]string
}

func (s infFileService) Get(_ context.Context, _ *types.Repository, path, _ string) (*file.File, error) {
	data, ok := s.files[path]
	if !ok {
		return nil, errors.New("not found")
	}
	return &file.File{Data: []byte(data)}, nil
}

func infArgs(configPath, content string) *ConvertArgs {
	return &ConvertArgs{
		Repo:      &types.Repository{Identifier: "r", Path: "space/r"},
		Pipeline:  &types.Pipeline{ConfigPath: configPath},
		Execution: &types.Execution{Ref: "refs/heads/main"},
		File:      &file.File{Data: []byte(content)},
	}
}

func TestInfConverter_Convert(t *testing.T) {
	ctx := context.Background()
	fs := infFileService{files: map[string]string{"lib.libsonnet": `{ from: "lib" }`}}
	starlarkScript := "def main(ctx):\n    return {\"repo\": ctx.repo.name}\n"

	tests := []struct {
		name    string
		path    string
		content string
		want    string
		wantErr bool
	}{
		{
			name:    "plain yaml is passed through",
			path:    ".harness/pipeline.yaml",
			content: "stages: []",
			want:    "stages: []",
		},
		{
			name:    "jsonnet",
			path:    ".drone.jsonnet",
			content: `local lib = import "lib.libsonnet"; { name: std.extVar("repo.identifier"), from: lib.from }`,
			want:    "---\n{\n   \"from\": \"lib\",\n   \"name\": \"r\"\n}\n",
		},
		{name: "jsonnet error", path: ".drone.jsonnet", content: "{ broken", wantErr: true},
		{name: "starlark .star", path: ".drone.star", content: starlarkScript, want: `{"repo": "r"}`},
		{name: "starlark .script", path: "ci/.drone.script", content: starlarkScript, want: `{"repo": "r"}`},
		{name: "starlark .starlark", path: ".drone.starlark", content: starlarkScript, want: `{"repo": "r"}`},
		{name: "starlark error", path: ".drone.star", content: "x = 1", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pa := &infPublicAccess{public: true}
			svc := newConverter(fs, pa)
			got, err := svc.Convert(ctx, infArgs(tt.path, tt.content))
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, string(got.Data))
			require.Equal(t, "space/r", pa.gotRes)
		})
	}
}

func TestInfConverter_PublicAccessErrorIsReturned(t *testing.T) {
	svc := newConverter(infFileService{}, &infPublicAccess{err: errors.New("lookup failed")})
	_, err := svc.Convert(context.Background(), infArgs(".drone.star", ""))
	require.ErrorContains(t, err, "lookup failed")
}

func TestInfPathMatchers(t *testing.T) {
	require.True(t, isJSONNet("a/.drone.jsonnet"))
	require.False(t, isJSONNet(".drone.yml"))
	for _, p := range []string{".drone.script", ".drone.star", "x/.drone.starlark"} {
		require.True(t, isStarlark(p), p)
	}
	require.False(t, isStarlark(".drone.jsonnet"))
}
