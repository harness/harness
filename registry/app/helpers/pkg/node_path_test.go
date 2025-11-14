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

package pkg

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Updated tests with new method signatures (no error returns)

func TestCargoPackageType_GetNodePathsForImage_Updated(t *testing.T) {
	cargoPackage := NewCargoPackageType(nil, nil)

	tests := []struct {
		name          string
		packageName   string
		expectedPaths []string
	}{
		{
			name:          "simple crate name",
			packageName:   "serde",
			expectedPaths: []string{"/crates/serde"},
		},
		{
			name:          "crate with hyphens",
			packageName:   "serde-json",
			expectedPaths: []string{"/crates/serde-json"},
		},
		{
			name:          "crate with underscores",
			packageName:   "tokio_util",
			expectedPaths: []string{"/crates/tokio_util"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paths := cargoPackage.GetNodePathsForImage(nil, tt.packageName)
			assert.Equal(t, tt.expectedPaths, paths)
		})
	}
}

func TestCargoPackageType_GetNodePathsForArtifact_Updated(t *testing.T) {
	cargoPackage := NewCargoPackageType(nil, nil)

	tests := []struct {
		name          string
		packageName   string
		version       string
		expectedPaths []string
	}{
		{
			name:          "simple crate with version",
			packageName:   "serde",
			version:       "1.0.0",
			expectedPaths: []string{"/crates/serde/1.0.0"},
		},
		{
			name:          "crate with pre-release version",
			packageName:   "tokio",
			version:       "1.0.0-alpha.1",
			expectedPaths: []string{"/crates/tokio/1.0.0-alpha.1"},
		},
		{
			name:          "crate with build metadata",
			packageName:   "async-std",
			version:       "1.0.0+build.1",
			expectedPaths: []string{"/crates/async-std/1.0.0+build.1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paths := cargoPackage.GetNodePathsForArtifact(nil, tt.packageName, tt.version)
			assert.Equal(t, tt.expectedPaths, paths)
		})
	}
}

func TestDockerPackageType_GetNodePathsForImage_Updated(t *testing.T) {
	dockerPackage := NewDockerPackageType(nil)

	tests := []struct {
		name          string
		packageName   string
		expectedPaths []string
	}{
		{
			name:          "simple image name",
			packageName:   "nginx",
			expectedPaths: []string{"/nginx"},
		},
		{
			name:          "namespaced image",
			packageName:   "library/nginx",
			expectedPaths: []string{"/library/nginx"},
		},
		{
			name:          "registry with namespace",
			packageName:   "gcr.io/project/image",
			expectedPaths: []string{"/gcr.io/project/image"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paths := dockerPackage.GetNodePathsForImage(nil, tt.packageName)
			assert.Equal(t, tt.expectedPaths, paths)
		})
	}
}

func TestDockerPackageType_GetNodePathsForArtifact_Updated(t *testing.T) {
	dockerPackage := NewDockerPackageType(nil)

	tests := []struct {
		name          string
		packageName   string
		version       string
		expectedPaths []string
	}{
		{
			name:          "simple image with tag",
			packageName:   "nginx",
			version:       "1.21",
			expectedPaths: []string{"/nginx/1.21"},
		},
		{
			name:          "image with latest tag",
			packageName:   "alpine",
			version:       "latest",
			expectedPaths: []string{"/alpine/latest"},
		},
		{
			name:          "namespaced image with version",
			packageName:   "library/postgres",
			version:       "13.4",
			expectedPaths: []string{"/library/postgres/13.4"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paths := dockerPackage.GetNodePathsForArtifact(nil, tt.packageName, tt.version)
			assert.Equal(t, tt.expectedPaths, paths)
		})
	}
}

