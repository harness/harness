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
	"context"
	"errors"
	"testing"
	"time"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/types"
	coretypes "github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

func rmdStrPtr(s string) *string { return &s }

func rmdInt64Ptr(i int64) *int64 { return &i }

func TestRmdValidators(t *testing.T) {
	t.Run("scope", func(t *testing.T) {
		require.NoError(t, ValidateScope(""))
		require.NoError(t, ValidateScope(string(artifact.Ancestors)))
		require.NoError(t, ValidateScope(string(artifact.Descendants)))
		require.NoError(t, ValidateScope(string(artifact.None)))
		require.Error(t, ValidateScope("sideways"))
		require.True(t, IsScopeValid("none"))
		require.False(t, IsScopeValid("sideways"))
	})

	t.Run("artifact_type", func(t *testing.T) {
		got, err := ValidateAndGetArtifactType(artifact.PackageTypeHUGGINGFACE, "model")
		require.NoError(t, err)
		require.Equal(t, artifact.ArtifactTypeModel, *got)

		got, err = ValidateAndGetArtifactType(artifact.PackageTypeHUGGINGFACE, "dataset")
		require.NoError(t, err)
		require.Equal(t, artifact.ArtifactTypeDataset, *got)

		_, err = ValidateAndGetArtifactType(artifact.PackageTypeHUGGINGFACE, "space")
		require.Error(t, err)

		_, err = ValidateAndGetArtifactType(artifact.PackageTypeNPM, "model")
		require.Error(t, err)
	})

	changeTests := []struct {
		name    string
		fn      func(string, string) error
		from    string
		to      string
		wantErr bool
	}{
		{"package_same", ValidatePackageTypeChange, "NPM", "NPM", false},
		{"package_changed", ValidatePackageTypeChange, "NPM", "DOCKER", true},
		{"package_empty_db", ValidatePackageTypeChange, "", "NPM", true},
		{"package_empty_new", ValidatePackageTypeChange, "NPM", "", true},
		{"repo_same", ValidateRepoTypeChange, "VIRTUAL", "VIRTUAL", false},
		{"repo_changed", ValidateRepoTypeChange, "VIRTUAL", "UPSTREAM", true},
		{"repo_empty", ValidateRepoTypeChange, "", "", true},
		{"identifier_same", ValidateIdentifierChange, "reg", "reg", false},
		{"identifier_changed", ValidateIdentifierChange, "reg", "other", true},
		{"identifier_empty", ValidateIdentifierChange, "reg", "", true},
	}
	for _, tt := range changeTests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn(tt.from, tt.to)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestRmdValidateIdentifierPatterns(t *testing.T) {
	tests := []struct {
		id      string
		wantErr bool
	}{
		{"abc", false},
		{"a1-b2_c3.d4", false},
		{"", true},
		{"UPPER", true},
		{"-lead", true},
		{"trail-", true},
		{"dou--ble", true},
		{"has space", true},
	}
	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			err := ValidateIdentifier(tt.id)
			if tt.wantErr {
				require.EqualError(t, err, RegistryIdentifierErrorMsg)
				return
			}
			require.NoError(t, err)
		})
	}
}

func rmdUpstreamConfigBody(t *testing.T, cfg artifact.UpstreamConfig) *artifact.RegistryConfig {
	t.Helper()
	rc := &artifact.RegistryConfig{}
	require.NoError(t, rc.FromUpstreamConfig(cfg))
	return rc
}

func rmdUserPasswordAuth(t *testing.T, up artifact.UserPassword) *artifact.UpstreamConfig_Auth {
	t.Helper()
	auth := &artifact.UpstreamConfig_Auth{}
	require.NoError(t, auth.FromUserPassword(up))
	return auth
}

func rmdAccessKeyAuth(t *testing.T, ak artifact.AccessKeySecretKey) *artifact.UpstreamConfig_Auth {
	t.Helper()
	auth := &artifact.UpstreamConfig_Auth{}
	require.NoError(t, auth.FromAccessKeySecretKey(ak))
	return auth
}

func TestRmdValidateUpstream(t *testing.T) {
	src := artifact.UpstreamConfigSourceCustom
	tests := []struct {
		name    string
		wrapper rmdPkgWrapper
		cfg     *artifact.RegistryConfig
		wantMsg string
	}{
		{
			name:    "virtual_registry_skips_checks",
			wrapper: rmdPkgWrapper{urlRequired: true},
			cfg: func() *artifact.RegistryConfig {
				rc := &artifact.RegistryConfig{}
				require.NoError(t, rc.FromVirtualConfig(artifact.VirtualConfig{}))
				return rc
			}(),
		},
		{
			name:    "url_required_but_missing",
			wrapper: rmdPkgWrapper{urlRequired: true},
			cfg: rmdUpstreamConfigBody(t, artifact.UpstreamConfig{
				AuthType: artifact.AuthTypeAnonymous, Source: &src,
			}),
			wantMsg: "URL is required for upstream repository",
		},
		{
			name:    "url_required_and_present",
			wrapper: rmdPkgWrapper{urlRequired: true},
			cfg: rmdUpstreamConfigBody(t, artifact.UpstreamConfig{
				AuthType: artifact.AuthTypeAnonymous, Source: &src, Url: rmdStrPtr("https://up.example.com"),
			}),
		},
		{
			name: "nil_source_is_accepted",
			cfg: rmdUpstreamConfigBody(t, artifact.UpstreamConfig{
				AuthType: artifact.AuthTypeUserPassword,
			}),
		},
		{
			name: "unsupported_auth_type",
			cfg: rmdUpstreamConfigBody(t, artifact.UpstreamConfig{
				AuthType: artifact.AuthType("Kerberos"),
			}),
			wantMsg: "Upstream auth type must be one of",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUpstream(tt.wrapper, artifact.PackageTypeNPM, tt.cfg)
			if tt.wantMsg == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.wantMsg)
		})
	}
}

func TestRmdSortingAndPaging(t *testing.T) {
	t.Run("sort_by_field", func(t *testing.T) {
		tests := []struct {
			field, resource, want string
		}{
			{"identifier", RepositoryResource, "name"},
			{"registrySize", RepositoryResource, "size"},
			{"bogus", RepositoryResource, "created_at"},
			{"name", ArtifactResource, "image_name"},
			{"downloadsCount", ArtifactResource, "download_count"},
			{"size", ArtifactVersionResource, "name"},
			{"lastModified", ArtifactVersionResource, "updated_at"},
			{"size", ArtifactFilesResource, "size"},
			{"bogus", ArtifactFilesResource, "created_at"},
			{"name", "unknown-resource", "created_at"},
		}
		for _, tt := range tests {
			require.Equal(t, tt.want, GetSortByField(tt.field, tt.resource), tt.field+"/"+tt.resource)
		}
	})

	t.Run("sort_order", func(t *testing.T) {
		require.Equal(t, "ASC", GetSortByOrder(""))
		require.Equal(t, "ASC", GetSortByOrder("sideways"))
		require.Equal(t, "ASC", GetSortByOrder("ASC"))
		require.Equal(t, "DESC", GetSortByOrder("DESC"))
	})

	t.Run("page_limit_offset_number", func(t *testing.T) {
		size := artifact.PageSize(25)
		page := artifact.PageNumber(3)
		zero := artifact.PageNumber(0)
		require.Equal(t, 10, GetPageLimit(nil))
		require.Equal(t, 25, GetPageLimit(&size))
		require.Equal(t, 0, GetOffset(nil, nil))
		require.Equal(t, 0, GetOffset(&size, nil))
		require.Equal(t, 0, GetOffset(nil, &page))
		require.Equal(t, 0, GetOffset(&size, &zero))
		require.Equal(t, 75, GetOffset(&size, &page))
		require.Equal(t, int64(1), GetPageNumber(nil))
		require.Equal(t, int64(3), GetPageNumber(&page))
	})

	t.Run("page_count", func(t *testing.T) {
		require.Equal(t, int64(0), GetPageCount(0, 10))
		require.Equal(t, int64(1), GetPageCount(10, 10))
		require.Equal(t, int64(2), GetPageCount(11, 10))
	})
}

