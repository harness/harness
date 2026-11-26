//  Copyright 2026 Harness, Inc.
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

package metadata

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	registrymetadata "github.com/harness/gitness/registry/app/metadata"
	"github.com/harness/gitness/registry/types"

	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/require"
)

func rmdTime() time.Time {
	return time.UnixMilli(1700000000000)
}

func TestRmdGetArtifactMetadata(t *testing.T) {
	ctx := rmdAdminCtx()
	deleted := rmdTime()
	hfType := artifact.ArtifactTypeModel
	items := []types.ArtifactMetadata{
		{Name: "img", RepoName: "dockerreg", PackageType: artifact.PackageTypeDOCKER, Version: "v1",
			ModifiedAt: rmdTime(), Tags: []string{"a", "b"}, DeletedAt: &deleted, DownloadCount: 3},
		{Name: "gen", RepoName: "genreg", PackageType: artifact.PackageTypeGENERIC, Version: "1.0",
			ModifiedAt: rmdTime()},
		{Name: "hf", RepoName: "hfreg", PackageType: artifact.PackageTypeHUGGINGFACE, Version: "main",
			ModifiedAt: rmdTime(), ArtifactType: &hfType},
	}

	t.Run("mapped_without_wrapper_override", func(t *testing.T) {
		got := GetArtifactMetadata(ctx, items, "root", rmdURLProvider{}, "Bearer", true, rmdPkgWrapper{})
		require.Len(t, got, 3)

		require.Equal(t, "img", got[0].Name)
		require.Equal(t, "docker pull reg.example.com/root/dockerreg/img@v1", *got[0].PullCommand)
		require.NotNil(t, got[0].DeletedAt)
		require.Equal(t, "1700000000000", *got[0].DeletedAt)
		require.NotNil(t, got[0].Metadata)
		require.Equal(t, []string{"a", "b"}, (*got[0].Metadata)["tags"])

		require.Equal(t, "genreg", got[1].RegistryIdentifier)
		require.Contains(t, *got[1].PullCommand, "https://reg.example.com/root/generic/genreg/gen/1.0/<FILENAME>")
		require.Nil(t, got[1].DeletedAt)
		require.Nil(t, got[1].Metadata)

		require.Contains(t, *got[2].PullCommand, "/model/hf/resolve/main/<FILENAME>")
	})

	t.Run("untagged_images_use_digest_delimiter", func(t *testing.T) {
		got := GetArtifactMetadata(ctx, items[:1], "root", rmdURLProvider{}, "Bearer", false, rmdPkgWrapper{})
		require.Equal(t, "docker pull reg.example.com/root/dockerreg/img:v1", *got[0].PullCommand)
	})

	t.Run("wrapper_override_wins", func(t *testing.T) {
		override := &artifact.ArtifactMetadata{Name: "custom"}
		got := GetArtifactMetadata(ctx, items, "root", rmdURLProvider{}, "Bearer", true,
			rmdPkgWrapper{artifactMetadata: override})
		require.Len(t, got, 3)
		for _, g := range got {
			require.Equal(t, "custom", g.Name)
		}
	})

	t.Run("empty_input", func(t *testing.T) {
		got := GetArtifactMetadata(ctx, nil, "root", rmdURLProvider{}, "Bearer", true, rmdPkgWrapper{})
		require.Empty(t, got)
		require.NotNil(t, got)
	})
}

func TestRmdGetRegistryArtifactMetadataDeleted(t *testing.T) {
	deleted := rmdTime()
	got := GetRegistryArtifactMetadata([]types.ArtifactMetadata{
		{Name: "n", DeletedAt: &deleted, ModifiedAt: rmdTime(), LatestVersion: "1"},
		{Name: "m", ModifiedAt: rmdTime()},
	})
	require.Len(t, got, 2)
	require.Equal(t, "1700000000000", *got[0].DeletedAt)
	require.Nil(t, got[1].DeletedAt)
	require.Equal(t, "1700000000000", *got[1].LastModified)
}

