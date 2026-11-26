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

package types

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

func TestGc5LifecycleCommandUnmarshal(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		discriminator string
		wantErr       bool
		check         func(t *testing.T, lc LifecycleCommand)
	}{
		{
			name: "string", input: `"npm install"`, discriminator: TypeString,
			check: func(t *testing.T, lc LifecycleCommand) { require.Equal(t, "npm install", lc.CommandString) },
		},
		{
			name: "array", input: `["npm","install"]`, discriminator: TypeArray,
			check: func(t *testing.T, lc LifecycleCommand) { require.Equal(t, []string{"npm", "install"}, lc.CommandArray) },
		},
		{
			name: "map of strings and arrays", input: `{"a":"echo a","b":["echo","b"]}`, discriminator: TypeCommandMap,
			check: func(t *testing.T, lc LifecycleCommand) {
				require.Equal(t, "echo a", lc.CommandMap["a"])
				require.Equal(t, []string{"echo", "b"}, lc.CommandMap["b"])
			},
		},
		{name: "map with non string array item", input: `{"a":["x",1]}`, wantErr: true},
		{name: "map with unsupported value", input: `{"a":1}`, wantErr: true},
		{name: "number", input: `12`, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var lc LifecycleCommand
			err := json.Unmarshal([]byte(tt.input), &lc)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.discriminator, lc.Discriminator)
			tt.check(t, lc)

			// marshaling restores the original shape.
			raw, err := json.Marshal(&lc)
			require.NoError(t, err)
			require.JSONEq(t, tt.input, string(raw))
		})
	}
}

func TestGc5LifecycleCommandMarshalErrors(t *testing.T) {
	raw, err := (&LifecycleCommand{}).MarshalJSON()
	require.NoError(t, err)
	require.Equal(t, "{}", string(raw))

	_, err = (&LifecycleCommand{Discriminator: "other"}).MarshalJSON()
	require.Error(t, err)
}

func TestGc5LifecycleCommandToCommandArray(t *testing.T) {
	require.Nil(t, (&LifecycleCommand{}).ToCommandArray())
	require.Nil(t, (&LifecycleCommand{Discriminator: "other"}).ToCommandArray())
	require.Equal(t, []string{"ls"}, (&LifecycleCommand{Discriminator: TypeString, CommandString: "ls"}).ToCommandArray())
	require.Equal(t, []string{"ls -la"},
		(&LifecycleCommand{Discriminator: TypeArray, CommandArray: []string{"ls", "-la"}}).ToCommandArray())

	lc := &LifecycleCommand{
		Discriminator: TypeCommandMap,
		CommandMap:    map[string]any{"a": "echo a", "b": []string{"echo", "b"}, "c": 5},
	}
	require.ElementsMatch(t, []string{"echo a", "echo b"}, lc.ToCommandArray())
}

func TestGc5FeaturesUnmarshal(t *testing.T) {
	input := `{
		"ghcr.io/devcontainers/features/go": {"version": "1.22", "flag": true},
		"ghcr.io/devcontainers/features/node:1": "20",
		"https://example.com/features/tool.tgz": {}
	}`

	var cfg DevcontainerConfig
	require.NoError(t, json.Unmarshal([]byte(`{"features":`+input+`}`), &cfg))
	require.NotNil(t, cfg.Features)

	features := *cfg.Features
	goFeature := features["ghcr.io/devcontainers/features/go:"+FeatureDefaultTag]
	require.NotNil(t, goFeature)
	require.EqualValues(t, enum.FeatureSourceTypeOCI, goFeature.SourceType)
	require.Equal(t, "1.22", goFeature.Options["version"])
	require.Equal(t, true, goFeature.Options["flag"])

	nodeFeature := features["ghcr.io/devcontainers/features/node:1"]
	require.NotNil(t, nodeFeature)
	require.Equal(t, "20", nodeFeature.Options["version"])

	tarball := features["https://example.com/features/tool.tgz"]
	require.NotNil(t, tarball)
	require.EqualValues(t, enum.FeatureSourceTypeTarball, tarball.SourceType)

	raw, err := json.Marshal(nodeFeature)
	require.NoError(t, err)
	require.JSONEq(t, `{"version":"20"}`, string(raw))
}