func TestRmdSizeAndURLHelpers(t *testing.T) {
	require.Contains(t, GetSize(1024), "KB")
	require.Equal(t, GetSize(0), GetImageSize("not-a-number"))
	require.Equal(t, GetSize(2048), GetImageSize("2048"))
	require.Equal(t, "root/reg", GetRegistryRef("root", "reg"))
	require.Equal(t, "1700000000000", GetTimeInMs(time.UnixMilli(1700000000000)))
	require.Equal(t, "reg.example.com/a/b", GetRepoURLWithoutProtocol("https://reg.example.com/a/b"))
	require.Equal(t, "", GetRepoURLWithoutProtocol("http://bad host\x7f/"))
	require.Equal(t, "https://r/img/v1", GetTagURL("img", "v1", "https://r"))

	e := GetErrorResponse(404, "nope")
	require.Equal(t, "404", e.Code)
	require.Equal(t, "nope", e.Message)
	require.Equal(t, artifact.StatusSUCCESS, GetSuccessResponse().Status)
}

func TestRmdCleanURLPath(t *testing.T) {
	require.NotPanics(t, func() { CleanURLPath(nil) })
	tests := []struct{ in, want string }{
		{"https://a.example.com/path/", "https://a.example.com/path"},
		{"https://a.example.com/path///", "https://a.example.com/path"},
		{"https://a.example.com", "https://a.example.com"},
		{"https://a.example.com/x?y=1", "https://a.example.com/x?y=1"},
	}
	for _, tt := range tests {
		in := tt.in
		CleanURLPath(&in)
		require.Equal(t, tt.want, in)
	}
	bad := "http://[::1"
	CleanURLPath(&bad)
	require.Equal(t, "http://[::1", bad, "unparseable input is left untouched")
}

func TestRmdDownloadCommands(t *testing.T) {
	require.Equal(t, "docker pull h/p/img:v1", GetDockerPullCommand("img", "v1", "https://h/p", true))
	require.Equal(t, "docker pull h/p/img@sha", GetDockerPullCommand("img", "sha", "https://h/p", false))
	require.Equal(t, "helm pull oci://h/p/c --version 1", GetHelmPullCommand("c", "1", "https://h/p", true))
	require.Equal(t, "helm pull oci://h/p/c@1", GetHelmPullCommand("c", "1", "https://h/p", false))
	require.Equal(t, "yum install a-1", GetRPMDownloadCommand("a", "1"))
	require.Equal(t, "nuget install a -Version 1 -Source <SOURCE>", GetNugetDownloadCommand("a", "1"))
	require.Equal(t, "cargo add a@1 --registry <REGISTRY>", GetCargoDownloadCommand("a", "1"))
	require.Equal(t, "go get a@1", GetGoDownloadCommand("a", "1"))
	require.Equal(t, "npm install a@1 ", GetNPMDownloadCommand("a", "1"))
	require.Equal(t, "pip install a==1 ", GetPythonDownloadCommand("a", "1"))

	const out = " -J -o '<OUTPUT_FILE_NAME>'"
	const hdr = " --header 'Bearer <API_KEY>'"
	require.Equal(t, "curl --location 'https://h/files/a/1/f'"+hdr+out,
		GetGenericFileDownloadCommand("https://h", "a", "1", "f", "Bearer", false))
	require.Equal(t, "curl --location 'https://h/files/a/1/f'"+out,
		GetGenericFileDownloadCommand("https://h", "a", "1", "f", "Bearer", true))
	require.Equal(t, "curl --location 'https://h/a/1/f'"+hdr+out,
		GetGenericArtifactFileDownloadCommand("https://h", "a", "1", "f", "Bearer", false))
	require.Equal(t, "curl --location 'https://h/a/-/f' -J -o '<OUTPUT_FILE_NAME>'",
		GetNPMArtifactFileDownloadCommand("https://h", "a", "1", "f", true))
	require.Equal(t, "curl --location 'https://h/package/a/1/f'"+hdr+out,
		GetNugetArtifactFileDownloadCommand("https://h", "a", "1", "f", "Bearer", false))
	require.Equal(t, "curl --location 'https://h/api/v1/crates/a/1/download'"+hdr+out,
		GetCargoArtifactFileDownloadCommand("https://h", "a", "1", "Bearer", false))
	require.Equal(t, "curl --location 'https://h/a/@v/f'"+out,
		GetGoArtifactFileDownloadCommand("https://h", "a", "f", "Bearer", true))
	require.Equal(t, "curl --location 'https://h/a/1/f'"+hdr+" -o '<OUTPUT_FILE_NAME>'",
		GetMavenArtifactFileDownloadCommand("https://h", "a", "1", "f", "Bearer", false))

	ds := artifact.ArtifactTypeDataset
	require.Equal(t, "curl --location 'https://h/dataset/a/resolve/main/f'"+hdr+out,
		GetHuggingFaceArtifactFileDownloadCommand("https://h", "a", "main", "f", "Bearer", &ds, false))

	require.Equal(t, "curl --location 'https://h/package/a/1.0/x86_64/f.rpm'"+hdr+out,
		GetRPMArtifactFileDownloadCommand("https://h", "a", "1.0.x86_64", "dir/f.rpm", "Bearer", false))
}