func TestRmdGetMavenArtifactDetail(t *testing.T) {
	image := &types.Image{Name: "lib"}
	art := &types.Artifact{Version: "1.2", CreatedAt: rmdTime(), UpdatedAt: rmdTime()}
	tests := []struct {
		name     string
		metadata registrymetadata.MavenMetadata
		wantSize string
	}{
		{
			name:     "explicit_size",
			metadata: registrymetadata.MavenMetadata{Size: 2048},
			wantSize: GetSize(2048),
		},
		{
			name: "size_summed_from_files",
			metadata: registrymetadata.MavenMetadata{
				Files: []registrymetadata.File{{Size: 1024}, {Size: 1024}},
			},
			wantSize: GetSize(2048),
		},
		{
			name:     "no_size_no_files",
			metadata: registrymetadata.MavenMetadata{},
			wantSize: GetSize(0),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetMavenArtifactDetail(image, art, tt.metadata)
			require.Equal(t, "lib", *got.Name)
			require.Equal(t, "1.2", got.Version)
			require.Equal(t, tt.wantSize, *got.Size)
		})
	}
}

func TestRmdToPackageType(t *testing.T) {
	valid := []artifact.PackageType{
		artifact.PackageTypeDOCKER, artifact.PackageTypeGENERIC, artifact.PackageTypeHELM,
		artifact.PackageTypeMAVEN, artifact.PackageTypePYTHON, artifact.PackageTypeNPM,
		artifact.PackageTypeRPM, artifact.PackageTypeNUGET, artifact.PackageTypeGO,
		artifact.PackageTypeHUGGINGFACE,
	}
	for _, pt := range valid {
		t.Run(string(pt), func(t *testing.T) {
			got, err := toPackageType(string(pt))
			require.NoError(t, err)
			require.Equal(t, pt, got)
		})
	}
	_, err := toPackageType("bogus")
	require.Error(t, err)
}

func TestRmdGetTagMetadata(t *testing.T) {
	ctx := rmdAdminCtx()
	versions := []types.OciVersionMetadata{
		{Name: "v1", Size: "1024", PackageType: artifact.PackageTypeDOCKER, DigestCount: 2,
			ModifiedAt: rmdTime(), DownloadCount: 5, Tags: []string{"t1"}, IsQuarantined: true,
			QuarantineReason: "bad"},
		{Name: "skipped", PackageType: artifact.PackageType("unknown")},
		{Name: "chart", Size: "0", PackageType: artifact.PackageTypeHELM, ModifiedAt: rmdTime()},
	}
	got := GetTagMetadata(ctx, &versions, "img", "https://reg.example.com/r", "Bearer", false)
	require.Len(t, got, 2, "versions with an unknown package type are skipped")
	require.Equal(t, "v1", got[0].Name)
	require.Equal(t, "docker pull reg.example.com/r/img:v1", *got[0].PullCommand)
	require.Equal(t, 2, *got[0].DigestCount)
	require.Equal(t, int64(5), *got[0].DownloadsCount)
	require.True(t, *got[0].IsQuarantined)
	require.Equal(t, "bad", *got[0].QuarantineReason)
	require.Equal(t, GetImageSize("1024"), *got[0].Size)
	require.Equal(t, []string{"t1"}, (*got[0].Metadata)["tags"])
	require.Equal(t, "helm pull oci://reg.example.com/r/img --version chart", *got[1].PullCommand)

	resp := GetAllArtifactVersionResponse(ctx, &versions, "img", 25, 2, 10,
		"https://reg.example.com/r", "Bearer", true)
	require.Equal(t, artifact.StatusSUCCESS, resp.Status)
	require.Equal(t, int64(3), *resp.Data.PageCount)
	require.Equal(t, int64(2), *resp.Data.PageIndex)
	require.Len(t, *resp.Data.ArtifactVersions, 2)
}