func TestMavenPackageType_GetNodePathsForImage_Updated(t *testing.T) {
	mavenPackage := NewMavenPackageType(nil)

	tests := []struct {
		name          string
		packageName   string
		expectedPaths []string
	}{
		{
			name:          "simple group and artifact",
			packageName:   "com.example:my-artifact",
			expectedPaths: []string{"/com/example/my-artifact"},
		},
		{
			name:          "nested group structure",
			packageName:   "org.springframework.boot:spring-boot-starter",
			expectedPaths: []string{"/org/springframework/boot/spring-boot-starter"},
		},
		{
			name:          "artifact with multiple dots",
			packageName:   "com.fasterxml.jackson.core:jackson-core",
			expectedPaths: []string{"/com/fasterxml/jackson/core/jackson-core"},
		},
		{
			name:          "artifact without colon separator",
			packageName:   "junit.junit",
			expectedPaths: []string{"/"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paths := mavenPackage.GetNodePathsForImage(nil, tt.packageName)
			assert.Equal(t, tt.expectedPaths, paths)
		})
	}
}

func TestMavenPackageType_GetNodePathsForArtifact_Updated(t *testing.T) {
	mavenPackage := NewMavenPackageType(nil)

	tests := []struct {
		name          string
		packageName   string
		version       string
		expectedPaths []string
	}{
		{
			name:          "simple group and artifact with version",
			packageName:   "com.example:my-artifact",
			version:       "1.0.0",
			expectedPaths: []string{"/com/example/my-artifact/1.0.0"},
		},
		{
			name:          "spring boot artifact with version",
			packageName:   "org.springframework.boot:spring-boot-starter",
			version:       "2.7.0",
			expectedPaths: []string{"/org/springframework/boot/spring-boot-starter/2.7.0"},
		},
		{
			name:          "snapshot version",
			packageName:   "com.example:test-artifact",
			version:       "1.0.0-SNAPSHOT",
			expectedPaths: []string{"/com/example/test-artifact/1.0.0-SNAPSHOT"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paths := mavenPackage.GetNodePathsForArtifact(nil, tt.packageName, tt.version)
			assert.Equal(t, tt.expectedPaths, paths)
		})
	}
}

func TestGoPackageType_GetNodePathsForImage_Updated(t *testing.T) {
	goPackage := NewGoPackageType(nil)

	tests := []struct {
		name          string
		packageName   string
		expectedPaths []string
	}{
		{
			name:          "github module",
			packageName:   "github.com/gin-gonic/gin",
			expectedPaths: []string{"/github.com/gin-gonic/gin"},
		},
		{
			name:          "golang.org module",
			packageName:   "golang.org/x/crypto",
			expectedPaths: []string{"/golang.org/x/crypto"},
		},
		{
			name:          "custom domain module",
			packageName:   "go.uber.org/zap",
			expectedPaths: []string{"/go.uber.org/zap"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paths := goPackage.GetNodePathsForImage(nil, tt.packageName)
			assert.Equal(t, tt.expectedPaths, paths)
		})
	}
}

func TestGoPackageType_GetNodePathsForArtifact_Updated(t *testing.T) {
	goPackage := NewGoPackageType(nil)

	tests := []struct {
		name          string
		packageName   string
		version       string
		expectedPaths []string
	}{
		{
			name:          "github module with version",
			packageName:   "github.com/gin-gonic/gin",
			version:       "v1.8.1",
			expectedPaths: []string{"/github.com/gin-gonic/gin/@v/v1.8.1"},
		},
		{
			name:          "golang.org module with version",
			packageName:   "golang.org/x/crypto",
			version:       "v0.0.0-20220622213112-05595931fe9d",
			expectedPaths: []string{"/golang.org/x/crypto/@v/v0.0.0-20220622213112-05595931fe9d"},
		},
		{
			name:          "pre-release version",
			packageName:   "go.uber.org/zap",
			version:       "v1.22.0-rc.1",
			expectedPaths: []string{"/go.uber.org/zap/@v/v1.22.0-rc.1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paths := goPackage.GetNodePathsForArtifact(nil, tt.packageName, tt.version)
			assert.Equal(t, tt.expectedPaths, paths)
		})
	}
}