func TestRmdGetPullCommand(t *testing.T) {
	ds := artifact.ArtifactTypeDataset
	tests := []struct {
		name string
		pt   artifact.PackageType
		want string
	}{
		{"docker", artifact.PackageTypeDOCKER, "docker pull h/p/img:v1"},
		{"helm", artifact.PackageTypeHELM, "helm pull oci://h/p/img --version v1"},
		{"python", artifact.PackageTypePYTHON, "pip install img==v1 "},
		{"npm", artifact.PackageTypeNPM, "npm install img@v1 "},
		{"rpm", artifact.PackageTypeRPM, "yum install img-v1"},
		{"nuget", artifact.PackageTypeNUGET, "nuget install img -Version v1 -Source <SOURCE>"},
		{"go", artifact.PackageTypeGO, "go get img@v1"},
		{"maven_has_no_pull_command", artifact.PackageTypeMAVEN, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, GetPullCommand(rmdAdminCtx(), "img", "v1", string(tt.pt),
				"https://h/p", "Bearer", nil, true))
		})
	}

	t.Run("generic_depends_on_session", func(t *testing.T) {
		named := GetPullCommand(rmdAdminCtx(), "img", "v1", string(artifact.PackageTypeGENERIC),
			"https://h/p", "Bearer", nil, true)
		anon := GetPullCommand(rmdAnonCtx(), "img", "v1", string(artifact.PackageTypeGENERIC),
			"https://h/p", "Bearer", nil, true)
		require.Contains(t, named, "--header")
		require.NotContains(t, anon, "--header")
	})

	t.Run("huggingface", func(t *testing.T) {
		got := GetPullCommand(rmdAdminCtx(), "img", "v1", string(artifact.PackageTypeHUGGINGFACE),
			"https://h/p", "Bearer", &ds, true)
		require.Contains(t, got, "/dataset/img/resolve/v1/<FILENAME>")
	})
}

func TestRmdHandleAuthError(t *testing.T) {
	code, msg := HandleAuthError(rmdErrUnauthorized())
	require.Equal(t, 401, code)
	require.NotEmpty(t, msg)

	code, _ = HandleAuthError(rmdErrForbidden())
	require.Equal(t, 403, code)

	code, msg = HandleAuthError(errors.New("boom"))
	require.Equal(t, 403, code)
	require.Equal(t, "boom", msg)
}

func TestRmdCleanupPolicyMapper(t *testing.T) {
	validPolicy := func() artifact.CleanupPolicy {
		return artifact.CleanupPolicy{
			Name:          rmdStrPtr("old"),
			ExpireDays:    func() *int { d := 2; return &d }(),
			VersionPrefix: &[]string{"v1"},
			PackagePrefix: &[]string{"pkg"},
		}
	}
	t.Run("nil_config_yields_empty", func(t *testing.T) {
		got, err := CreateCleanupPolicyEntity(nil, 1)
		require.NoError(t, err)
		require.Empty(t, *got)
		got, err = CreateCleanupPolicyEntity(&artifact.ModifyRegistryJSONRequestBody{}, 1)
		require.NoError(t, err)
		require.Empty(t, *got)
	})

	t.Run("valid_policy", func(t *testing.T) {
		body := &artifact.ModifyRegistryJSONRequestBody{CleanupPolicy: &[]artifact.CleanupPolicy{validPolicy()}}
		got, err := CreateCleanupPolicyEntity(body, 9)
		require.NoError(t, err)
		require.Len(t, *got, 1)
		p := (*got)[0]
		require.Equal(t, "old", p.Name)
		require.Equal(t, int64(9), p.RegistryID)
		require.Equal(t, []string{"v1"}, p.VersionPrefix)
		require.Equal(t, []string{"pkg"}, p.PackagePrefix)
		require.Equal(t, (48 * time.Hour).Milliseconds(), p.ExpiryTime)
	})

	missing := map[string]func(p *artifact.CleanupPolicy){
		"expireDays":    func(p *artifact.CleanupPolicy) { p.ExpireDays = nil },
		"name":          func(p *artifact.CleanupPolicy) { p.Name = nil },
		"versionPrefix": func(p *artifact.CleanupPolicy) { p.VersionPrefix = nil },
		"packagePrefix": func(p *artifact.CleanupPolicy) { p.PackagePrefix = nil },
	}
	for field, mutate := range missing {
		t.Run("missing_"+field, func(t *testing.T) {
			p := validPolicy()
			mutate(&p)
			body := &artifact.ModifyRegistryJSONRequestBody{CleanupPolicy: &[]artifact.CleanupPolicy{p}}
			got, err := CreateCleanupPolicyEntity(body, 1)
			require.Nil(t, got)
			require.Error(t, err)
			require.Contains(t, err.Error(), field)
			var ue *usererror.Error
			require.ErrorAs(t, err, &ue)
			require.Equal(t, 400, ue.Status)
		})
	}

	t.Run("response_mapping", func(t *testing.T) {
		got := CreateCleanupPolicyResponse(&[]types.CleanupPolicy{
			{Name: "a", VersionPrefix: []string{"v"}, PackagePrefix: []string{"p"}},
			{Name: "b"},
		})
		require.Len(t, *got, 2)
		require.Equal(t, "a", *(*got)[0].Name)
		require.Equal(t, []string{"v"}, *(*got)[0].VersionPrefix)
		require.Equal(t, []string{"p"}, *(*got)[0].PackagePrefix)
		require.Equal(t, "b", *(*got)[1].Name)
	})
}

func rmdBaseInfo() *types.RegistryRequestBaseInfo {
	return &types.RegistryRequestBaseInfo{
		RootIdentifier: "root", RootIdentifierID: 1, RegistryIdentifier: "reg", RegistryID: 5,
		ParentRef: "root", ParentID: 2,
	}
}

func TestRmdGetRegistryRequestInfo(t *testing.T) {
	ctrl := &APIController{RegistryMetadataHelper: rmdHelper{info: rmdBaseInfo()}}
	pkgs := artifact.PackageTypeParam{"NPM", "DOCKER"}
	ids := artifact.RegistryIdentifierParam{"r1"}
	labels := artifact.LabelsParam{"l1"}
	order := artifact.SortOrder("DESC")
	field := artifact.SortField("lastModified")
	page := artifact.PageNumber(2)
	size := artifact.PageSize(5)
	search := artifact.SearchTerm("term")
	recursive := artifact.RecursiveParam(true)
	scope := artifact.GetAllRegistriesParamsScope("descendants")

	t.Run("all_params", func(t *testing.T) {
		got, err := ctrl.GetRegistryRequestInfo(context.Background(), RegistryRequestParams{
			packageTypesParam: &pkgs, page: &page, size: &size, search: &search, Resource: RepositoryResource,
			ParentRef: "root", labelsParam: &labels, sortOrder: &order, sortField: &field,
			registryIDsParam: &ids, recursive: &recursive, scope: &scope,
		})
		require.NoError(t, err)
		require.Equal(t, []string{"NPM", "DOCKER"}, got.packageTypes)
		require.Equal(t, "updated_at", got.sortByField)
		require.Equal(t, "DESC", got.sortByOrder)
		require.Equal(t, 10, got.offset)
		require.Equal(t, 5, got.limit)
		require.Equal(t, int64(2), got.pageNumber)
		require.Equal(t, "term", got.searchTerm)
		require.Equal(t, []string{"l1"}, got.labels)
		require.Equal(t, []string{"r1"}, got.registryIDs)
		require.True(t, got.recursive)
		require.Equal(t, "descendants", got.scope)
		require.Equal(t, "reg", got.RegistryIdentifier)
	})

	t.Run("defaults", func(t *testing.T) {
		got, err := ctrl.GetRegistryRequestInfo(context.Background(), RegistryRequestParams{
			Resource: ArtifactResource,
		})
		require.NoError(t, err)
		require.Empty(t, got.packageTypes)
		require.Empty(t, got.labels)
		require.Empty(t, got.registryIDs)
		require.Equal(t, "ASC", got.sortByOrder)
		require.Equal(t, "created_at", got.sortByField)
		require.Equal(t, 10, got.limit)
		require.Equal(t, 0, got.offset)
		require.Equal(t, int64(1), got.pageNumber)
		require.False(t, got.recursive)
		require.Empty(t, got.scope)
	})

	t.Run("helper_error", func(t *testing.T) {
		bad := &APIController{RegistryMetadataHelper: rmdHelper{infoErr: errors.New("nope")}}
		got, err := bad.GetRegistryRequestInfo(context.Background(), RegistryRequestParams{})
		require.Nil(t, got)
		require.EqualError(t, err, "nope")
	})
}