func TestRmdGetNonOCIArtifactMetadata(t *testing.T) {
	ctx := rmdAdminCtx()
	reason := "r"
	tags := []types.NonOCIArtifactMetadata{
		{Name: "1.0", UUID: "u1", Size: "2048", FileCount: 4, ModifiedAt: rmdTime(), DownloadCount: 9,
			IsQuarantined: true, QuarantineReason: &reason},
	}

	t.Run("default_mapping", func(t *testing.T) {
		got := GetNonOCIArtifactMetadata(ctx, &tags, "pkg", "https://reg.example.com/r", "Bearer",
			string(artifact.PackageTypeNPM), rmdPkgWrapper{}, "reg-uuid")
		require.Len(t, got, 1)
		require.Equal(t, "reg-uuid", got[0].RegistryUUID)
		require.Equal(t, "u1", got[0].Uuid)
		require.Equal(t, int64(4), *got[0].FileCount)
		require.Equal(t, "npm install pkg@1.0 ", *got[0].PullCommand)
		require.Equal(t, &reason, got[0].QuarantineReason)
		require.True(t, *got[0].IsQuarantined)
	})

	t.Run("unknown_package_type_skipped", func(t *testing.T) {
		got := GetNonOCIArtifactMetadata(ctx, &tags, "pkg", "u", "Bearer", "weird", rmdPkgWrapper{}, "x")
		require.Empty(t, got)
	})

	t.Run("wrapper_override_gets_registry_uuid", func(t *testing.T) {
		got := GetNonOCIArtifactMetadata(ctx, &tags, "pkg", "u", "Bearer", "weird",
			rmdPkgWrapper{versionMetadata: &artifact.ArtifactVersionMetadata{Name: "custom"}}, "reg-uuid")
		require.Len(t, got, 1)
		require.Equal(t, "custom", got[0].Name)
		require.Equal(t, "reg-uuid", got[0].RegistryUUID)
	})

	t.Run("response_wrapper", func(t *testing.T) {
		resp := GetNonOCIAllArtifactVersionResponse(ctx, &tags, "pkg", 1, 1, 10, "u", "Bearer",
			string(artifact.PackageTypeGO), rmdPkgWrapper{}, "reg-uuid")
		require.Equal(t, artifact.StatusSUCCESS, resp.Status)
		require.Equal(t, int64(1), *resp.Data.PageCount)
		require.Len(t, *resp.Data.ArtifactVersions, 1)
		require.Equal(t, "go get pkg@1.0", *(*resp.Data.ArtifactVersions)[0].PullCommand)
	})
}

func TestRmdGetAllArtifactResponse(t *testing.T) {
	ctx := rmdAdminCtx()
	t.Run("nil_artifacts", func(t *testing.T) {
		resp := GetAllArtifactResponse(ctx, nil, 0, 1, 10, "root", rmdURLProvider{}, "Bearer", true, rmdPkgWrapper{})
		require.NotNil(t, resp.Data.Artifacts)
		require.Empty(t, resp.Data.Artifacts)
		require.Equal(t, int64(0), *resp.Data.PageCount)
	})
	t.Run("with_artifacts", func(t *testing.T) {
		arts := []types.ArtifactMetadata{{Name: "a", RepoName: "r", PackageType: artifact.PackageTypeNPM}}
		resp := GetAllArtifactResponse(ctx, &arts, 11, 1, 5, "root", rmdURLProvider{}, "Bearer", true, rmdPkgWrapper{})
		require.Len(t, resp.Data.Artifacts, 1)
		require.Equal(t, int64(3), *resp.Data.PageCount)
		require.Equal(t, int64(11), *resp.Data.ItemCount)
	})
}

