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

package packages

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/auth/authn"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/registry/app/api/handler/generic"
	"github.com/harness/gitness/registry/app/api/handler/maven"
	pkghandler "github.com/harness/gitness/registry/app/api/handler/packages"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

var errRb5Info = errors.New("artifact info unavailable")

func rb5Say(w http.ResponseWriter, name, method string) {
	_, _ = w.Write([]byte(name + "." + method))
}

// rb5H is a package handler fake. It implements the union of all package handler interfaces.
type rb5H struct {
	name string
}

func (h rb5H) GetPackageArtifactInfo(*http.Request) (pkg.PackageArtifactInfo, error) {
	return pkg.GenericArtifactInfo{ArtifactInfo: &pkg.ArtifactInfo{}, FileName: "f"}, nil
}

type rb5Auth struct {
	mu     sync.Mutex
	header string
}

func (a *rb5Auth) Authenticate(r *http.Request) (*auth.Session, error) {
	a.mu.Lock()
	a.header = r.Header.Get("Authorization")
	a.mu.Unlock()
	if r.Header.Get("X-Rb5-User") != "" {
		return &auth.Session{Principal: types.Principal{ID: 5, UID: "rb5user"}}, nil
	}
	return nil, authn.ErrNoAuthData
}

func (a *rb5Auth) authorization() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.header
}

// rb5Pkg is the shared packages.Handler fake.
type rb5Pkg struct {
	pkghandler.Handler
	authenticator *rb5Auth
	accessErr     error
	quarantineErr error
	mu            sync.Mutex
	perms         []enum.Permission
	stats         int
}

func (p *rb5Pkg) GetAuthenticator() authn.Authenticator { return p.authenticator }

func (p *rb5Pkg) GetRegistryCheckAccess(_ context.Context, _ *http.Request, perms ...enum.Permission) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.perms = append(p.perms, perms...)
	return p.accessErr
}

func (p *rb5Pkg) GetArtifactInfo(*http.Request) (pkg.ArtifactInfo, error) {
	return pkg.ArtifactInfo{}, errRb5Info
}

func (p *rb5Pkg) CheckQuarantineStatus(context.Context) error { return p.quarantineErr }

func (p *rb5Pkg) TrackDownloadStats(context.Context, *http.Request) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stats++
	return nil
}

func (p *rb5Pkg) DownloadFile(w http.ResponseWriter, _ *http.Request) {
	rb5Say(w, "pkg", "DownloadFile")
}

func (p *rb5Pkg) lastPerm() enum.Permission {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.perms) == 0 {
		return ""
	}
	return p.perms[len(p.perms)-1]
}

func rb5Router(p *rb5Pkg) Handler {
	return NewRouter(
		p, &maven.Handler{}, &generic.Handler{Handler: p}, rb5H{"py"}, rb5H{"nuget"}, rb5H{"npm"}, rb5H{"rpm"},
		rb5H{"cargo"}, rb5H{"go"}, rb5H{"hf"}, refcache.SpaceFinder{}, nil,
	)
}

func rb5NewPkg() *rb5Pkg { return &rb5Pkg{authenticator: &rb5Auth{}} }

func rb5Do(h http.Handler, method, path string, headers ...string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "/root/reg"+path, nil)
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

type rb5Route struct {
	method string
	path   string
	want   string
	perm   enum.Permission
}

