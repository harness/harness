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

package utils

import (
	"net/http"
	"testing"
	"time"

	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/types"

	"github.com/stretchr/testify/require"
)

func rpfMavenInfo(group, artifact, version, file string) pkg.MavenArtifactInfo {
	return pkg.MavenArtifactInfo{GroupID: group, ArtifactID: artifact, Version: version, FileName: file}
}

func TestRpfGetFilePath(t *testing.T) {
	tests := []struct {
		name string
		info pkg.MavenArtifactInfo
		want string
	}{
		{"with version", rpfMavenInfo("org.acme.tools", "demo", "1.0", "demo-1.0.jar"),
			"/org/acme/tools/demo/1.0/demo-1.0.jar"},
		{"without version", rpfMavenInfo("org.acme", "demo", "", "maven-metadata.xml"),
			"/org/acme/demo/maven-metadata.xml"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, GetFilePath(tc.info))
		})
	}
}

func TestRpfIsMainArtifactFile(t *testing.T) {
	tests := []struct {
		file string
		want bool
	}{
		{"demo-1.0.jar", true},
		{"demo-1.0.JAR", true},
		{"demo-1.0.war", true},
		{"demo-1.0.zip", true},
		{"demo-1.0.pom", false},
		{"demo-1.0.jar.sha1", false},
		{"demo-1.0.module", false},
	}
	for _, tc := range tests {
		t.Run(tc.file, func(t *testing.T) {
			require.Equal(t, tc.want, IsMainArtifactFile(rpfMavenInfo("g", "demo", "1.0", tc.file)))
		})
	}
}

func TestRpfIsMetadataFile(t *testing.T) {
	for _, name := range []string{
		"maven-metadata.xml", "maven-metadata.xml.md5", "maven-metadata.xml.sha1",
		"maven-metadata.xml.sha256", "maven-metadata.xml.sha512",
	} {
		require.True(t, IsMetadataFile(name), name)
	}
	for _, name := range []string{"", "demo.pom", "maven-metadata.xml.asc", "other-maven-metadata.xml"} {
		require.False(t, IsMetadataFile(name), name)
	}
}

func TestRpfSetHeaders(t *testing.T) {
	created := time.Unix(1700000000, 0)
	fileInfo := types.FileInfo{Size: 123, CreatedAt: created, Filename: "demo.jar"}

	tests := []struct {
		file        string
		contentType string
	}{
		{"demo-1.0.jar", "application/java-archive"},
		{"demo-1.0.pom", "text/xml"},
		{"maven-metadata.xml", "text/xml"},
		{"demo-1.0.jar.md5", "text/plain"},
		{"demo-1.0.jar.sha1", "text/plain"},
		{"demo-1.0.jar.sha256", "text/plain"},
		{"demo-1.0.jar.sha512", "text/plain"},
		{"demo-1.0.bin", ""},
	}
	for _, tc := range tests {
		t.Run(tc.file, func(t *testing.T) {
			h := SetHeaders(rpfMavenInfo("g", "demo", "1.0", tc.file), fileInfo)
			require.Equal(t, http.StatusOK, h.Code)
			require.Equal(t, "123", h.Headers["Content-Length"])
			require.Equal(t, "1700000000", h.Headers["LastModified"])
			require.Equal(t, "demo.jar", h.Headers["FileName"])
			require.Equal(t, tc.contentType, h.Headers["Content-Type"])
		})
	}
}

func TestRpfIsSnapshotVersion(t *testing.T) {
	require.True(t, IsSnapshotVersion(rpfMavenInfo("g", "a", "1.0-SNAPSHOT", "")))
	require.False(t, IsSnapshotVersion(rpfMavenInfo("g", "a", "1.0", "")))
	require.False(t, IsSnapshotVersion(rpfMavenInfo("g", "a", "1.0-snapshot-x", "")))
}

func TestRpfAddLikeBeforeExtension(t *testing.T) {
	got := AddLikeBeforeExtension(rpfMavenInfo("org.acme", "demo", "1.0-SNAPSHOT", "demo-1.0-SNAPSHOT.jar"))
	require.Equal(t, "/org/acme/demo/1.0-SNAPSHOT/demo-1.0-SNAPSHOT%.jar", got)
}

func TestRpfParseResponseHeaders(t *testing.T) {
	resp := &http.Response{Header: http.Header{
		"Content-Type": []string{"text/xml", "ignored"},
		"X-Multi":      []string{"first"},
	}}
	h := ParseResponseHeaders(resp)
	require.Equal(t, http.StatusOK, h.Code)
	require.Equal(t, "text/xml", h.Headers["Content-Type"])
	require.Equal(t, "first", h.Headers["X-Multi"])

	h = ParseResponseHeaders(&http.Response{})
	require.Empty(t, h.Headers)
}