func TestGc5FeaturesUnmarshalErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"not an object", `[1]`},
		{"unsupported source", `{"not-a-feature": {}}`},
		{"invalid option type", `{"ghcr.io/devcontainers/features/go": {"version": 1}}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var f Features
			require.Error(t, json.Unmarshal([]byte(tt.input), &f))
		})
	}

	// a feature value that is neither string nor object is silently ignored.
	var f Features
	require.NoError(t, json.Unmarshal([]byte(`{"ghcr.io/devcontainers/features/go": 5}`), &f))
	require.Nil(t, f["ghcr.io/devcontainers/features/go:"+FeatureDefaultTag].Options)
}

func TestGc5ValidateFeatureSource(t *testing.T) {
	src, typ, err := validateFeatureSource("ghcr.io/org/feature")
	require.NoError(t, err)
	require.Equal(t, "ghcr.io/org/feature:latest", src)
	require.EqualValues(t, enum.FeatureSourceTypeOCI, typ)

	src, typ, err = validateFeatureSource("https://example.com/f.tgz")
	require.NoError(t, err)
	require.Equal(t, "https://example.com/f.tgz", src)
	require.EqualValues(t, enum.FeatureSourceTypeTarball, typ)

	_, _, err = validateFeatureSource("not a valid source")
	require.Error(t, err)

	require.Error(t, validateTarballURL("ftp://example.com/f.tgz"))
	require.Error(t, validateTarballURL("https://example.com/f.zip"))
	require.Error(t, validateTarballURL("http://[::1"))
	require.NoError(t, validateTarballURL("http://example.com/f.tgz"))
}

func TestGc5MountUnmarshal(t *testing.T) {
	var m Mount
	require.NoError(t, json.Unmarshal([]byte(`{"source":"vol","target":"/data","type":"volume"}`), &m))
	require.Equal(t, Mount{Source: "vol", Target: "/data", Type: "volume"}, m)

	var fromString Mount
	require.NoError(t, json.Unmarshal([]byte(`"type=bind,src=/host,dst=/guest"`), &fromString))
	require.Equal(t, Mount{Source: "/host", Target: "/guest", Type: "bind"}, fromString)

	var invalid Mount
	require.Error(t, json.Unmarshal([]byte(`"type=bind,broken"`), &invalid))
	require.Error(t, json.Unmarshal([]byte(`12`), &invalid))
}

func TestGc5StringToMount(t *testing.T) {
	abs, err := filepath.Abs(".")
	require.NoError(t, err)

	tests := []struct {
		in      string
		want    Mount
		wantErr bool
	}{
		{in: "source=a,target=/b", want: Mount{Type: "volume", Source: "a", Target: "/b"}},
		{in: "TYPE=Bind,Src=/x,Destination=/y", want: Mount{Type: "bind", Source: "/x", Target: "/y"}},
		{in: "type=bind,source=.,target=/y", want: Mount{Type: "bind", Source: abs, Target: "/y"}},
		{in: "source=a,unknown=b", want: Mount{Type: "volume", Source: "a"}},
		{in: "justakey", wantErr: true},
		{in: `"unterminated`, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := stringToObject(tt.in)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, *got)
		})
	}
}

func TestGc5ParseMounts(t *testing.T) {
	existing := &Mount{Source: "s", Target: "t", Type: "bind"}

	mounts, err := ParseMountsFromRawSlice([]any{
		existing,
		map[string]any{"source": "ms", "target": "mt", "type": "volume"},
		map[string]any{"source": 5},
		"src=a,dst=/b",
	})
	require.NoError(t, err)
	require.Len(t, mounts, 4)
	require.Same(t, existing, mounts[0])
	require.Equal(t, Mount{Source: "ms", Target: "mt", Type: "volume"}, *mounts[1])
	require.Equal(t, Mount{}, *mounts[2])
	require.Equal(t, Mount{Type: "volume", Source: "a", Target: "/b"}, *mounts[3])

	_, err = ParseMountsFromRawSlice([]any{5})
	require.Error(t, err)
	_, err = ParseMountsFromRawSlice([]any{"broken"})
	require.Error(t, err)

	fromStrings, err := ParseMountsFromStringSlice([]string{"src=a,dst=/b", "type=tmpfs,dst=/tmp"})
	require.NoError(t, err)
	require.Len(t, fromStrings, 2)
	require.Equal(t, "tmpfs", fromStrings[1].Type)

	_, err = ParseMountsFromStringSlice([]string{"broken"})
	require.Error(t, err)
}