func TestRmdGetArtifactFilesMetadata(t *testing.T) {
	ctx := rmdAdminCtx()
	hfType := artifact.ArtifactTypeModel
	file := func(path string) types.FileNodeMetadata {
		return types.FileNodeMetadata{
			Path: path, Size: 1024, Sha1: "s1", Sha256: "s256", Sha512: "s512", MD5: "md5", CreatedAt: 5,
		}
	}
	tests := []struct {
		name         string
		packageType  artifact.PackageType
		artifactName string
		version      string
		path         string
		wantName     string
		wantCmd      string
	}{
		{
			name: "generic", packageType: artifact.PackageTypeGENERIC, artifactName: "g", version: "1",
			path: "/g/1/file.bin", wantName: "file.bin",
			wantCmd: "curl --location 'https://r/g/1/file.bin' --header 'Bearer <API_KEY>' -J -o '<OUTPUT_FILE_NAME>'",
		},
		{
			name: "python", packageType: artifact.PackageTypePYTHON, artifactName: "p", version: "2",
			path: "/p/2/p-2.whl", wantName: "p-2.whl",
			wantCmd: "curl --location 'https://r/files/p/2/p-2.whl' --header 'Bearer <API_KEY>' -J -o '<OUTPUT_FILE_NAME>'",
		},
		{
			name: "npm", packageType: artifact.PackageTypeNPM, artifactName: "n", version: "3",
			path: "/n/3/n-3.tgz", wantName: "n-3.tgz",
			wantCmd: "curl --location 'https://r/n/-/n-3.tgz' --header 'Authorization: Bearer <API_KEY>' " +
				"-J -o '<OUTPUT_FILE_NAME>'",
		},
		{
			name: "maven_dots_become_slashes", packageType: artifact.PackageTypeMAVEN,
			artifactName: "com.acme:lib", version: "4", path: "/com/acme/lib/4/lib-4.jar", wantName: "lib-4.jar",
			wantCmd: "curl --location 'https://r/com/acme/lib/4/lib-4.jar' --header 'Bearer <API_KEY>' " +
				"-o '<OUTPUT_FILE_NAME>'",
		},
		{
			name: "rpm", packageType: artifact.PackageTypeRPM, artifactName: "r", version: "1.0.x86_64",
			path: "/r/1.0.x86_64/dir/r.rpm", wantName: "r.rpm",
			wantCmd: "curl --location 'https://r/package/r/1.0/x86_64/r.rpm' --header 'Bearer <API_KEY>' " +
				"-J -o '<OUTPUT_FILE_NAME>'",
		},
		{
			name: "nuget", packageType: artifact.PackageTypeNUGET, artifactName: "nu", version: "5",
			path: "/nu/5/nu.nupkg", wantName: "nu.nupkg",
			wantCmd: "curl --location 'https://r/package/nu/5/nu.nupkg' --header 'Bearer <API_KEY>' " +
				"-J -o '<OUTPUT_FILE_NAME>'",
		},
		{
			name: "go", packageType: artifact.PackageTypeGO, artifactName: "mod", version: "v1",
			path: "/mod/@v/v1/v1.zip", wantName: "v1.zip",
			wantCmd: "curl --location 'https://r/mod/@v/v1.zip' --header 'Bearer <API_KEY>' -J -o '<OUTPUT_FILE_NAME>'",
		},
		{
			name: "huggingface", packageType: artifact.PackageTypeHUGGINGFACE, artifactName: "hf", version: "main",
			path: "/hf/main/dir/model/w.bin", wantName: "dir/w.bin",
			wantCmd: "curl --location 'https://r/model/hf/resolve/main/dir/w.bin' --header 'Bearer <API_KEY>' " +
				"-J -o '<OUTPUT_FILE_NAME>'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files := []types.FileNodeMetadata{file(tt.path)}
			got := GetArtifactFilesMetadata(ctx, &files, "https://r", tt.artifactName, tt.version,
				tt.packageType, "Bearer", &hfType, false, "root", "reg", rmdPkgWrapper{})
			require.Len(t, got, 1)
			require.Equal(t, tt.wantName, got[0].Name)
			require.Equal(t, tt.wantCmd, got[0].DownloadCommand)
			require.Equal(t, tt.path, got[0].Path)
			require.Equal(t, GetSize(1024), got[0].Size)
			require.Equal(t, []string{"SHA-512: s512", "SHA-256: s256", "SHA-1: s1", "MD5: md5"}, got[0].Checksums)
		})
	}

	t.Run("anonymous_has_no_auth_header", func(t *testing.T) {
		files := []types.FileNodeMetadata{file("/g/1/f")}
		got := GetArtifactFilesMetadata(ctx, &files, "https://r", "g", "1",
			artifact.PackageTypeGENERIC, "Bearer", nil, true, "root", "reg", rmdPkgWrapper{})
		require.Equal(t, "curl --location 'https://r/g/1/f' -J -o '<OUTPUT_FILE_NAME>'", got[0].DownloadCommand)
	})

	t.Run("default_type_uses_wrapper_metadata", func(t *testing.T) {
		files := []types.FileNodeMetadata{file("/c/1/f")}
		got := GetArtifactFilesMetadata(ctx, &files, "https://r", "c", "1", artifact.PackageTypeCARGO,
			"Bearer", nil, false, "root", "reg",
			rmdPkgWrapper{fileMetadata: &artifact.FileDetail{Name: "wrapped"}})
		require.Len(t, got, 1)
		require.Equal(t, "wrapped", got[0].Name)
	})

	t.Run("default_type_without_wrapper_metadata_falls_back", func(t *testing.T) {
		files := []types.FileNodeMetadata{file("/c/1/f")}
		got := GetArtifactFilesMetadata(ctx, &files, "https://r", "c", "1", artifact.PackageTypeCARGO,
			"Bearer", nil, false, "root", "reg", rmdPkgWrapper{})
		require.Len(t, got, 1)
		require.Equal(t, "f", got[0].Name)
		require.Empty(t, got[0].DownloadCommand)
	})
}