func rb5Routes() []rb5Route {
	dl, up, del, view := enum.PermissionArtifactsDownload, enum.PermissionArtifactsUpload,
		enum.PermissionArtifactsDelete, enum.PermissionRegistryView
	return []rb5Route{
		{http.MethodGet, "/download/", "pkg.DownloadFile", dl},

		{http.MethodPost, "/python/upload", "py.UploadPackageFile", up},
		{http.MethodGet, "/python/files/img/1.0/f.whl", "py.DownloadPackageFile", dl},
		{http.MethodGet, "/python/_/other/img/1.0/f.whl", "py.DownloadPackageFile", dl},
		{http.MethodGet, "/python/simple/img/", "py.PackageMetadata", dl},

		{http.MethodPut, "/nuget/pkg.nupkg", "nuget.UploadPackage", up},
		{http.MethodPut, "/nuget/symbolpackage/pkg.snupkg", "nuget.UploadSymbolPackage", up},
		{http.MethodGet, "/nuget/package/id/1.0/f.nupkg", "nuget.DownloadPackage", dl},
		{http.MethodDelete, "/nuget/id/1.0", "nuget.DeletePackage", del},
		{http.MethodGet, "/nuget/index.json", "nuget.GetServiceEndpoint", dl},
		{http.MethodGet, "/nuget/", "nuget.GetServiceEndpointV2", dl},
		{http.MethodGet, "/nuget/$metadata", "nuget.GetServiceMetadataV2", dl},
		{http.MethodGet, "/nuget/package/id/index.json", "nuget.ListPackageVersion", dl},
		{http.MethodGet, "/nuget/FindPackagesById()", "nuget.ListPackageVersionV2", dl},
		{http.MethodGet, "/nuget/query", "nuget.SearchPackage", dl},
		{http.MethodGet, "/nuget/Packages()", "nuget.SearchPackageV2", dl},
		{http.MethodGet, "/nuget/Packages()/$count", "nuget.CountPackageV2", dl},
		{http.MethodGet, "/nuget/Search()", "nuget.SearchPackageV2", dl},
		{http.MethodGet, "/nuget/Search()/$count", "nuget.CountPackageV2", dl},
		{http.MethodGet, "/nuget/FindPackagesById()/$count", "nuget.GetPackageVersionCountV2", dl},
		{http.MethodGet, "/nuget/registration/id/index.json", "nuget.GetPackageMetadata", dl},
		{http.MethodGet, "/nuget/Packages(Id='a',Version='1')", "nuget.GetPackageVersionMetadataV2", dl},
		{http.MethodGet, "/nuget/id/1.0/readme", "nuget.GetReadme", dl},
		{http.MethodGet, "/nuget/registration/id/1.0", "nuget.GetPackageVersionMetadata", dl},

		{http.MethodPut, "/npm/@sc/id", "npm.UploadPackage", up},
		{http.MethodGet, "/npm/@sc/id/-/1.0/@sc/f.tgz", "npm.DownloadPackageFile", dl},
		{http.MethodGet, "/npm/@sc/id/-/@sc/f.tgz", "npm.DownloadPackageFileByName", dl},
		{http.MethodHead, "/npm/@sc/id/-/@sc/f.tgz", "npm.HeadPackageFileByName", dl},
		{http.MethodGet, "/npm/@sc/id", "npm.GetPackageMetadata", dl},
		{http.MethodDelete, "/npm/@sc/id/-/1.0/@sc/f.tgz/-rev/1", "npm.DeleteVersion", del},
		{http.MethodPut, "/npm/id", "npm.UploadPackage", up},
		{http.MethodGet, "/npm/id/-/1.0/f.tgz", "npm.DownloadPackageFile", dl},
		{http.MethodGet, "/npm/id/-/f.tgz", "npm.DownloadPackageFileByName", dl},
		{http.MethodHead, "/npm/id/-/f.tgz", "npm.HeadPackageFileByName", dl},
		{http.MethodGet, "/npm/id", "npm.GetPackageMetadata", dl},
		{http.MethodDelete, "/npm/id/-/1.0/f.tgz/-rev/1", "npm.DeleteVersion", del},
		{http.MethodGet, "/npm/-/package/id/dist-tags", "npm.ListPackageTag", up},
		{http.MethodPut, "/npm/-/package/id/dist-tags/latest", "npm.AddPackageTag", up},
		{http.MethodDelete, "/npm/-/package/id/dist-tags/latest", "npm.DeletePackageTag", up},
		{http.MethodGet, "/npm/-/package/@sc/id/dist-tags", "npm.ListPackageTag", up},
		{http.MethodDelete, "/npm/id/-rev/1", "npm.DeletePackage", del},
		{http.MethodPut, "/npm/id/-rev/1", "npm.DeletePreview", del},
		{http.MethodDelete, "/npm/@sc/-rev/1", "npm.DeletePackage", del},
		{http.MethodGet, "/npm/-/v1/search", "npm.SearchPackage", dl},

		{http.MethodPut, "/rpm/x.rpm", "rpm.UploadPackageFile", up},
		{http.MethodGet, "/rpm/repodata/repomd.xml", "rpm.GetRepoData", dl},
		{http.MethodGet, "/rpm/package/n/1/x86_64/f.rpm", "rpm.DownloadPackageFile", dl},
		{http.MethodGet, "/rpm/package/n/1/x86_64/f.rpm/extra", "rpm.DownloadPackageFile", dl},

		{http.MethodGet, "/cargo/index/config.json", "cargo.GetRegistryConfig", view},
		{http.MethodPut, "/cargo/index/n/regenerate", "cargo.RegeneratePackageIndex", view},
		{http.MethodGet, "/cargo/index/ab/cd/name", "cargo.DownloadPackageIndex", view},
		{http.MethodGet, "/cargo/api/v1/crates", "cargo.SearchPackage", view},
		{http.MethodPut, "/cargo/api/v1/crates/new", "cargo.UploadPackage", up},
		{http.MethodGet, "/cargo/api/v1/crates/n/1/download", "cargo.DownloadPackage", dl},
		{http.MethodDelete, "/cargo/api/v1/crates/n/1/yank", "cargo.YankVersion", up},
		{http.MethodPut, "/cargo/api/v1/crates/n/1/unyank", "cargo.UnYankVersion", up},

		{http.MethodPut, "/go/upload", "go.UploadPackage", up},
		{http.MethodPut, "/go/regenerate-index", "go.RegeneratePackageIndex", up},
		{http.MethodPut, "/go/regenerate-metadata", "go.RegeneratePackageMetadata", up},
		{http.MethodGet, "/go/a/b/@v/list", "go.DownloadPackageFile", dl},

		{http.MethodPost, "/huggingface/api/validate-yaml", "hf.ValidateYAML", dl},
		{http.MethodPost, "/huggingface/api/models/r/preupload/main", "hf.PreUpload", dl},
		{http.MethodGet, "/huggingface/api/models/r/revision/main", "hf.RevisionInfo", dl},
		{http.MethodPost, "/huggingface/r.git/info/lfs/objects/batch", "hf.LfsInfo", dl},
		{http.MethodPost, "/huggingface/models/r.git/info/lfs/objects/batch", "hf.LfsInfo", dl},
		{http.MethodPut, "/huggingface/api/models/r/main/multipart/upload/abc", "hf.LfsUpload", up},
		{http.MethodPost, "/huggingface/api/models/r/main/multipart/verify/abc", "hf.LfsVerify", dl},
		{http.MethodPost, "/huggingface/api/models/r/commit/main", "hf.CommitRevision", up},
		{http.MethodHead, "/huggingface/r/resolve/main/f.bin", "hf.HeadFile", dl},
		{http.MethodHead, "/huggingface/models/r/resolve/main/f.bin", "hf.HeadFile", dl},
		{http.MethodGet, "/huggingface/r/resolve/main/f.bin", "hf.DownloadFile", dl},
		{http.MethodGet, "/huggingface/models/r/resolve/main/f.bin", "hf.DownloadFile", dl},
	}
}