func TestRmdGetArtifactFilesRequestInfo(t *testing.T) {
	ctrl := &APIController{RegistryMetadataHelper: rmdHelper{info: rmdBaseInfo()}}
	order := artifact.SortOrder("DESC")
	field := artifact.SortField("size")
	page := artifact.PageNumber(1)
	size := artifact.PageSize(20)
	search := artifact.SearchTerm("x")
	req := artifact.GetArtifactFilesRequestObject{
		RegistryRef: "root/reg",
		Params: artifact.GetArtifactFilesParams{
			SortOrder: &order, SortField: &field, Page: &page, Size: &size, SearchTerm: &search,
		},
	}
	got, err := ctrl.GetArtifactFilesRequestInfo(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, "size", got.sortByField)
	require.Equal(t, "DESC", got.sortByOrder)
	require.Equal(t, 20, got.offset)
	require.Equal(t, 20, got.limit)
	require.Equal(t, "x", got.searchTerm)

	def, err := ctrl.GetArtifactFilesRequestInfo(context.Background(), artifact.GetArtifactFilesRequestObject{})
	require.NoError(t, err)
	require.Equal(t, "created_at", def.sortByField)
	require.Equal(t, "ASC", def.sortByOrder)
	require.Equal(t, int64(1), def.pageNumber)

	bad := &APIController{RegistryMetadataHelper: rmdHelper{infoErr: errors.New("x")}}
	_, err = bad.GetArtifactFilesRequestInfo(context.Background(), req)
	require.Error(t, err)
}

func TestRmdGetRepoEntityFields(t *testing.T) {
	allowed, blocked, desc, labels := getRepoEntityFields(artifact.RegistryRequest{})
	require.Empty(t, allowed)
	require.Empty(t, blocked)
	require.Empty(t, desc)
	require.Empty(t, labels)
	require.NotNil(t, allowed)

	allowed, blocked, desc, labels = getRepoEntityFields(artifact.RegistryRequest{
		AllowedPattern: &[]string{"a"}, BlockedPattern: &[]string{"b"},
		Description: rmdStrPtr("d"), Labels: &[]string{"l"},
	})
	require.Equal(t, []string{"a"}, allowed)
	require.Equal(t, []string{"b"}, blocked)
	require.Equal(t, "d", desc)
	require.Equal(t, []string{"l"}, labels)
}

func TestRmdDeduplicateTriggers(t *testing.T) {
	require.Empty(t, deduplicateTriggers(nil))
	require.NotNil(t, deduplicateTriggers(nil))
	in := []enum.WebhookTrigger{
		enum.WebhookTriggerArtifactCreated, enum.WebhookTriggerArtifactDeleted,
		enum.WebhookTriggerArtifactCreated,
	}
	require.Equal(t, []enum.WebhookTrigger{
		enum.WebhookTriggerArtifactCreated, enum.WebhookTriggerArtifactDeleted,
	}, deduplicateTriggers(in))
}

func TestRmdBuildVirtualRepositoryResponse(t *testing.T) {
	deleted := time.UnixMilli(1700000000000)
	reg := &types.Registry{
		UUID: "u", Name: "virt", Description: "d", PackageType: artifact.PackageTypeNPM,
		AllowedPattern: []string{"a"}, BlockedPattern: []string{"b"}, Labels: []string{"l"},
		CreatedAt: deleted, UpdatedAt: deleted, DeletedAt: &deleted,
	}
	policies := []types.CleanupPolicy{{Name: "p"}}

	t.Run("success", func(t *testing.T) {
		c := &APIController{PublicAccess: rmdPublicAccess{isPublic: true}}
		resp, err := c.BuildVirtualRepositoryResponse(context.Background(), reg, []string{"up1", "up2"},
			&policies, "https://r/virt", "root/virt")
		require.NoError(t, err)
		require.Equal(t, artifact.StatusSUCCESS, resp.Status)
		require.Equal(t, "virt", resp.Data.Identifier)
		require.True(t, resp.Data.IsPublic)
		require.Equal(t, "1700000000000", *resp.Data.DeletedAt)
		require.Len(t, *resp.Data.CleanupPolicy, 1)
		cfg, err := resp.Data.Config.AsVirtualConfig()
		require.NoError(t, err)
		require.Equal(t, []string{"up1", "up2"}, *cfg.UpstreamProxies)
	})

	t.Run("public_access_error", func(t *testing.T) {
		c := &APIController{PublicAccess: rmdPublicAccess{getErr: errors.New("pa")}}
		resp, err := c.BuildVirtualRepositoryResponse(context.Background(), reg, nil, &policies, "u", "root/virt")
		require.Nil(t, resp)
		require.ErrorContains(t, err, "root/virt")
	})
}