func TestRmdGetAllArtifactFilesResponse(t *testing.T) {
	ctx := rmdAdminCtx()
	t.Run("empty_files", func(t *testing.T) {
		resp := GetAllArtifactFilesResponse(ctx, nil, 0, 1, 10, "https://r", "a", "1",
			artifact.PackageTypeGENERIC, "Bearer", nil, false, "root", "reg", rmdPkgWrapper{})
		require.NotNil(t, resp.Data.Files)
		require.Empty(t, resp.Data.Files)
		require.Equal(t, artifact.StatusSUCCESS, resp.Status)
	})
	t.Run("with_files", func(t *testing.T) {
		files := []types.FileNodeMetadata{{Path: "/a/1/x", Size: 1}, {Path: "/a/1/y", Size: 2}}
		resp := GetAllArtifactFilesResponse(ctx, &files, 12, 3, 5, "https://r", "a", "1",
			artifact.PackageTypeGENERIC, "Bearer", nil, false, "root", "reg", rmdPkgWrapper{})
		require.Len(t, resp.Data.Files, 2)
		require.Equal(t, int64(3), *resp.Data.PageCount)
		require.Equal(t, int64(3), *resp.Data.PageIndex)
	})
}

func TestRmdGetArtifactFileResponseJSONResponse(t *testing.T) {
	tests := []struct {
		name string
		pt   artifact.PackageType
		want string
	}{
		{"generic", artifact.PackageTypeGENERIC, "https://r/a/1?filename=f.txt"},
		{"maven", artifact.PackageTypeMAVEN, "https://r/a/1/f.txt"},
		{"npm", artifact.PackageTypeNPM, "https://r/a/1/f.txt"},
		{"python", artifact.PackageTypePYTHON, "https://r/a/1/f.txt"},
		{"cargo", artifact.PackageTypeCARGO, "https://r/a/1/f.txt"},
		{"docker_unsupported", artifact.PackageTypeDOCKER, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := GetArtifactFileResponseJSONResponse("https://r", tt.pt, "a", "1", "f.txt")
			require.Equal(t, artifact.StatusSUCCESS, resp.Status)
			require.Equal(t, tt.want, resp.DownloadUrl)
		})
	}
}

func TestRmdGetQuarantinePathJSONResponse(t *testing.T) {
	resp := GetQuarantinePathJSONResponse("id1", 1, 2, 3, "reason", "/p/f")
	require.Equal(t, artifact.StatusSUCCESS, resp.Status)
	require.Equal(t, "id1", resp.Data.Id)
	require.Equal(t, int64(1), resp.Data.RegistryId)
	require.Equal(t, int64(2), resp.Data.ArtifactId)
	require.Equal(t, int64(3), *resp.Data.VersionId)
	require.Equal(t, "/p/f", *resp.Data.FilePath)
	require.Equal(t, "reason", resp.Data.Reason)
}