func TestRpmPackageType_GetNodePathsForImage_Updated(t *testing.T) {
	rpmPackage := NewRPMPackageType(nil)

	tests := []struct {
		name          string
		packageName   string
		expectedPaths []string
	}{
		{
			name:          "simple package name",
			packageName:   "nginx",
			expectedPaths: []string{"/nginx"},
		},
		{
			name:          "package with hyphens",
			packageName:   "httpd-tools",
			expectedPaths: []string{"/httpd-tools"},
		},
		{
			name:          "kernel package",
			packageName:   "kernel-devel",
			expectedPaths: []string{"/kernel-devel"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paths := rpmPackage.GetNodePathsForImage(nil, tt.packageName)
			assert.Equal(t, tt.expectedPaths, paths)
		})
	}
}

func TestRpmPackageType_GetNodePathsForArtifact_Updated(t *testing.T) {
	rpmPackage := NewRPMPackageType(nil)

	tests := []struct {
		name          string
		packageName   string
		version       string
		expectedPaths []string
	}{
		{
			name:          "simple package with version",
			packageName:   "nginx",
			version:       "1.20.1-1.el8.x86_64",
			expectedPaths: []string{"/nginx/1.20.1-1.el8/x86_64"},
		},
		{
			name:          "package with complex architecture",
			packageName:   "httpd",
			version:       "2.4.37-43.module+el8.5.0+13806+b30d9eec.x86_64",
			expectedPaths: []string{"/httpd/2.4.37-43.module+el8.5.0+13806+b30d9eec/x86_64"},
		},
		{
			name:          "kernel package with version",
			packageName:   "kernel-devel",
			version:       "4.18.0-348.el8.noarch",
			expectedPaths: []string{"/kernel-devel/4.18.0-348.el8/noarch"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paths := rpmPackage.GetNodePathsForArtifact(nil, tt.packageName, tt.version)
			assert.Equal(t, tt.expectedPaths, paths)
		})
	}
}

func TestNpmPackageType_GetNodePathsForImage_Updated(t *testing.T) {
	npmPackage := NewNPMPackageType(nil)

	tests := []struct {
		name          string
		packageName   string
		expectedPaths []string
	}{
		{
			name:          "simple package name",
			packageName:   "express",
			expectedPaths: []string{"/express"},
		},
		{
			name:          "scoped package",
			packageName:   "@types/node",
			expectedPaths: []string{"/@types/node"},
		},
		{
			name:          "organization scoped package",
			packageName:   "@angular/core",
			expectedPaths: []string{"/@angular/core"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paths := npmPackage.GetNodePathsForImage(nil, tt.packageName)
			assert.Equal(t, tt.expectedPaths, paths)
		})
	}
}

func TestNpmPackageType_GetNodePathsForArtifact_Updated(t *testing.T) {
	npmPackage := NewNPMPackageType(nil)

	tests := []struct {
		name          string
		packageName   string
		version       string
		expectedPaths []string
	}{
		{
			name:          "simple package with version",
			packageName:   "express",
			version:       "4.18.1",
			expectedPaths: []string{"/express/4.18.1"},
		},
		{
			name:          "scoped package with version",
			packageName:   "@types/node",
			version:       "18.0.0",
			expectedPaths: []string{"/@types/node/18.0.0"},
		},
		{
			name:          "pre-release version",
			packageName:   "react",
			version:       "18.0.0-rc.0",
			expectedPaths: []string{"/react/18.0.0-rc.0"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paths := npmPackage.GetNodePathsForArtifact(nil, tt.packageName, tt.version)
			assert.Equal(t, tt.expectedPaths, paths)
		})
	}
}

func TestHelmPackageType_GetNodePathsForImage_Updated(t *testing.T) {
	helmPackage := NewHelmPackageType(nil)

	tests := []struct {
		name          string
		packageName   string
		expectedPaths []string
	}{
		{
			name:          "simple chart name",
			packageName:   "nginx",
			expectedPaths: []string{"/nginx"},
		},
		{
			name:          "chart with hyphens",
			packageName:   "nginx-ingress",
			expectedPaths: []string{"/nginx-ingress"},
		},
		{
			name:          "complex chart name",
			packageName:   "prometheus-operator",
			expectedPaths: []string{"/prometheus-operator"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paths := helmPackage.GetNodePathsForImage(nil, tt.packageName)
			assert.Equal(t, tt.expectedPaths, paths)
		})
	}
}