func TestRb5RouterDispatch(t *testing.T) {
	for _, rt := range rb5Routes() {
		t.Run(rt.method+" "+rt.path, func(t *testing.T) {
			p := rb5NewPkg()
			rec := rb5Do(rb5Router(p), rt.method, rt.path)
			require.Equal(t, http.StatusOK, rec.Code)
			if rt.method != http.MethodHead {
				require.Equal(t, rt.want, rec.Body.String())
			}
			require.Equal(t, rt.perm, p.lastPerm())
		})
	}
}

func TestRb5RouterAccessDenied(t *testing.T) {
	routes := []rb5Route{
		{method: http.MethodPost, path: "/python/upload"},
		{method: http.MethodGet, path: "/npm/id"},
		{method: http.MethodPut, path: "/rpm/x.rpm"},
		{method: http.MethodGet, path: "/cargo/index/config.json"},
		{method: http.MethodPut, path: "/go/upload"},
		{method: http.MethodGet, path: "/huggingface/r/resolve/main/f.bin"},
		{method: http.MethodGet, path: "/download/"},
	}
	for _, rt := range routes {
		t.Run(rt.method+" "+rt.path, func(t *testing.T) {
			p := rb5NewPkg()
			p.accessErr = apiauth.ErrForbidden
			rec := rb5Do(rb5Router(p), rt.method, rt.path)
			require.Equal(t, http.StatusForbidden, rec.Code)
			require.NotContains(t, rec.Body.String(), ".")
		})
	}
}