func TestRmdBuildUpstreamProxyResponse(t *testing.T) {
	mode := "BLOCK"
	base := func() *types.UpstreamProxy {
		return &types.UpstreamProxy{
			RegistryUUID: "u", RepoKey: "up", Source: "Dockerhub", RepoURL: "https://hub",
			PackageType:    artifact.PackageTypeDOCKER,
			AllowedPattern: []string{"a"}, BlockedPattern: []string{"b"}, Labels: []string{"l"},
			CreatedAt: time.UnixMilli(1), UpdatedAt: time.UnixMilli(2),
		}
	}
	c := &APIController{PublicAccess: rmdPublicAccess{}}

	t.Run("user_password", func(t *testing.T) {
		p := base()
		p.RepoAuthType = string(artifact.AuthTypeUserPassword)
		p.UserName = "bob"
		p.SecretIdentifier = "sec"
		p.SecretSpacePath = "root/space"
		p.FirewallMode = &mode
		p.Config = &types.RegistryConfig{RemoteUrlSuffix: "/simple"}
		resp, err := c.BuildUpstreamProxyResponse(context.Background(), p, "root/up")
		require.NoError(t, err)
		cfg, err := resp.Data.Config.AsUpstreamConfig()
		require.NoError(t, err)
		require.Equal(t, artifact.AuthTypeUserPassword, cfg.AuthType)
		up, err := cfg.Auth.AsUserPassword()
		require.NoError(t, err)
		require.Equal(t, "bob", up.UserName)
		require.Equal(t, "sec", *up.SecretIdentifier)
		require.Equal(t, "root/space", *up.SecretSpacePath)
		require.Equal(t, "/simple", *cfg.RemoteUrlSuffix)
		require.Equal(t, "BLOCK", string(*cfg.UpstreamProxyConfigFirewallMode))
		require.Equal(t, "https://hub", *cfg.Url)
	})

	t.Run("access_key", func(t *testing.T) {
		p := base()
		p.RepoAuthType = string(artifact.AuthTypeAccessKeySecretKey)
		p.UserName = "AKIA"
		p.UserNameSecretIdentifier = "aksec"
		p.SecretIdentifier = "sk"
		resp, err := c.BuildUpstreamProxyResponse(context.Background(), p, "root/up")
		require.NoError(t, err)
		cfg, err := resp.Data.Config.AsUpstreamConfig()
		require.NoError(t, err)
		ak, err := cfg.Auth.AsAccessKeySecretKey()
		require.NoError(t, err)
		require.Equal(t, "AKIA", *ak.AccessKey)
		require.Equal(t, "aksec", *ak.AccessKeySecretIdentifier)
		require.Equal(t, "sk", ak.SecretKeyIdentifier)
		require.Nil(t, cfg.RemoteUrlSuffix)
	})

	t.Run("anonymous_with_deleted_at", func(t *testing.T) {
		p := base()
		p.RepoAuthType = string(artifact.AuthTypeAnonymous)
		d := time.UnixMilli(1700000000000)
		p.DeletedAt = &d
		resp, err := c.BuildUpstreamProxyResponse(context.Background(), p, "root/up")
		require.NoError(t, err)
		require.Equal(t, "1700000000000", *resp.Data.DeletedAt)
		require.Equal(t, "up", resp.Data.Identifier)
	})

	t.Run("public_access_error", func(t *testing.T) {
		bad := &APIController{PublicAccess: rmdPublicAccess{getErr: errors.New("pa")}}
		resp, err := bad.BuildUpstreamProxyResponse(context.Background(), base(), "root/up")
		require.Nil(t, resp)
		require.Error(t, err)
	})
}

func TestRmdSetUpstreamProxyIDs(t *testing.T) {
	virtualDTO := func(upstreams ...string) artifact.RegistryRequest {
		rc := &artifact.RegistryConfig{}
		require.NoError(t, rc.FromVirtualConfig(artifact.VirtualConfig{UpstreamProxies: &upstreams}))
		return artifact.RegistryRequest{Identifier: "virt", Config: rc}
	}
	metas := &[]store.RegistryMetadata{
		{RegIdentifier: "up1", RegID: "11"},
		{RegIdentifier: "up2", RegID: "12"},
		{RegIdentifier: "broken", RegID: "not-a-number"},
	}
	newCtrl := func(repo *rmdRegRepo) *APIController {
		return &APIController{
			SpaceStore:         rmdSpaceStore{ancestors: []int64{1, 2}},
			RegistryRepository: repo,
		}
	}

	t.Run("non_virtual_rejected", func(t *testing.T) {
		dto := artifact.RegistryRequest{Config: rmdUpstreamConfigBody(t, artifact.UpstreamConfig{})}
		err := newCtrl(&rmdRegRepo{}).setUpstreamProxyIDs(context.Background(), &types.Registry{}, dto, 2)
		require.ErrorContains(t, err, "invalid call")
	})

	t.Run("no_upstreams_is_noop", func(t *testing.T) {
		reg := &types.Registry{UpstreamProxies: []int64{99}}
		err := newCtrl(&rmdRegRepo{}).setUpstreamProxyIDs(context.Background(), reg, virtualDTO(), 2)
		require.NoError(t, err)
		require.Equal(t, []int64{99}, reg.UpstreamProxies)
	})

	t.Run("resolves_matching_ids", func(t *testing.T) {
		reg := &types.Registry{ID: 100, Name: "virt", PackageType: artifact.PackageTypeNPM}
		repo := &rmdRegRepo{all: metas, byID: map[int64]*types.Registry{}}
		err := newCtrl(repo).setUpstreamProxyIDs(context.Background(), reg, virtualDTO("up2", "up1", "broken", "x"), 2)
		require.NoError(t, err)
		require.Equal(t, []int64{12, 11}, reg.UpstreamProxies)
	})

	t.Run("ancestor_lookup_error", func(t *testing.T) {
		c := &APIController{SpaceStore: rmdSpaceStore{err: errors.New("db")}, RegistryRepository: &rmdRegRepo{}}
		err := c.setUpstreamProxyIDs(context.Background(), &types.Registry{}, virtualDTO("up1"), 2)
		require.ErrorContains(t, err, "ancestors")
	})

	t.Run("repository_lookup_error", func(t *testing.T) {
		repo := &rmdRegRepo{allErr: errors.New("db")}
		err := newCtrl(repo).setUpstreamProxyIDs(context.Background(), &types.Registry{}, virtualDTO("up1"), 2)
		require.ErrorContains(t, err, "no repositories found")
	})

	t.Run("no_repositories_returned", func(t *testing.T) {
		err := newCtrl(&rmdRegRepo{}).setUpstreamProxyIDs(context.Background(), &types.Registry{}, virtualDTO("up1"), 2)
		require.ErrorContains(t, err, "no repositories found")
	})

	t.Run("self_reference_is_a_cycle", func(t *testing.T) {
		reg := &types.Registry{ID: 11, Name: "virt"}
		err := newCtrl(&rmdRegRepo{all: metas}).setUpstreamProxyIDs(context.Background(), reg, virtualDTO("up1"), 2)
		require.ErrorContains(t, err, "cycle detected")
	})

	t.Run("transitive_cycle", func(t *testing.T) {
		reg := &types.Registry{ID: 100, Name: "virt"}
		repo := &rmdRegRepo{
			all:  metas,
			byID: map[int64]*types.Registry{11: {ID: 11, UpstreamProxies: []int64{100}}},
		}
		err := newCtrl(repo).setUpstreamProxyIDs(context.Background(), reg, virtualDTO("up1"), 2)
		require.ErrorContains(t, err, "cycle detected")
	})

	t.Run("cycle_check_lookup_error", func(t *testing.T) {
		reg := &types.Registry{ID: 100, Name: "virt"}
		repo := &rmdRegRepo{all: metas, getErr: errors.New("db")}
		err := newCtrl(repo).setUpstreamProxyIDs(context.Background(), reg, virtualDTO("up1"), 2)
		require.EqualError(t, err, "db")
	})
}

func TestRmdGetUpstreamProxyKeys(t *testing.T) {
	c := &APIController{RegistryRepository: &rmdRegRepo{upstreamKeys: []string{"k1"}}}
	require.Equal(t, []string{"k1"}, c.getUpstreamProxyKeys(context.Background(), []int64{1}))
}