func TestHelmPackageType_GetNodePathsForArtifact_Updated(t *testing.T) {
	helmPackage := NewHelmPackageType(nil)

	tests := []struct {
		name          string
		packageName   string
		version       string
		expectedPaths []string
	}{
		{
			name:          "simple chart with version",
			packageName:   "nginx",
			version:       "1.0.0",
			expectedPaths: []string{"/nginx/1.0.0"},
		},
		{
			name:          "chart with semantic version",
			packageName:   "prometheus",
			version:       "15.10.1",
			expectedPaths: []string{"/prometheus/15.10.1"},
		},
		{
			name:          "chart with pre-release version",
			packageName:   "grafana",
			version:       "6.32.0-beta.1",
			expectedPaths: []string{"/grafana/6.32.0-beta.1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paths := helmPackage.GetNodePathsForArtifact(nil, tt.packageName, tt.version)
			assert.Equal(t, tt.expectedPaths, paths)
		})
	}
}

func TestGenericPackageType_GetNodePathsForImage_Updated(t *testing.T) {
	genericPackage := NewGenericPackageType(nil)

	tests := []struct {
		name          string
		packageName   string
		expectedPaths []string
	}{
		{
			name:          "simple package name",
			packageName:   "mypackage",
			expectedPaths: []string{"/mypackage"},
		},
		{
			name:          "package with hyphens",
			packageName:   "my-generic-package",
			expectedPaths: []string{"/my-generic-package"},
		},
		{
			name:          "package with underscores",
			packageName:   "my_package_name",
			expectedPaths: []string{"/my_package_name"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paths := genericPackage.GetNodePathsForImage(nil, tt.packageName)
			assert.Equal(t, tt.expectedPaths, paths)
		})
	}
}

func TestGenericPackageType_GetNodePathsForArtifact_Updated(t *testing.T) {
	genericPackage := NewGenericPackageType(nil)

	tests := []struct {
		name          string
		packageName   string
		version       string
		expectedPaths []string
	}{
		{
			name:          "simple package with version",
			packageName:   "mypackage",
			version:       "1.0.0",
			expectedPaths: []string{"/mypackage/1.0.0"},
		},
		{
			name:          "package with semantic version",
			packageName:   "data-processor",
			version:       "2.5.3",
			expectedPaths: []string{"/data-processor/2.5.3"},
		},
		{
			name:          "package with custom version string",
			packageName:   "my_tool",
			version:       "v1.2.3-alpha",
			expectedPaths: []string{"/my_tool/v1.2.3-alpha"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paths := genericPackage.GetNodePathsForArtifact(nil, tt.packageName, tt.version)
			assert.Equal(t, tt.expectedPaths, paths)
		})
	}
}

func TestNugetPackageType_GetNodePathsForImage_Updated(t *testing.T) {
	nugetPackage := NewNugetPackageType(nil)

	tests := []struct {
		name          string
		packageName   string
		expectedPaths []string
	}{
		{
			name:          "simple package name",
			packageName:   "Newtonsoft.Json",
			expectedPaths: []string{"/Newtonsoft.Json"},
		},
		{
			name:          "microsoft package",
			packageName:   "Microsoft.Extensions.Logging",
			expectedPaths: []string{"/Microsoft.Extensions.Logging"},
		},
		{
			name:          "entity framework package",
			packageName:   "EntityFramework.Core",
			expectedPaths: []string{"/EntityFramework.Core"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paths := nugetPackage.GetNodePathsForImage(nil, tt.packageName)
			assert.Equal(t, tt.expectedPaths, paths)
		})
	}
}