func TestGc5CustomizationsExtract(t *testing.T) {
	var cfg DevcontainerConfig
	require.NoError(t, json.Unmarshal([]byte(`{
		"customizations": {
			"vscode": {"extensions": ["ext.one"], "settings": {"a": 1}},
			"jetbrains": {"backend": "Goland", "plugins": ["p1"]},
			"harnessGitspaces": {
				"connectors": [{"type": "DockerRegistry", "identifier": "reg"}],
				"ai_agent": {"type": "claude-code", "auth": "API-Key", "secret_ref": "ref"}
			}
		}
	}`), &cfg))

	vscode := cfg.Customizations.ExtractVSCodeSpec()
	require.NotNil(t, vscode)
	require.Equal(t, []string{"ext.one"}, vscode.Extensions)

	jb := cfg.Customizations.ExtractJetBrainsSpecs()
	require.NotNil(t, jb)
	require.Equal(t, GolandJetBrainsBackend, jb.Backend)
	require.True(t, jb.Backend.Valid())

	gs := cfg.Customizations.ExtractGitspaceSpec()
	require.NotNil(t, gs)
	require.Len(t, gs.Connectors, 1)
	require.Equal(t, "reg", gs.Connectors[0].ID)
	require.Equal(t, "claude-code", gs.AIAgent.Type)
	require.Equal(t, "ref", gs.AIAgent.SecretRef)
}

func TestGc5CustomizationsExtractMissingOrInvalid(t *testing.T) {
	empty := DevContainerConfigCustomizations{}
	require.Nil(t, empty.ExtractGitspaceSpec())
	require.Nil(t, empty.ExtractVSCodeSpec())
	require.Nil(t, empty.ExtractJetBrainsSpecs())

	invalid := DevContainerConfigCustomizations{
		"harnessGitspaces": "not an object",
		"vscode":           "not an object",
		"jetbrains":        "not an object",
	}
	require.Nil(t, invalid.ExtractGitspaceSpec())
	require.Nil(t, invalid.ExtractVSCodeSpec())
	require.Nil(t, invalid.ExtractJetBrainsSpecs())

	badFields := DevContainerConfigCustomizations{
		"vscode": map[string]any{"extensions": "not a list"},
	}
	require.Nil(t, badFields.ExtractVSCodeSpec())

	unmarshalable := DevContainerConfigCustomizations{
		"vscode":    map[string]any{"settings": make(chan int)},
		"jetbrains": make(chan int),
	}
	require.Nil(t, unmarshalable.ExtractVSCodeSpec())
	require.Nil(t, unmarshalable.ExtractJetBrainsSpecs())
}

func TestGc5JetBrainsBackend(t *testing.T) {
	tests := []struct {
		backend JetBrainsBackend
		ide     enum.IDEType
	}{
		{IntelliJJetBrainsBackend, enum.IDETypeIntelliJ},
		{GolandJetBrainsBackend, enum.IDETypeGoland},
		{PyCharmJetBrainsBackend, enum.IDETypePyCharm},
		{WebStormJetBrainsBackend, enum.IDETypeWebStorm},
		{CLionJetBrainsBackend, enum.IDETypeCLion},
		{PhpStormJetBrainsBackend, enum.IDETypePHPStorm},
		{RubyMineJetBrainsBackend, enum.IDETypeRubyMine},
		{RiderJetBrainsBackend, enum.IDETypeRider},
	}
	for _, tt := range tests {
		t.Run(tt.backend.String(), func(t *testing.T) {
			require.True(t, tt.backend.Valid())
			require.Equal(t, tt.ide, tt.backend.IdeType())
		})
	}

	require.False(t, JetBrainsBackend("Unknown").Valid())
	require.Empty(t, JetBrainsBackend("Unknown").IdeType())
	require.Equal(t, "vscode", VSCodeCustomizationsKey.String())
}