func TestRmdGetAllArtifactLabelsResponse(t *testing.T) {
	labels := []string{"x", "y"}
	resp := GetAllArtifactLabelsResponse(&labels, 21, 2, 10)
	require.Equal(t, labels, resp.Data.Labels)
	require.Equal(t, int64(3), *resp.Data.PageCount)
	require.Equal(t, int64(21), *resp.Data.ItemCount)
	require.Equal(t, int64(2), *resp.Data.PageIndex)
	require.Equal(t, 10, *resp.Data.PageSize)
}

func TestRmdGetDockerArtifactDetailsMapper(t *testing.T) {
	reg := &types.Registry{Name: "dockerreg", PackageType: artifact.PackageTypeDOCKER}
	reason := "why"
	resp := GetDockerArtifactDetails(reg, "img", "v1", rmdTime(), rmdTime(), "sha256:abc", 2048,
		"https://reg.example.com/dockerreg", 7, true, &reason)
	require.Equal(t, artifact.StatusSUCCESS, resp.Status)
	d := resp.Data
	require.Equal(t, "img", d.ImageName)
	require.Equal(t, "v1", d.Version)
	require.Equal(t, "dockerreg/img/sha256:abc", d.RegistryPath)
	require.Equal(t, "https://reg.example.com/dockerreg/img/v1", d.Url)
	require.Equal(t, GetSize(2048), *d.Size)
	require.Equal(t, int64(7), *d.DownloadsCount)
	require.True(t, *d.IsQuarantined)
	require.Equal(t, &reason, d.QuarantineReason)
	require.Equal(t, "1700000000000", *d.CreatedAt)
}

func TestRmdGetHelmArtifactDetailsMapper(t *testing.T) {
	reg := &types.Registry{Name: "helmreg", PackageType: artifact.PackageTypeHELM}
	manifest := &types.Manifest{Digest: digest.Digest("sha256:def"), TotalSize: 4096}
	resp := GetHelmArtifactDetails(reg, "chart", "0.1.0", rmdTime(), rmdTime(), manifest,
		"https://reg.example.com/helmreg")
	require.Equal(t, artifact.StatusSUCCESS, resp.Status)
	require.Equal(t, "chart", *resp.Data.Artifact)
	require.Equal(t, "helmreg/chart/sha256:def", resp.Data.RegistryPath)
	require.Equal(t, GetSize(4096), *resp.Data.Size)
	require.Equal(t, artifact.PackageTypeHELM, resp.Data.PackageType)
}