func TestNugetPackageType_GetNodePathsForArtifact_Updated(t *testing.T) {
	nugetPackage := NewNugetPackageType(nil)

	tests := []struct {
		name          string
		packageName   string
		version       string
		expectedPaths []string
	}{
		{
			name:          "simple package with version",
			packageName:   "Newtonsoft.Json",
			version:       "13.0.1",
			expectedPaths: []string{"/Newtonsoft.Json/13.0.1"},
		},
		{
			name:          "microsoft package with version",
			packageName:   "Microsoft.Extensions.Logging",
			version:       "6.0.0",
			expectedPaths: []string{"/Microsoft.Extensions.Logging/6.0.0"},
		},
		{
			name:          "pre-release version",
			packageName:   "EntityFramework.Core",
			version:       "7.0.0-preview.5",
			expectedPaths: []string{"/EntityFramework.Core/7.0.0-preview.5"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paths := nugetPackage.GetNodePathsForArtifact(nil, tt.packageName, tt.version)
			assert.Equal(t, tt.expectedPaths, paths)
		})
	}
}

func TestPythonPackageType_GetNodePathsForImage_Updated(t *testing.T) {
	pythonPackage := NewPythonPackageType(nil)

	tests := []struct {
		name          string
		packageName   string
		expectedPaths []string
	}{
		{
			name:          "simple package name",
			packageName:   "requests",
			expectedPaths: []string{"/requests"},
		},
		{
			name:          "package with hyphens",
			packageName:   "django-rest-framework",
			expectedPaths: []string{"/django-rest-framework"},
		},
		{
			name:          "numpy package",
			packageName:   "numpy",
			expectedPaths: []string{"/numpy"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paths := pythonPackage.GetNodePathsForImage(nil, tt.packageName)
			assert.Equal(t, tt.expectedPaths, paths)
		})
	}
}

func TestPythonPackageType_GetNodePathsForArtifact_Updated(t *testing.T) {
	pythonPackage := NewPythonPackageType(nil)

	tests := []struct {
		name          string
		packageName   string
		version       string
		expectedPaths []string
	}{
		{
			name:          "simple package with version",
			packageName:   "requests",
			version:       "2.28.1",
			expectedPaths: []string{"/requests/2.28.1"},
		},
		{
			name:          "django package with version",
			packageName:   "django",
			version:       "4.1.0",
			expectedPaths: []string{"/django/4.1.0"},
		},
		{
			name:          "pre-release version",
			packageName:   "pytest",
			version:       "7.2.0rc1",
			expectedPaths: []string{"/pytest/7.2.0rc1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paths := pythonPackage.GetNodePathsForArtifact(nil, tt.packageName, tt.version)
			assert.Equal(t, tt.expectedPaths, paths)
		})
	}
}

func TestHuggingFacePackageType_GetNodePathsForImage_Updated(t *testing.T) {
	huggingFacePackage := NewHuggingFacePackageType(nil)

	tests := []struct {
		name          string
		packageName   string
		expectedPaths []string
	}{
		{
			name:          "simple model name",
			packageName:   "bert-base-uncased",
			expectedPaths: []string{"/bert-base-uncased"},
		},
		{
			name:          "namespaced model",
			packageName:   "facebook/bart-large",
			expectedPaths: []string{"/facebook/bart-large"},
		},
		{
			name:          "organization model",
			packageName:   "openai/whisper-base",
			expectedPaths: []string{"/openai/whisper-base"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paths := huggingFacePackage.GetNodePathsForImage(nil, tt.packageName)
			assert.Equal(t, tt.expectedPaths, paths)
		})
	}
}

func TestHuggingFacePackageType_GetNodePathsForArtifact_Updated(t *testing.T) {
	huggingFacePackage := NewHuggingFacePackageType(nil)

	tests := []struct {
		name          string
		packageName   string
		version       string
		expectedPaths []string
	}{
		{
			name:          "simple model with version",
			packageName:   "bert-base-uncased",
			version:       "v1.0",
			expectedPaths: []string{"/bert-base-uncased/v1.0"},
		},
		{
			name:          "namespaced model with commit hash",
			packageName:   "facebook/bart-large",
			version:       "abc123def456",
			expectedPaths: []string{"/facebook/bart-large/abc123def456"},
		},
		{
			name:          "model with semantic version",
			packageName:   "openai/whisper-base",
			version:       "1.2.3",
			expectedPaths: []string{"/openai/whisper-base/1.2.3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paths := huggingFacePackage.GetNodePathsForArtifact(nil, tt.packageName, tt.version)
			assert.Equal(t, tt.expectedPaths, paths)
		})
	}
}