func TestRb5RouterNugetChallenge(t *testing.T) {
	tests := []struct {
		method    string
		path      string
		challenge string
	}{
		{http.MethodPut, "/nuget/pkg.nupkg", `ApiKey realm="Harness Registry"`},
		{http.MethodDelete, "/nuget/id/1.0", `ApiKey realm="Harness Registry"`},
		{http.MethodGet, "/nuget/index.json", `Basic realm="Harness Registry"`},
	}
	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			p := rb5NewPkg()
			p.accessErr = apiauth.ErrUnauthorized
			rec := rb5Do(rb5Router(p), tt.method, tt.path)
			require.Equal(t, http.StatusUnauthorized, rec.Code)
			require.Equal(t, tt.challenge, rec.Header().Get("WWW-Authenticate"))
		})
	}
	t.Run("other errors are translated", func(t *testing.T) {
		p := rb5NewPkg()
		p.accessErr = apiauth.ErrForbidden
		rec := rb5Do(rb5Router(p), http.MethodGet, "/nuget/index.json")
		require.Equal(t, http.StatusForbidden, rec.Code)
	})
}

func TestRb5RouterCredentialHeaders(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		headers []string
		want    string
	}{
		{"nuget api key", "/nuget/index.json", []string{"x-nuget-apikey", "nugetkey"}, "nugetkey"},
		{"go api key", "/go/a/b/@v/list", []string{"x-api-key", "gokey"}, "gokey"},
		{"maven api key", "/maven/x", []string{"x-api-key", "mvnkey"}, "mvnkey"},
		{"hf sig bearer", "/huggingface/r/resolve/main/f.bin?sig=Bearer%20hftok", nil, "Bearer hftok"},
		{"hf sig basic", "/huggingface/r/resolve/main/f.bin?sig=Basic%20hftok", nil, "Bearer hftok"},
		{"hf sig raw", "/huggingface/r/resolve/main/f.bin?sig=hftok", nil, "Bearer hftok"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := rb5NewPkg()
			rb5Do(rb5Router(p), http.MethodGet, tt.path, tt.headers...)
			require.Equal(t, tt.want, p.authenticator.authorization())
		})
	}
}

func TestRb5RouterUnsupportedAndRedirect(t *testing.T) {
	h := rb5Router(rb5NewPkg())

	rec := rb5Do(h, http.MethodGet, "/composer/anything")
	require.Equal(t, http.StatusNotFound, rec.Code)
	require.Contains(t, rec.Body.String(), "Package type 'composer' is not supported")

	rec = rb5Do(h, http.MethodGet, "/python/simple/img")
	require.Equal(t, http.StatusMovedPermanently, rec.Code)
	require.Equal(t, "/root/reg/python/simple/img/", rec.Header().Get("Location"))

	rec = rb5Do(h, http.MethodGet, "/nope")
	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestRb5RouterGenericAndFiles(t *testing.T) {
	// Generic routes use the real generic handler; the artifact info lookup fails first, which proves wiring.
	p := rb5NewPkg()
	h := rb5Router(p)
	for _, rt := range []struct{ method, path string }{
		{http.MethodGet, "/generic/pkg/1.0/"},
		{http.MethodPut, "/generic/pkg/1.0/"},
		{http.MethodGet, "/files/a/b"},
		{http.MethodDelete, "/files/a/b"},
		{http.MethodHead, "/files/a/b"},
		{http.MethodPut, "/files/a/b"},
	} {
		t.Run(rt.method+" "+rt.path, func(t *testing.T) {
			rec := rb5Do(h, rt.method, rt.path)
			require.Equal(t, http.StatusInternalServerError, rec.Code, rec.Body.String())
		})
	}
}

func TestRb5RouterMaven(t *testing.T) {
	h := rb5Router(rb5NewPkg())

	// Anonymous callers are challenged because the artifact info cannot be resolved.
	rec := rb5Do(h, http.MethodGet, "/maven/")
	require.Equal(t, http.StatusUnauthorized, rec.Code)
	require.Equal(t, `Basic realm="Harness Registry"`, rec.Header().Get("WWW-Authenticate"))

	// Authenticated callers pass the challenge and fail while storing the artifact info.
	for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodPut} {
		rec = rb5Do(h, method, "/maven/", "X-Rb5-User", "1")
		require.NotEqual(t, http.StatusOK, rec.Code, method)
		require.NotEqual(t, http.StatusNotFound, rec.Code, method)
		require.NotEqual(t, http.StatusMethodNotAllowed, rec.Code, method)
	}
}