func TestRmdRegistryEntityBuilders(t *testing.T) {
	virtualDTO := func(id string, pt artifact.PackageType) artifact.RegistryRequest {
		rc := &artifact.RegistryConfig{}
		require.NoError(t, rc.FromVirtualConfig(artifact.VirtualConfig{}))
		return artifact.RegistryRequest{
			Identifier: id, PackageType: pt, Config: rc, Description: rmdStrPtr("desc"), IsPublic: true,
			Labels: &[]string{"l"},
		}
	}

	t.Run("create_registry_entity", func(t *testing.T) {
		c := &APIController{PackageWrapper: rmdPkgWrapper{}}
		got, err := c.CreateRegistryEntity(virtualDTO("my-reg", artifact.PackageTypeNPM), 2, 1)
		require.NoError(t, err)
		require.Equal(t, "my-reg", got.Name)
		require.Equal(t, int64(2), got.ParentID)
		require.Equal(t, int64(1), got.RootParentID)
		require.Equal(t, artifact.RegistryTypeVIRTUAL, got.Type)
		require.True(t, got.IsPublic)
		require.Equal(t, "desc", got.Description)
		require.Equal(t, []string{"l"}, got.Labels)
	})

	t.Run("create_registry_entity_errors", func(t *testing.T) {
		_, err := (&APIController{PackageWrapper: rmdPkgWrapper{invalidPkgType: true}}).
			CreateRegistryEntity(virtualDTO("my-reg", artifact.PackageTypeNPM), 2, 1)
		require.ErrorContains(t, err, "invalid package type")

		_, err = (&APIController{PackageWrapper: rmdPkgWrapper{invalidRepoType: true}}).
			CreateRegistryEntity(virtualDTO("my-reg", artifact.PackageTypeNPM), 2, 1)
		require.ErrorContains(t, err, "invalid repo type")

		_, err = (&APIController{PackageWrapper: rmdPkgWrapper{}}).
			CreateRegistryEntity(virtualDTO("Bad Name", artifact.PackageTypeNPM), 2, 1)
		require.EqualError(t, err, RegistryIdentifierErrorMsg)
	})

	t.Run("update_repo_entity", func(t *testing.T) {
		existing := &types.Registry{
			ID: 4, Name: "my-reg", PackageType: artifact.PackageTypeNPM, Type: artifact.RegistryTypeVIRTUAL,
			CreatedAt: time.UnixMilli(5),
		}
		got, err := UpdateRepoEntity(virtualDTO("my-reg", artifact.PackageTypeNPM), 2, 1, existing)
		require.NoError(t, err)
		require.Equal(t, int64(4), got.ID)
		require.Equal(t, existing.CreatedAt, got.CreatedAt)
		require.Equal(t, artifact.RegistryTypeVIRTUAL, got.Type)
		require.True(t, got.IsPublic)

		_, err = UpdateRepoEntity(virtualDTO("my-reg", artifact.PackageTypeDOCKER), 2, 1, existing)
		require.ErrorContains(t, err, "package type change")

		upstreamDTO := virtualDTO("my-reg", artifact.PackageTypeNPM)
		upstreamDTO.Config = rmdUpstreamConfigBody(t, artifact.UpstreamConfig{AuthType: artifact.AuthTypeAnonymous})
		_, err = UpdateRepoEntity(upstreamDTO, 2, 1, existing)
		require.ErrorContains(t, err, "registry type change")

		_, err = UpdateRepoEntity(virtualDTO("renamed", artifact.PackageTypeNPM), 2, 1, existing)
		require.ErrorContains(t, err, "identifier change")
	})
}

