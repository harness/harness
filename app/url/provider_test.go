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
	"context"
	"net/url"
	"testing"
)

func TestBuildGITCloneSSHURL(t *testing.T) {
	tests := []struct {
		name     string
		user     string
		sshURL   string
		repoPath string
		want     string
	}{
		{
			name:     "standard port 22",
			user:     "git",
			sshURL:   "ssh://git.example.com:22",
			repoPath: "org/repo",
			want:     "git@git.example.com:org/repo.git",
		},
		{
			name:     "default port (empty)",
			user:     "git",
			sshURL:   "ssh://git.example.com",
			repoPath: "org/repo",
			want:     "git@git.example.com:org/repo.git",
		},
		{
			name:     "custom port",
			user:     "git",
			sshURL:   "ssh://git.example.com:2222",
			repoPath: "org/repo",
			want:     "ssh://git@git.example.com:2222/org/repo.git",
		},
		{
			name:     "repo path with .git suffix",
			user:     "git",
			sshURL:   "ssh://git.example.com",
			repoPath: "org/repo.git",
			want:     "git@git.example.com:org/repo.git",
		},
		{
			name:     "repo path with leading slash",
			user:     "git",
			sshURL:   "ssh://git.example.com",
			repoPath: "/org/repo",
			want:     "git@git.example.com:org/repo.git",
		},
		{
			name:     "repo path with trailing slash",
			user:     "git",
			sshURL:   "ssh://git.example.com",
			repoPath: "org/repo/",
			want:     "git@git.example.com:org/repo.git",
		},
		{
			name:     "custom port with path in URL",
			user:     "git",
			sshURL:   "ssh://git.example.com:2222/base",
			repoPath: "org/repo",
			want:     "ssh://git@git.example.com:2222/base/org/repo.git",
		},
		{
			name:     "standard port with path in URL",
			user:     "git",
			sshURL:   "ssh://git.example.com:22/base",
			repoPath: "org/repo",
			want:     "git@git.example.com:base/org/repo.git",
		},
		{
			name:     "port 0 treated as default",
			user:     "git",
			sshURL:   "ssh://git.example.com:0",
			repoPath: "org/repo",
			want:     "git@git.example.com:org/repo.git",
		},
		{
			name:     "different username",
			user:     "admin",
			sshURL:   "ssh://git.example.com",
			repoPath: "org/repo",
			want:     "admin@git.example.com:org/repo.git",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sshURL, err := url.Parse(tt.sshURL)
			if err != nil {
				t.Fatalf("failed to parse sshURL: %v", err)
			}

			got := BuildGITCloneSSHURL(tt.user, sshURL, tt.repoPath)
			if got != tt.want {
				t.Errorf("BuildGITCloneSSHURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewProvider(t *testing.T) {
	tests := []struct {
		name            string
		internalURLRaw  string
		containerURLRaw string
		apiURLRaw       string
		gitURLRaw       string
		gitSSHURLRaw    string
		sshDefaultUser  string
		sshEnabled      bool
		uiURLRaw        string
		registryURLRaw  string
		wantErr         bool
	}{
		{
			name:            "valid URLs",
			internalURLRaw:  "http://internal.example.com",
			containerURLRaw: "http://container.example.com",
			apiURLRaw:       "http://api.example.com",
			gitURLRaw:       "http://git.example.com",
			gitSSHURLRaw:    "ssh://git.example.com:22",
			sshDefaultUser:  "git",
			sshEnabled:      true,
			uiURLRaw:        "http://ui.example.com",
			registryURLRaw:  "http://registry.example.com",
			wantErr:         false,
		},
		{
			name:            "URLs with trailing slashes",
			internalURLRaw:  "http://internal.example.com/",
			containerURLRaw: "http://container.example.com/",
			apiURLRaw:       "http://api.example.com/",
			gitURLRaw:       "http://git.example.com/",
			gitSSHURLRaw:    "ssh://git.example.com:22/",
			sshDefaultUser:  "git",
			sshEnabled:      true,
			uiURLRaw:        "http://ui.example.com/",
			registryURLRaw:  "http://registry.example.com/",
			wantErr:         false,
		},
		{
			name:            "invalid internal URL",
			internalURLRaw:  "://invalid",
			containerURLRaw: "http://container.example.com",
			apiURLRaw:       "http://api.example.com",
			gitURLRaw:       "http://git.example.com",
			gitSSHURLRaw:    "ssh://git.example.com:22",
			sshDefaultUser:  "git",
			sshEnabled:      false,
			uiURLRaw:        "http://ui.example.com",
			registryURLRaw:  "http://registry.example.com",
			wantErr:         true,
		},
		{
			name:            "invalid container URL",
			internalURLRaw:  "http://internal.example.com",
			containerURLRaw: "://invalid",
			apiURLRaw:       "http://api.example.com",
			gitURLRaw:       "http://git.example.com",
			gitSSHURLRaw:    "ssh://git.example.com:22",
			sshDefaultUser:  "git",
			sshEnabled:      false,
			uiURLRaw:        "http://ui.example.com",
			registryURLRaw:  "http://registry.example.com",
			wantErr:         true,
		},
		{
			name:            "invalid SSH URL when SSH enabled",
			internalURLRaw:  "http://internal.example.com",
			containerURLRaw: "http://container.example.com",
			apiURLRaw:       "http://api.example.com",
			gitURLRaw:       "http://git.example.com",
			gitSSHURLRaw:    "://invalid",
			sshDefaultUser:  "git",
			sshEnabled:      true,
			uiURLRaw:        "http://ui.example.com",
			registryURLRaw:  "http://registry.example.com",
			wantErr:         true,
		},
		{
			name:            "invalid SSH URL when SSH disabled (should not error)",
			internalURLRaw:  "http://internal.example.com",
			containerURLRaw: "http://container.example.com",
			apiURLRaw:       "http://api.example.com",
			gitURLRaw:       "http://git.example.com",
			gitSSHURLRaw:    "://invalid",
			sshDefaultUser:  "git",
			sshEnabled:      false,
			uiURLRaw:        "http://ui.example.com",
			registryURLRaw:  "http://registry.example.com",
			wantErr:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewProvider(
				tt.internalURLRaw,
				tt.containerURLRaw,
				tt.apiURLRaw,
				tt.gitURLRaw,
				tt.gitSSHURLRaw,
				tt.sshDefaultUser,
				tt.sshEnabled,
				tt.uiURLRaw,
				tt.registryURLRaw,
			)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewProvider() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestProvider_GetInternalAPIURL(t *testing.T) {
	p, err := NewProvider(
		"http://internal.example.com",
		"http://container.example.com",
		"http://api.example.com",
		"http://git.example.com",
		"ssh://git.example.com:22",
		"git",
		false,
		"http://ui.example.com",
		"http://registry.example.com",
	)
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}

	ctx := context.Background()
	got := p.GetInternalAPIURL(ctx)
	want := "http://internal.example.com/api"

	if got != want {
		t.Errorf("GetInternalAPIURL() = %v, want %v", got, want)
	}
}

func TestProvider_GenerateContainerGITCloneURL(t *testing.T) {
	p, err := NewProvider(
		"http://internal.example.com",
		"http://container.example.com",
		"http://api.example.com",
		"http://git.example.com",
		"ssh://git.example.com:22",
		"git",
		false,
		"http://ui.example.com",
		"http://registry.example.com",
	)
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}

	tests := []struct {
		name     string
		repoPath string
		want     string
	}{
		{
			name:     "simple repo path",
			repoPath: "org/repo",
			want:     "http://container.example.com/git/org/repo.git",
		},
		{
			name:     "repo path with .git suffix",
			repoPath: "org/repo.git",
			want:     "http://container.example.com/git/org/repo.git",
		},
		{
			name:     "repo path with leading slash",
			repoPath: "/org/repo",
			want:     "http://container.example.com/git/org/repo.git",
		},
	}

	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.GenerateContainerGITCloneURL(ctx, tt.repoPath)
			if got != tt.want {
				t.Errorf("GenerateContainerGITCloneURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProvider_GenerateGITCloneURL(t *testing.T) {
	p, err := NewProvider(
		"http://internal.example.com",
		"http://container.example.com",
		"http://api.example.com",
		"http://git.example.com",
		"ssh://git.example.com:22",
		"git",
		false,
		"http://ui.example.com",
		"http://registry.example.com",
	)
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}

	tests := []struct {
		name     string
		repoPath string
		want     string
	}{
		{
			name:     "simple repo path",
			repoPath: "org/repo",
			want:     "http://git.example.com/org/repo.git",
		},
		{
			name:     "repo path with .git suffix",
			repoPath: "org/repo.git",
			want:     "http://git.example.com/org/repo.git",
		},
	}

	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.GenerateGITCloneURL(ctx, tt.repoPath)
			if got != tt.want {
				t.Errorf("GenerateGITCloneURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProvider_GenerateGITCloneSSHURL(t *testing.T) {
	tests := []struct {
		name       string
		sshEnabled bool
		repoPath   string
		want       string
	}{
		{
			name:       "SSH enabled",
			sshEnabled: true,
			repoPath:   "org/repo",
			want:       "git@git.example.com:org/repo.git",
		},
		{
			name:       "SSH disabled",
			sshEnabled: false,
			repoPath:   "org/repo",
			want:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := NewProvider(
				"http://internal.example.com",
				"http://container.example.com",
				"http://api.example.com",
				"http://git.example.com",
				"ssh://git.example.com:22",
				"git",
				tt.sshEnabled,
				"http://ui.example.com",
				"http://registry.example.com",
			)
			if err != nil {
				t.Fatalf("NewProvider() error = %v", err)
			}

			ctx := context.Background()
			got := p.GenerateGITCloneSSHURL(ctx, tt.repoPath)
			if got != tt.want {
				t.Errorf("GenerateGITCloneSSHURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProvider_GenerateUIRepoURL(t *testing.T) {
	p, err := NewProvider(
		"http://internal.example.com",
		"http://container.example.com",
		"http://api.example.com",
		"http://git.example.com",
		"ssh://git.example.com:22",
		"git",
		false,
		"http://ui.example.com",
		"http://registry.example.com",
	)
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}

	ctx := context.Background()
	got := p.GenerateUIRepoURL(ctx, "org/repo")
	want := "http://ui.example.com/org/repo"

	if got != want {
		t.Errorf("GenerateUIRepoURL() = %v, want %v", got, want)
	}
}

func TestProvider_GenerateUIPRURL(t *testing.T) {
	p, err := NewProvider(
		"http://internal.example.com",
		"http://container.example.com",
		"http://api.example.com",
		"http://git.example.com",
		"ssh://git.example.com:22",
		"git",
		false,
		"http://ui.example.com",
		"http://registry.example.com",
	)
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}

	ctx := context.Background()
	got := p.GenerateUIPRURL(ctx, "org/repo", 123)
	want := "http://ui.example.com/org/repo/-/pulls/123"

	if got != want {
		t.Errorf("GenerateUIPRURL() = %v, want %v", got, want)
	}
}

func TestProvider_GenerateUICompareURL(t *testing.T) {
	p, err := NewProvider(
		"http://internal.example.com",
		"http://container.example.com",
		"http://api.example.com",
		"http://git.example.com",
		"ssh://git.example.com:22",
		"git",
		false,
		"http://ui.example.com",
		"http://registry.example.com",
	)
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}

	ctx := context.Background()
	got := p.GenerateUICompareURL(ctx, "org/repo", "main", "develop")
	want := "http://ui.example.com/org/repo/-/pulls/compare/main...develop"

	if got != want {
		t.Errorf("GenerateUICompareURL() = %v, want %v", got, want)
	}
}

func TestProvider_GenerateUIRefURL(t *testing.T) {
	p, err := NewProvider(
		"http://internal.example.com",
		"http://container.example.com",
		"http://api.example.com",
		"http://git.example.com",
		"ssh://git.example.com:22",
		"git",
		false,
		"http://ui.example.com",
		"http://registry.example.com",
	)
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}

	ctx := context.Background()
	got := p.GenerateUIRefURL(ctx, "org/repo", "abc123")
	want := "http://ui.example.com/org/repo/-/commit/abc123"

	if got != want {
		t.Errorf("GenerateUIRefURL() = %v, want %v", got, want)
	}
}

func TestProvider_GetAPIHostname(t *testing.T) {
	p, err := NewProvider(
		"http://internal.example.com",
		"http://container.example.com",
		"http://api.example.com:8080",
		"http://git.example.com",
		"ssh://git.example.com:22",
		"git",
		false,
		"http://ui.example.com",
		"http://registry.example.com",
	)
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}

	ctx := context.Background()
	got := p.GetAPIHostname(ctx)
	want := "api.example.com"

	if got != want {
		t.Errorf("GetAPIHostname() = %v, want %v", got, want)
	}
}

func TestProvider_GetGITHostname(t *testing.T) {
	p, err := NewProvider(
		"http://internal.example.com",
		"http://container.example.com",
		"http://api.example.com",
		"http://git.example.com:9090",
		"ssh://git.example.com:22",
		"git",
		false,
		"http://ui.example.com",
		"http://registry.example.com",
	)
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}

	ctx := context.Background()
	got := p.GetGITHostname(ctx)
	want := "git.example.com"

	if got != want {
		t.Errorf("GetGITHostname() = %v, want %v", got, want)
	}
}

func TestProvider_GetAPIProto(t *testing.T) {
	tests := []struct {
		name      string
		apiURLRaw string
		want      string
	}{
		{
			name:      "http protocol",
			apiURLRaw: "http://api.example.com",
			want:      "http",
		},
		{
			name:      "https protocol",
			apiURLRaw: "https://api.example.com",
			want:      "https",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := NewProvider(
				"http://internal.example.com",
				"http://container.example.com",
				tt.apiURLRaw,
				"http://git.example.com",
				"ssh://git.example.com:22",
				"git",
				false,
				"http://ui.example.com",
				"http://registry.example.com",
			)
			if err != nil {
				t.Fatalf("NewProvider() error = %v", err)
			}

			ctx := context.Background()
			got := p.GetAPIProto(ctx)
			if got != tt.want {
				t.Errorf("GetAPIProto() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProvider_GetUIBaseURL(t *testing.T) {
	p, err := NewProvider(
		"http://internal.example.com",
		"http://container.example.com",
		"http://api.example.com",
		"http://git.example.com",
		"ssh://git.example.com:22",
		"git",
		false,
		"http://ui.example.com",
		"http://registry.example.com",
	)
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}

	ctx := context.Background()
	got := p.GetUIBaseURL(ctx)
	want := "http://ui.example.com"

	if got != want {
		t.Errorf("GetUIBaseURL() = %v, want %v", got, want)
	}
}

func TestProvider_GenerateUIBuildURL(t *testing.T) {
	p, err := NewProvider(
		"http://internal.example.com",
		"http://container.example.com",
		"http://api.example.com",
		"http://git.example.com",
		"ssh://git.example.com:22",
		"git",
		false,
		"http://ui.example.com",
		"http://registry.example.com",
	)
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}

	ctx := context.Background()
	got := p.GenerateUIBuildURL(ctx, "org/repo", "my-pipeline", 42)
	want := "http://ui.example.com/org/repo/-/pipelines/my-pipeline/execution/42"

	if got != want {
		t.Errorf("GenerateUIBuildURL() = %v, want %v", got, want)
	}
}

func TestProvider_RegistryURL(t *testing.T) {
	p, err := NewProvider(
		"http://internal.example.com",
		"http://container.example.com",
		"http://api.example.com",
		"http://git.example.com",
		"ssh://git.example.com:22",
		"git",
		false,
		"http://ui.example.com",
		"http://registry.example.com",
	)
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}

	tests := []struct {
		name   string
		params []string
		want   string
	}{
		{
			name:   "no params",
			params: []string{},
			want:   "http://registry.example.com",
		},
		{
			name:   "single param",
			params: []string{"docker"},
			want:   "http://registry.example.com/docker",
		},
		{
			name:   "generic type swaps params",
			params: []string{"myregistry", "generic"},
			want:   "http://registry.example.com/generic/myregistry",
		},
		{
			name:   "maven type swaps params",
			params: []string{"myregistry", "maven"},
			want:   "http://registry.example.com/maven/myregistry",
		},
		{
			name:   "docker type lowercase",
			params: []string{"DOCKER"},
			want:   "http://registry.example.com/docker",
		},
	}

	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.RegistryURL(ctx, tt.params...)
			if got != tt.want {
				t.Errorf("RegistryURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProvider_PackageURL(t *testing.T) {
	p, err := NewProvider(
		"http://internal.example.com",
		"http://container.example.com",
		"http://api.example.com",
		"http://git.example.com",
		"ssh://git.example.com:22",
		"git",
		false,
		"http://ui.example.com",
		"http://registry.example.com",
	)
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}

	tests := []struct {
		name    string
		regRef  string
		pkgType string
		params  []string
		want    string
	}{
		{
			name:    "basic package URL",
			regRef:  "myregistry",
			pkgType: "docker",
			params:  []string{},
			want:    "http://registry.example.com/pkg/myregistry/docker",
		},
		{
			name:    "package URL with params",
			regRef:  "myregistry",
			pkgType: "docker",
			params:  []string{"myimage", "v1.0.0"},
			want:    "http://registry.example.com/pkg/myregistry/docker/myimage/v1.0.0",
		},
	}

	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.PackageURL(ctx, tt.regRef, tt.pkgType, tt.params...)
			if got != tt.want {
				t.Errorf("PackageURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProvider_GenerateUIRegistryURL(t *testing.T) {
	p, err := NewProvider(
		"http://internal.example.com",
		"http://container.example.com",
		"http://api.example.com",
		"http://git.example.com",
		"ssh://git.example.com:22",
		"git",
		false,
		"http://ui.example.com",
		"http://registry.example.com",
	)
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}

	tests := []struct {
		name            string
		parentSpacePath string
		registryName    string
		want            string
	}{
		{
			name:            "valid space path",
			parentSpacePath: "myspace",
			registryName:    "myregistry",
			want:            "http://ui.example.com/spaces/myspace/registries/myregistry",
		},
		{
			name:            "nested space path",
			parentSpacePath: "myspace/subspace",
			registryName:    "myregistry",
			want:            "http://ui.example.com/spaces/myspace/registries/myregistry",
		},
	}

	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.GenerateUIRegistryURL(ctx, tt.parentSpacePath, tt.registryName)
			if got != tt.want {
				t.Errorf("GenerateUIRegistryURL() = %v, want %v", got, tt.want)
			}
		})
	}
}