func TestRb5RouterDownloadStats(t *testing.T) {
	p := rb5NewPkg()
	h := rb5Router(p)
	rec := rb5Do(h, http.MethodGet, "/npm/id/-/1.0/f.tgz")
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, 1, p.stats)

	// Failed downloads are not counted.
	p.accessErr = apiauth.ErrForbidden
	rb5Do(h, http.MethodGet, "/npm/id/-/1.0/f.tgz")
	require.Equal(t, 1, p.stats)
}

func (h rb5H) UploadPackage(w http.ResponseWriter, _ *http.Request) {
	rb5Say(w, h.name, "UploadPackage")
}
func (h rb5H) UploadSymbolPackage(w http.ResponseWriter, _ *http.Request) {
	rb5Say(w, h.name, "UploadSymbolPackage")
}
func (h rb5H) DownloadPackage(w http.ResponseWriter, _ *http.Request) {
	rb5Say(w, h.name, "DownloadPackage")
}
func (h rb5H) DeletePackage(w http.ResponseWriter, _ *http.Request) {
	rb5Say(w, h.name, "DeletePackage")
}
func (h rb5H) GetReadme(w http.ResponseWriter, _ *http.Request) { rb5Say(w, h.name, "GetReadme") }
func (h rb5H) GetServiceEndpoint(w http.ResponseWriter, _ *http.Request) {
	rb5Say(w, h.name, "GetServiceEndpoint")
}
func (h rb5H) GetServiceEndpointV2(w http.ResponseWriter, _ *http.Request) {
	rb5Say(w, h.name, "GetServiceEndpointV2")
}
func (h rb5H) ListPackageVersion(w http.ResponseWriter, _ *http.Request) {
	rb5Say(w, h.name, "ListPackageVersion")
}
func (h rb5H) ListPackageVersionV2(w http.ResponseWriter, _ *http.Request) {
	rb5Say(w, h.name, "ListPackageVersionV2")
}
func (h rb5H) GetPackageMetadata(w http.ResponseWriter, _ *http.Request) {
	rb5Say(w, h.name, "GetPackageMetadata")
}
func (h rb5H) GetPackageVersionMetadataV2(w http.ResponseWriter, _ *http.Request) {
	rb5Say(w, h.name, "GetPackageVersionMetadataV2")
}
func (h rb5H) GetPackageVersionMetadata(w http.ResponseWriter, _ *http.Request) {
	rb5Say(w, h.name, "GetPackageVersionMetadata")
}
func (h rb5H) SearchPackage(w http.ResponseWriter, _ *http.Request) {
	rb5Say(w, h.name, "SearchPackage")
}
func (h rb5H) SearchPackageV2(w http.ResponseWriter, _ *http.Request) {
	rb5Say(w, h.name, "SearchPackageV2")
}
func (h rb5H) CountPackageV2(w http.ResponseWriter, _ *http.Request) {
	rb5Say(w, h.name, "CountPackageV2")
}
func (h rb5H) GetPackageVersionCountV2(w http.ResponseWriter, _ *http.Request) {
	rb5Say(w, h.name, "GetPackageVersionCountV2")
}
func (h rb5H) GetServiceMetadataV2(w http.ResponseWriter, _ *http.Request) {
	rb5Say(w, h.name, "GetServiceMetadataV2")
}
func (h rb5H) DownloadPackageFile(w http.ResponseWriter, _ *http.Request) {
	rb5Say(w, h.name, "DownloadPackageFile")
}
func (h rb5H) DownloadPackageFileByName(w http.ResponseWriter, _ *http.Request) {
	rb5Say(w, h.name, "DownloadPackageFileByName")
}
func (h rb5H) HeadPackageFileByName(w http.ResponseWriter, _ *http.Request) {
	rb5Say(w, h.name, "HeadPackageFileByName")
}
func (h rb5H) ListPackageTag(w http.ResponseWriter, _ *http.Request) {
	rb5Say(w, h.name, "ListPackageTag")
}
func (h rb5H) AddPackageTag(w http.ResponseWriter, _ *http.Request) {
	rb5Say(w, h.name, "AddPackageTag")
}
func (h rb5H) DeletePackageTag(w http.ResponseWriter, _ *http.Request) {
	rb5Say(w, h.name, "DeletePackageTag")
}
func (h rb5H) DeleteVersion(w http.ResponseWriter, _ *http.Request) {
	rb5Say(w, h.name, "DeleteVersion")
}
func (h rb5H) DeletePreview(w http.ResponseWriter, _ *http.Request) {
	rb5Say(w, h.name, "DeletePreview")
}
func (h rb5H) UploadPackageFile(w http.ResponseWriter, _ *http.Request) {
	rb5Say(w, h.name, "UploadPackageFile")
}
func (h rb5H) GetRepoData(w http.ResponseWriter, _ *http.Request) { rb5Say(w, h.name, "GetRepoData") }
func (h rb5H) GetRegistryConfig(w http.ResponseWriter, _ *http.Request) {
	rb5Say(w, h.name, "GetRegistryConfig")
}
func (h rb5H) DownloadPackageIndex(w http.ResponseWriter, _ *http.Request) {
	rb5Say(w, h.name, "DownloadPackageIndex")
}
func (h rb5H) RegeneratePackageIndex(w http.ResponseWriter, _ *http.Request) {
	rb5Say(w, h.name, "RegeneratePackageIndex")
}
func (h rb5H) YankVersion(w http.ResponseWriter, _ *http.Request) { rb5Say(w, h.name, "YankVersion") }
func (h rb5H) UnYankVersion(w http.ResponseWriter, _ *http.Request) {
	rb5Say(w, h.name, "UnYankVersion")
}
func (h rb5H) RegeneratePackageMetadata(w http.ResponseWriter, _ *http.Request) {
	rb5Say(w, h.name, "RegeneratePackageMetadata")
}
func (h rb5H) ValidateYAML(w http.ResponseWriter, _ *http.Request) { rb5Say(w, h.name, "ValidateYAML") }
func (h rb5H) LfsInfo(w http.ResponseWriter, _ *http.Request)      { rb5Say(w, h.name, "LfsInfo") }
func (h rb5H) LfsUpload(w http.ResponseWriter, _ *http.Request)    { rb5Say(w, h.name, "LfsUpload") }
func (h rb5H) LfsVerify(w http.ResponseWriter, _ *http.Request)    { rb5Say(w, h.name, "LfsVerify") }
func (h rb5H) PreUpload(w http.ResponseWriter, _ *http.Request)    { rb5Say(w, h.name, "PreUpload") }
func (h rb5H) RevisionInfo(w http.ResponseWriter, _ *http.Request) { rb5Say(w, h.name, "RevisionInfo") }
func (h rb5H) CommitRevision(w http.ResponseWriter, _ *http.Request) {
	rb5Say(w, h.name, "CommitRevision")
}
func (h rb5H) HeadFile(w http.ResponseWriter, _ *http.Request)     { rb5Say(w, h.name, "HeadFile") }
func (h rb5H) DownloadFile(w http.ResponseWriter, _ *http.Request) { rb5Say(w, h.name, "DownloadFile") }
func (h rb5H) PackageMetadata(w http.ResponseWriter, _ *http.Request) {
	rb5Say(w, h.name, "PackageMetadata")
}

func (p *rb5Pkg) GetPackageArtifactInfo(r *http.Request) (pkg.PackageArtifactInfo, error) {
	return rb5H{}.GetPackageArtifactInfo(r)
}