func TestRmdCreateUpstreamProxyEntity(t *testing.T) {
	dockerhub := artifact.UpstreamConfigSourceDockerhub
	newDTO := func(t *testing.T, cfg artifact.UpstreamConfig) artifact.RegistryRequest {
		t.Helper()
		return artifact.RegistryRequest{
			Identifier: "up-reg", PackageType: artifact.PackageTypeDOCKER, Config: rmdUpstreamConfigBody(t, cfg),
			Labels: &[]string{"l"}, IsPublic: true,
		}
	}
	ctx := context.Background()

	t.Run("anonymous", func(t *testing.T) {
		c := &APIController{PackageWrapper: rmdPkgWrapper{}, RegistryMetadataHelper: rmdHelper{}}
		suffix := "/simple"
		dto := newDTO(t, artifact.UpstreamConfig{
			AuthType: artifact.AuthTypeAnonymous, Source: &dockerhub, Url: rmdStrPtr("https://hub.example.com/"),
			RemoteUrlSuffix: &suffix,
		})
		reg, up, err := c.CreateUpstreamProxyEntity(ctx, dto, 2, 1)
		require.NoError(t, err)
		require.Equal(t, artifact.RegistryTypeUPSTREAM, reg.Type)
		require.Equal(t, "/simple", reg.Config.RemoteUrlSuffix)
		require.True(t, reg.IsPublic)
		require.Equal(t, "https://hub.example.com", up.URL)
		require.Equal(t, "Dockerhub", up.Source)
		require.Equal(t, "Anonymous", up.AuthType)
		require.Empty(t, up.UserName)
	})

	t.Run("user_password_with_space_path", func(t *testing.T) {
		c := &APIController{PackageWrapper: rmdPkgWrapper{}, RegistryMetadataHelper: rmdHelper{secretSpace: 77}}
		dto := newDTO(t, artifact.UpstreamConfig{
			AuthType: artifact.AuthTypeUserPassword,
			Auth: rmdUserPasswordAuth(t, artifact.UserPassword{
				UserName: "bob", SecretIdentifier: rmdStrPtr("sec"), SecretSpacePath: rmdStrPtr("root/s"),
			}),
		})
		_, up, err := c.CreateUpstreamProxyEntity(ctx, dto, 2, 1)
		require.NoError(t, err)
		require.Equal(t, "bob", up.UserName)
		require.Equal(t, "sec", up.SecretIdentifier)
		require.Equal(t, int64(77), up.SecretSpaceID)
	})

	t.Run("user_password_with_space_id", func(t *testing.T) {
		c := &APIController{PackageWrapper: rmdPkgWrapper{}, RegistryMetadataHelper: rmdHelper{}}
		dto := newDTO(t, artifact.UpstreamConfig{
			AuthType: artifact.AuthTypeUserPassword,
			Auth: rmdUserPasswordAuth(t, artifact.UserPassword{
				UserName: "bob", SecretIdentifier: rmdStrPtr("sec"), SecretSpaceId: rmdInt64Ptr(5),
			}),
		})
		_, up, err := c.CreateUpstreamProxyEntity(ctx, dto, 2, 1)
		require.NoError(t, err)
		require.Equal(t, int64(5), up.SecretSpaceID)
	})

	t.Run("user_password_errors", func(t *testing.T) {
		c := &APIController{PackageWrapper: rmdPkgWrapper{}, RegistryMetadataHelper: rmdHelper{}}
		noSecret := newDTO(t, artifact.UpstreamConfig{
			AuthType: artifact.AuthTypeUserPassword,
			Auth:     rmdUserPasswordAuth(t, artifact.UserPassword{UserName: "bob"}),
		})
		_, _, err := c.CreateUpstreamProxyEntity(ctx, noSecret, 2, 1)
		require.ErrorContains(t, err, "secret_identifier missing")

		bad := &APIController{
			PackageWrapper:         rmdPkgWrapper{},
			RegistryMetadataHelper: rmdHelper{secretSpaceE: errors.New("space")},
		}
		withPath := newDTO(t, artifact.UpstreamConfig{
			AuthType: artifact.AuthTypeUserPassword,
			Auth: rmdUserPasswordAuth(t, artifact.UserPassword{
				UserName: "bob", SecretIdentifier: rmdStrPtr("s"), SecretSpacePath: rmdStrPtr("p"),
			}),
		})
		_, _, err = bad.CreateUpstreamProxyEntity(ctx, withPath, 2, 1)
		require.EqualError(t, err, "space")
	})

	t.Run("access_key_variants", func(t *testing.T) {
		c := &APIController{PackageWrapper: rmdPkgWrapper{}, RegistryMetadataHelper: rmdHelper{secretSpace: 9}}

		plain := newDTO(t, artifact.UpstreamConfig{
			AuthType: artifact.AuthTypeAccessKeySecretKey,
			Auth: rmdAccessKeyAuth(t, artifact.AccessKeySecretKey{
				AccessKey: rmdStrPtr("AKIA"), SecretKeyIdentifier: "sk", SecretKeySpaceId: rmdInt64Ptr(3),
			}),
		})
		_, up, err := c.CreateUpstreamProxyEntity(ctx, plain, 2, 1)
		require.NoError(t, err)
		require.Equal(t, "AKIA", up.UserName)
		require.Equal(t, "sk", up.SecretIdentifier)
		require.Equal(t, int64(3), up.SecretSpaceID)

		viaSecret := newDTO(t, artifact.UpstreamConfig{
			AuthType: artifact.AuthTypeAccessKeySecretKey,
			Auth: rmdAccessKeyAuth(t, artifact.AccessKeySecretKey{
				AccessKeySecretIdentifier: rmdStrPtr("aks"), AccessKeySecretSpacePath: rmdStrPtr("root/a"),
				SecretKeyIdentifier: "sk", SecretKeySpacePath: rmdStrPtr("root/b"),
			}),
		})
		_, up, err = c.CreateUpstreamProxyEntity(ctx, viaSecret, 2, 1)
		require.NoError(t, err)
		require.Equal(t, "aks", up.UserNameSecretIdentifier)
		require.Equal(t, int64(9), up.UserNameSecretSpaceID)
		require.Equal(t, int64(9), up.SecretSpaceID)

		viaID := newDTO(t, artifact.UpstreamConfig{
			AuthType: artifact.AuthTypeAccessKeySecretKey,
			Auth: rmdAccessKeyAuth(t, artifact.AccessKeySecretKey{
				AccessKeySecretIdentifier: rmdStrPtr("aks"), AccessKeySecretSpaceId: rmdInt64Ptr(4),
				SecretKeyIdentifier: "sk",
			}),
		})
		_, up, err = c.CreateUpstreamProxyEntity(ctx, viaID, 2, 1)
		require.NoError(t, err)
		require.Equal(t, int64(4), up.UserNameSecretSpaceID)

		missing := newDTO(t, artifact.UpstreamConfig{
			AuthType: artifact.AuthTypeAccessKeySecretKey,
			Auth:     rmdAccessKeyAuth(t, artifact.AccessKeySecretKey{SecretKeyIdentifier: "sk"}),
		})
		_, _, err = c.CreateUpstreamProxyEntity(ctx, missing, 2, 1)
		require.ErrorContains(t, err, "access_key_secret_identifier missing")

		bad := &APIController{
			PackageWrapper:         rmdPkgWrapper{},
			RegistryMetadataHelper: rmdHelper{secretSpaceE: errors.New("space")},
		}
		_, _, err = bad.CreateUpstreamProxyEntity(ctx, viaSecret, 2, 1)
		require.EqualError(t, err, "space")
		secretKeyPathOnly := newDTO(t, artifact.UpstreamConfig{
			AuthType: artifact.AuthTypeAccessKeySecretKey,
			Auth: rmdAccessKeyAuth(t, artifact.AccessKeySecretKey{
				AccessKey: rmdStrPtr("AKIA"), SecretKeyIdentifier: "sk", SecretKeySpacePath: rmdStrPtr("root/b"),
			}),
		})
		_, _, err = bad.CreateUpstreamProxyEntity(ctx, secretKeyPathOnly, 2, 1)
		require.EqualError(t, err, "space")
	})

	t.Run("validation_failures", func(t *testing.T) {
		anon := artifact.UpstreamConfig{AuthType: artifact.AuthTypeAnonymous, Source: &dockerhub}

		_, _, err := (&APIController{PackageWrapper: rmdPkgWrapper{invalidPkgType: true}}).
			CreateUpstreamProxyEntity(ctx, newDTO(t, anon), 2, 1)
		require.ErrorContains(t, err, "invalid package type")

		_, _, err = (&APIController{PackageWrapper: rmdPkgWrapper{urlRequired: true}}).
			CreateUpstreamProxyEntity(ctx, newDTO(t, anon), 2, 1)
		require.ErrorContains(t, err, "URL is required")

		_, _, err = (&APIController{PackageWrapper: rmdPkgWrapper{invalidSource: true}}).
			CreateUpstreamProxyEntity(ctx, newDTO(t, anon), 2, 1)
		require.ErrorContains(t, err, "invalid upstream source")

		badID := newDTO(t, anon)
		badID.Identifier = "Not Valid"
		_, _, err = (&APIController{PackageWrapper: rmdPkgWrapper{}}).CreateUpstreamProxyEntity(ctx, badID, 2, 1)
		require.EqualError(t, err, RegistryIdentifierErrorMsg)

		_, _, err = (&APIController{PackageWrapper: rmdPkgWrapper{}}).CreateUpstreamProxyEntity(ctx,
			newDTO(t, artifact.UpstreamConfig{AuthType: artifact.AuthType("Kerberos")}), 2, 1)
		require.Error(t, err)
	})
}