func TestRmdGetPackageSpecificArtifactDetails(t *testing.T) {
	image := &types.Image{Name: "pkg"}
	hfType := artifact.ArtifactTypeDataset
	hfImage := &types.Image{Name: "pkg", ArtifactType: &hfType}
	art := &types.Artifact{Version: "1.0", CreatedAt: rmdTime(), UpdatedAt: rmdTime()}

	t.Run("generic", func(t *testing.T) {
		got := GetGenericArtifactDetail(image, art, registrymetadata.GenericMetadata{Description: "desc"})
		require.Equal(t, "pkg", *got.Name)
		cfg, err := got.AsGenericArtifactDetailConfig()
		require.NoError(t, err)
		require.Equal(t, "desc", *cfg.Description)
	})

	t.Run("python", func(t *testing.T) {
		got := GetPythonArtifactDetail(image, art, map[string]any{"k": "v"})
		cfg, err := got.AsPythonArtifactDetailConfig()
		require.NoError(t, err)
		require.Equal(t, "v", (*cfg.Metadata)["k"])
	})

	t.Run("nuget_with_and_without_size", func(t *testing.T) {
		got := GetNugetArtifactDetail(image, art, map[string]any{"size": float64(1024)}, 4)
		require.Equal(t, GetSize(1024), *got.Size)
		require.Equal(t, int64(4), *got.DownloadCount)
		missing := GetNugetArtifactDetail(image, art, map[string]any{}, 0)
		require.Equal(t, GetSize(0), *missing.Size)
	})

	t.Run("huggingface", func(t *testing.T) {
		got := GetHFArtifactDetail(hfImage, art, map[string]any{"size": float64(2048)}, 2)
		require.Equal(t, GetSize(2048), *got.Size)
		require.Equal(t, &hfType, got.ArtifactType)
		missing := GetHFArtifactDetail(hfImage, art, nil, 2)
		require.Equal(t, GetSize(0), *missing.Size)
	})

	t.Run("go", func(t *testing.T) {
		got := GetGoArtifactDetail(image, art, map[string]any{"size": float64(512)}, 1)
		require.Equal(t, GetSize(512), *got.Size)
		missing := GetGoArtifactDetail(image, art, map[string]any{}, 1)
		require.Equal(t, GetSize(0), *missing.Size)
	})

	t.Run("npm", func(t *testing.T) {
		npmArt := &types.Artifact{
			Version: "1.0", CreatedAt: rmdTime(), UpdatedAt: rmdTime(),
			Metadata: json.RawMessage(`{"size":321}`),
		}
		got := GetNPMArtifactDetail(image, npmArt, map[string]any{"a": "b"}, 3)
		require.Equal(t, "321", *got.Size)
		require.Equal(t, int64(3), *got.DownloadCount)

		bad := &types.Artifact{Version: "1.0", Metadata: json.RawMessage(`not-json`)}
		require.Equal(t, artifact.ArtifactDetail{}, GetNPMArtifactDetail(image, bad, nil, 0))
	})

	t.Run("rpm_strips_file_listings", func(t *testing.T) {
		meta := map[string]any{
			"size": float64(100),
			"file_metadata": map[string]any{
				"files": []any{"a"}, "changelogs": []any{"b"}, "keep": "me",
			},
		}
		got := GetRPMArtifactDetail(image, art, meta, 6)
		require.Equal(t, "100", *got.Size)
		fm, ok := meta["file_metadata"].(map[string]any)
		require.True(t, ok)
		require.NotContains(t, fm, "files")
		require.NotContains(t, fm, "changelogs")
		require.Contains(t, fm, "keep")

		noSize := GetRPMArtifactDetail(image, art, map[string]any{}, 0)
		require.Equal(t, "0", *noSize.Size)
	})
}

func TestRmdGetArtifactSummaryMapper(t *testing.T) {
	deleted := rmdTime()
	hfType := artifact.ArtifactTypeModel
	resp := GetArtifactSummary(types.ImageMetadata{
		Name: "img", UUID: "u", RegistryUUID: "ru", DownloadCount: 9, PackageType: artifact.PackageTypeHUGGINGFACE,
		ArtifactType: &hfType, CreatedAt: rmdTime(), ModifiedAt: rmdTime(), DeletedAt: &deleted,
	})
	require.Equal(t, artifact.StatusSUCCESS, resp.Status)
	require.Equal(t, "img", resp.Data.ImageName)
	require.Equal(t, int64(9), *resp.Data.DownloadsCount)
	require.Equal(t, "1700000000000", *resp.Data.DeletedAt)
	require.Equal(t, "ru", resp.Data.RegistryUUID)

	noDelete := GetArtifactSummary(types.ImageMetadata{Name: "x"})
	require.Nil(t, noDelete.Data.DeletedAt)
}

func TestRmdGetArtifactVersionSummaryMapper(t *testing.T) {
	deleted := rmdTime()
	hfType := artifact.ArtifactTypeModel
	resp := GetArtifactVersionSummary("img", artifact.PackageTypeDOCKER, "v1", true, "reason", &hfType,
		&deleted, "au", "ru")
	require.Equal(t, artifact.StatusSUCCESS, resp.Status)
	require.Equal(t, "img", resp.Data.ImageName)
	require.Equal(t, "v1", resp.Data.Version)
	require.True(t, *resp.Data.IsQuarantined)
	require.Equal(t, "reason", *resp.Data.QuarantineReason)
	require.Equal(t, "1700000000000", *resp.Data.DeletedAt)
	require.Equal(t, "au", resp.Data.Uuid)

	noDelete := GetArtifactVersionSummary("img", artifact.PackageTypeDOCKER, "v1", false, "", nil, nil, "", "")
	require.Nil(t, noDelete.Data.DeletedAt)
}

func TestRmdGetRepoPath(t *testing.T) {
	require.Equal(t, "reg/img/tag", getRepoPath("reg", "img", "tag"))
}