func TestRmdUpdateUpstreamProxyEntity(t *testing.T) {
	dockerhub := artifact.UpstreamConfigSourceDockerhub
	created := time.UnixMilli(10)
	existing := func() *types.UpstreamProxy {
		return &types.UpstreamProxy{
			ID: 8, RegistryID: 4, RepoKey: "up-reg", PackageType: artifact.PackageTypeDOCKER, CreatedAt: created,
		}
	}
	newDTO := func(t *testing.T, cfg artifact.UpstreamConfig) artifact.RegistryRequest {
		t.Helper()
		return artifact.RegistryRequest{
			Identifier: "up-reg", PackageType: artifact.PackageTypeDOCKER, Config: rmdUpstreamConfigBody(t, cfg),
			Description: rmdStrPtr("d"), AllowedPattern: &[]string{"a"}, BlockedPattern: &[]string{"b"},
		}
	}
	ctx := context.Background()
	c := &APIController{PackageWrapper: rmdPkgWrapper{}, RegistryMetadataHelper: rmdHelper{secretSpace: 6}}

	t.Run("anonymous_clears_url_for_sources_without_url", func(t *testing.T) {
		dto := newDTO(t, artifact.UpstreamConfig{
			AuthType: artifact.AuthTypeAnonymous, Source: &dockerhub, Url: rmdStrPtr("https://x/"),
		})
		reg, up, err := c.UpdateUpstreamProxyEntity(ctx, dto, 2, 1, existing())
		require.NoError(t, err)
		require.Equal(t, int64(4), reg.ID)
		require.Equal(t, created, reg.CreatedAt)
		require.Equal(t, int64(8), up.ID)
		require.Equal(t, int64(4), up.RegistryID)
		require.Empty(t, up.URL)
		require.Equal(t, "Dockerhub", up.Source)
		require.Equal(t, []string{"a"}, reg.AllowedPattern)
	})

	t.Run("url_kept_when_required", func(t *testing.T) {
		req := &APIController{PackageWrapper: rmdPkgWrapper{urlRequired: true}, RegistryMetadataHelper: rmdHelper{}}
		dto := newDTO(t, artifact.UpstreamConfig{
			AuthType: artifact.AuthTypeAnonymous, Source: &dockerhub, Url: rmdStrPtr("https://x/"),
			RemoteUrlSuffix: rmdStrPtr("/s"),
			UpstreamProxyConfigFirewallMode: func() *artifact.UpstreamProxyConfigFirewallMode {
				m := artifact.UpstreamProxyConfigFirewallMode("WARN")
				return &m
			}(),
		})
		reg, up, err := req.UpdateUpstreamProxyEntity(ctx, dto, 2, 1, existing())
		require.NoError(t, err)
		require.Equal(t, "https://x", up.URL)
		require.Equal(t, "WARN", up.FirewallMode)
		require.Equal(t, "/s", reg.Config.RemoteUrlSuffix)
	})

	t.Run("unknown_id_is_not_copied", func(t *testing.T) {
		u := existing()
		u.ID = -1
		dto := newDTO(t, artifact.UpstreamConfig{AuthType: artifact.AuthTypeAnonymous})
		_, up, err := c.UpdateUpstreamProxyEntity(ctx, dto, 2, 1, u)
		require.NoError(t, err)
		require.Equal(t, int64(0), up.ID)
	})

	t.Run("user_password", func(t *testing.T) {
		dto := newDTO(t, artifact.UpstreamConfig{
			AuthType: artifact.AuthTypeUserPassword,
			Auth: rmdUserPasswordAuth(t, artifact.UserPassword{
				UserName: "bob", SecretIdentifier: rmdStrPtr("sec"), SecretSpacePath: rmdStrPtr("root/s"),
			}),
		})
		_, up, err := c.UpdateUpstreamProxyEntity(ctx, dto, 2, 1, existing())
		require.NoError(t, err)
		require.Equal(t, "bob", up.UserName)
		require.Equal(t, int64(6), up.SecretSpaceID)

		withID := newDTO(t, artifact.UpstreamConfig{
			AuthType: artifact.AuthTypeUserPassword,
			Auth: rmdUserPasswordAuth(t, artifact.UserPassword{
				UserName: "bob", SecretIdentifier: rmdStrPtr("sec"), SecretSpaceId: rmdInt64Ptr(2),
			}),
		})
		_, up, err = c.UpdateUpstreamProxyEntity(ctx, withID, 2, 1, existing())
		require.NoError(t, err)
		require.Equal(t, int64(2), up.SecretSpaceID)

		noSecret := newDTO(t, artifact.UpstreamConfig{
			AuthType: artifact.AuthTypeUserPassword,
			Auth:     rmdUserPasswordAuth(t, artifact.UserPassword{UserName: "bob"}),
		})
		_, _, err = c.UpdateUpstreamProxyEntity(ctx, noSecret, 2, 1, existing())
		require.ErrorContains(t, err, "secret_identifier missing")

		bad := &APIController{
			PackageWrapper:         rmdPkgWrapper{},
			RegistryMetadataHelper: rmdHelper{secretSpaceE: errors.New("space")},
		}
		_, _, err = bad.UpdateUpstreamProxyEntity(ctx, dto, 2, 1, existing())
		require.EqualError(t, err, "space")
	})

	t.Run("access_key", func(t *testing.T) {
		dto := newDTO(t, artifact.UpstreamConfig{
			AuthType: artifact.AuthTypeAccessKeySecretKey,
			Auth: rmdAccessKeyAuth(t, artifact.AccessKeySecretKey{
				AccessKeySecretIdentifier: rmdStrPtr("aks"), AccessKeySecretSpacePath: rmdStrPtr("root/a"),
				SecretKeyIdentifier: "sk", SecretKeySpacePath: rmdStrPtr("root/b"),
			}),
		})
		_, up, err := c.UpdateUpstreamProxyEntity(ctx, dto, 2, 1, existing())
		require.NoError(t, err)
		require.Equal(t, "aks", up.UserNameSecretIdentifier)
		require.Equal(t, int64(6), up.UserNameSecretSpaceID)
		require.Equal(t, int64(6), up.SecretSpaceID)

		plain := newDTO(t, artifact.UpstreamConfig{
			AuthType: artifact.AuthTypeAccessKeySecretKey,
			Auth: rmdAccessKeyAuth(t, artifact.AccessKeySecretKey{
				AccessKey: rmdStrPtr("AKIA"), SecretKeyIdentifier: "sk", SecretKeySpaceId: rmdInt64Ptr(3),
			}),
		})
		_, up, err = c.UpdateUpstreamProxyEntity(ctx, plain, 2, 1, existing())
		require.NoError(t, err)
		require.Equal(t, "AKIA", up.UserName)
		require.Equal(t, int64(3), up.SecretSpaceID)

		missing := newDTO(t, artifact.UpstreamConfig{
			AuthType: artifact.AuthTypeAccessKeySecretKey,
			Auth:     rmdAccessKeyAuth(t, artifact.AccessKeySecretKey{SecretKeyIdentifier: "sk"}),
		})
		_, _, err = c.UpdateUpstreamProxyEntity(ctx, missing, 2, 1, existing())
		require.ErrorContains(t, err, "access_key_secret_identifier missing")
	})

	t.Run("validation_failures", func(t *testing.T) {
		anon := artifact.UpstreamConfig{AuthType: artifact.AuthTypeAnonymous, Source: &dockerhub}

		changedPkg := newDTO(t, anon)
		changedPkg.PackageType = artifact.PackageTypeNPM
		_, _, err := c.UpdateUpstreamProxyEntity(ctx, changedPkg, 2, 1, existing())
		require.ErrorContains(t, err, "package type change")

		renamed := newDTO(t, anon)
		renamed.Identifier = "other"
		_, _, err = c.UpdateUpstreamProxyEntity(ctx, renamed, 2, 1, existing())
		require.ErrorContains(t, err, "identifier change")

		_, _, err = (&APIController{PackageWrapper: rmdPkgWrapper{invalidSource: true}}).
			UpdateUpstreamProxyEntity(ctx, newDTO(t, anon), 2, 1, existing())
		require.ErrorContains(t, err, "invalid upstream source")

		_, _, err = (&APIController{PackageWrapper: rmdPkgWrapper{urlRequired: true}}).
			UpdateUpstreamProxyEntity(ctx, newDTO(t, anon), 2, 1, existing())
		require.ErrorContains(t, err, "URL is required")
	})
}

func rmdSpaceCore() *coretypes.SpaceCore {
	return &coretypes.SpaceCore{ID: 2, Path: "root", Identifier: "root"}
}
