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

package nuget

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/harness/gitness/app/services/refcache"
	urlprovider "github.com/harness/gitness/app/url"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/commons"
	nugettype "github.com/harness/gitness/registry/app/pkg/types/nuget"
	"github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/types"

	"github.com/stretchr/testify/require"
)

type rpfProxyStore struct {
	store.UpstreamProxyConfigRepository
	proxy *types.UpstreamProxy
	err   error
}

func (s *rpfProxyStore) GetByRegistryIdentifier(
	context.Context, int64, string, ...types.QueryOption,
) (*types.UpstreamProxy, error) {
	return s.proxy, s.err
}

type rpfURLProvider struct {
	urlprovider.Provider
}

func (rpfURLProvider) PackageURL(_ context.Context, regRef string, pkgType string, _ ...string) string {
	return "https://harness.test/pkg/" + regRef + "/" + pkgType
}

type rpfLocalHelper struct {
	exists      bool
	downloadErr error
	uploaded    chan string
}

func (h *rpfLocalHelper) FileExists(context.Context, nugettype.ArtifactInfo) bool { return h.exists }

func (h *rpfLocalHelper) DownloadFile(
	context.Context, nugettype.ArtifactInfo,
) (*commons.ResponseHeaders, *storage.FileReader, string, error) {
	if h.downloadErr != nil {
		return nil, nil, "", h.downloadErr
	}
	return &commons.ResponseHeaders{Code: http.StatusOK}, nil, "https://redirect.test/file", nil
}

func (h *rpfLocalHelper) UploadPackageFile(
	_ context.Context, info nugettype.ArtifactInfo, file io.ReadCloser,
) (*commons.ResponseHeaders, string, error) {
	data, _ := io.ReadAll(file)
	if h.uploaded != nil {
		h.uploaded <- info.Image + ":" + string(data)
	}
	return nil, "sha", nil
}

func (h *rpfLocalHelper) DeletePackage(context.Context, nugettype.ArtifactInfo) error { return nil }

const rpfFeedEntryTemplate = `<entry><id>%[1]s/v2/Packages(Id='demo',Version='1.0.0')</id>` +
	`<title type="text">demo</title><content type="application/zip" src="%[1]s/v2/package/demo/1.0.0"/></entry>`

func rpfUpstream(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/v3/index.json", func(w http.ResponseWriter, r *http.Request) {
		host := "http://" + r.Host
		_, _ = fmt.Fprintf(w, `{"version":"3.0.0","resources":[`+
			`{"@id":"%[1]s/v3/flat/","@type":"PackageBaseAddress/3.0.0"},`+
			`{"@id":"%[1]s/v3/reg/","@type":"RegistrationsBaseUrl/3.6.0"},`+
			`{"@id":"%[1]s/v3/query","@type":"SearchQueryService"}]}`, host)
	})
	mux.HandleFunc("/v3/flat/demo/index.json", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"versions":["1.0.0","2.0.0"]}`)
	})
	mux.HandleFunc("/v3/flat/demo/1.0.0/demo.1.0.0.nupkg", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "nupkg-bytes")
	})
	mux.HandleFunc("/v3/reg/demo/index.json", func(w http.ResponseWriter, r *http.Request) {
		host := "http://" + r.Host
		_, _ = fmt.Fprintf(w, `{"@id":"%[1]s/v3/reg/demo/index.json","count":1,"items":[`+
			`{"@id":"%[1]s/v3/reg/demo/page.json","count":1,"lower":"1.0.0","upper":"1.0.0","items":[`+
			`{"@id":"%[1]s/v3/reg/demo/1.0.0.json","packageContent":"x","catalogEntry":`+
			`{"id":"demo","version":"1.0.0","packageContent":"x"}}]}]}`, host)
	})
	mux.HandleFunc("/v3/reg/demo/page.json", func(w http.ResponseWriter, r *http.Request) {
		host := "http://" + r.Host
		_, _ = fmt.Fprintf(w, `{"@id":"%[1]s/v3/reg/demo/page.json","count":1,"lower":"1.0.0","upper":"1.0.0","items":[`+
			`{"@id":"%[1]s/v3/reg/demo/1.0.0.json","packageContent":"x","catalogEntry":`+
			`{"id":"demo","version":"1.0.0","packageContent":"x"}}]}`, host)
	})
	mux.HandleFunc("/v3/query", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"totalHits":1,"data":[{"id":"demo","registration":"https://up/reg",`+
			`"versions":[{"@id":"https://up/leaf","version":"1.0.0"}]}]}`)
	})
	mux.HandleFunc("/v2/FindPackagesById()/$count", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "2")
	})
	mux.HandleFunc("/v2/Search()/$count", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, " 5 ")
	})
	mux.HandleFunc("/v2/FindPackagesById()", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintf(w, `<feed xmlns="http://www.w3.org/2005/Atom"><title>t</title>`+rpfFeedEntryTemplate+
			`</feed>`, "http://"+r.Host)
	})
	mux.HandleFunc("/v2/Search()", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintf(w, `<feed xmlns="http://www.w3.org/2005/Atom"><title>t</title>`+rpfFeedEntryTemplate+
			`</feed>`, "http://"+r.Host)
	})
	mux.HandleFunc("/v2/Packages(Id='demo',Version='1.0.0')", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintf(w, `<entry xmlns="http://www.w3.org/2005/Atom"><id>%[1]s/v2/Packages(Id='demo',`+
			`Version='1.0.0')</id><content type="application/zip" src="%[1]s/v2/package/demo/1.0.0"/></entry>`,
			"http://"+r.Host)
	})
	mux.HandleFunc("/v2/package/demo/1.0.0", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "v2-nupkg")
	})
	mux.HandleFunc("/broken/index.json", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "not json")
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func rpfNewProxy(repoURL string, local *rpfLocalHelper, storeErr error) *proxy {
	up := &types.UpstreamProxy{
		RepoKey: "up", RepoURL: repoURL, RepoAuthType: string(artifact.AuthTypeAnonymous),
		PackageType: artifact.PackageTypeNUGET,
	}
	p := NewProxy(nil, &rpfProxyStore{proxy: up, err: storeErr}, nil, nil, nil, nil,
		rpfURLProvider{}, rpfSpaceFinder(), nil, local)
	px, ok := p.(*proxy)
	if !ok {
		return nil
	}
	return px
}

func rpfProxyInfo() nugettype.ArtifactInfo {
	return nugettype.ArtifactInfo{
		ArtifactInfo: pkg.ArtifactInfo{
			BaseInfo:      &pkg.BaseInfo{ParentID: 1, RootIdentifier: "root"},
			RegIdentifier: "reg",
			Image:         "demo",
		},
		Version:  "1.0.0",
		Filename: "demo.1.0.0.nupkg",
	}
}

func TestRpfProxyStatic(t *testing.T) {
	ctx := context.Background()
	p := rpfNewProxy("http://unused", &rpfLocalHelper{}, nil)
	info := rpfProxyInfo()

	require.Equal(t, artifact.RegistryTypeUPSTREAM, p.GetArtifactType())
	require.Equal(t, []artifact.PackageType{artifact.PackageTypeNUGET}, p.GetPackageTypes())

	_, _, err := p.UploadPackage(ctx, info, nil, 0)
	require.Error(t, err)
	_, err = p.DeletePackage(ctx, info)
	require.Error(t, err)
	_, err = p.GetPackageVersionMetadata(ctx, info)
	require.Error(t, err)
	_, err = p.GetReadme(ctx, info)
	require.Error(t, err)

	v3 := p.GetServiceEndpoint(ctx, info)
	require.Equal(t, "https://harness.test/pkg/root/reg/nuget/query", v3.Resources[0].ID)
	v2 := p.GetServiceEndpointV2(ctx, info)
	require.Equal(t, "https://harness.test/pkg/root/reg/nuget", v2.Base)
	require.NotNil(t, p.GetServiceMetadataV2(ctx, info))
}

func TestRpfProxyV3(t *testing.T) {
	ctx := context.Background()
	srv := rpfUpstream(t)
	p := rpfNewProxy(srv.URL+"/v3/index.json", &rpfLocalHelper{}, nil)
	info := rpfProxyInfo()

	t.Run("list versions", func(t *testing.T) {
		got, err := p.ListPackageVersion(ctx, info)
		require.NoError(t, err)
		require.Equal(t, []string{"1.0.0", "2.0.0"}, got.Versions)
	})

	t.Run("registration index is rewritten to the proxy", func(t *testing.T) {
		got, err := p.GetPackageMetadata(ctx, info)
		require.NoError(t, err)
		resp, ok := got.(*nugettype.RegistrationIndexResponse)
		require.True(t, ok)
		require.Equal(t, "https://harness.test/pkg/root/reg/nuget/registration/demo/index.json",
			resp.RegistrationIndexURL)
		require.Len(t, resp.Pages, 1)
		require.Contains(t, resp.Pages[0].RegistrationPageURL, "proxy_endpoint=")
		item := resp.Pages[0].Items[0]
		require.Equal(t, "https://harness.test/pkg/root/reg/nuget/package/demo/1.0.0/demo.1.0.0.nupkg",
			item.PackageContentURL)
		require.Contains(t, item.RegistrationLeafURL, "proxy_endpoint=")
	})

	t.Run("registration page through a proxy endpoint", func(t *testing.T) {
		pageInfo := info
		pageInfo.ProxyEndpoint = srv.URL + "/v3/reg/demo/page.json"
		got, err := p.GetPackageMetadata(ctx, pageInfo)
		require.NoError(t, err)
		resp, ok := got.(*nugettype.RegistrationIndexPageResponse)
		require.True(t, ok)
		require.Contains(t, resp.RegistrationPageURL, "proxy_endpoint=")
		require.Len(t, resp.Items, 1)
	})

	t.Run("search results are rewritten", func(t *testing.T) {
		got, err := p.SearchPackage(ctx, info, "demo", 10, 0)
		require.NoError(t, err)
		require.Equal(t, int64(1), got.TotalHits)
		require.Len(t, got.Data, 1)
		require.Equal(t, "https://harness.test/pkg/root/reg/nuget/registration/demo/index.json",
			got.Data[0].RegistrationIndexURL)
		require.Contains(t, got.Data[0].Versions[0].RegistrationLeafURL, "proxy_endpoint=https://up/leaf")
	})

	t.Run("download streams from upstream and caches locally", func(t *testing.T) {
		local := &rpfLocalHelper{uploaded: make(chan string, 1)}
		pp := rpfNewProxy(srv.URL+"/v3/index.json", local, nil)
		headers, reader, redirect, body, err := pp.DownloadPackage(ctx, info)
		require.NoError(t, err)
		require.Nil(t, headers)
		require.Nil(t, reader)
		require.Empty(t, redirect)
		data, err := io.ReadAll(body)
		require.NoError(t, err)
		require.Equal(t, "nupkg-bytes", string(data))
		select {
		case got := <-local.uploaded:
			require.Equal(t, "demo:nupkg-bytes", got)
		case <-time.After(5 * time.Second):
			t.Fatal("package was not cached locally")
		}
	})
}

func TestRpfProxyV3Failures(t *testing.T) {
	ctx := context.Background()
	srv := rpfUpstream(t)
	info := rpfProxyInfo()

	t.Run("store errors", func(t *testing.T) {
		p := rpfNewProxy(srv.URL+"/v3/index.json", &rpfLocalHelper{}, errors.New("db down"))
		_, _, _, _, err := p.DownloadPackage(ctx, info)
		require.Error(t, err)
		_, err = p.ListPackageVersion(ctx, info)
		require.Error(t, err)
		_, err = p.GetPackageMetadata(ctx, info)
		require.Error(t, err)
		_, err = p.SearchPackage(ctx, info, "x", 1, 0)
		require.Error(t, err)
		_, err = p.SearchPackageV2(ctx, info, "x", 1, 0)
		require.Error(t, err)
		_, err = p.ListPackageVersionV2(ctx, info)
		require.Error(t, err)
		_, err = p.GetPackageVersionMetadataV2(ctx, info)
		require.Error(t, err)
		_, err = p.CountPackageV2(ctx, info, "x")
		require.Error(t, err)
		_, err = p.CountPackageVersionV2(ctx, info)
		require.Error(t, err)
	})

	t.Run("unreachable upstream", func(t *testing.T) {
		dead := httptest.NewServer(http.NotFoundHandler())
		url := dead.URL
		dead.Close()
		p := rpfNewProxy(url+"/v3/index.json", &rpfLocalHelper{}, nil)
		_, err := p.ListPackageVersion(ctx, info)
		require.Error(t, err)
		_, err = p.GetPackageMetadata(ctx, info)
		require.Error(t, err)
		_, err = p.SearchPackage(ctx, info, "x", 1, 0)
		require.Error(t, err)
		_, err = p.SearchPackageV2(ctx, info, "x", 1, 0)
		require.Error(t, err)
		_, err = p.ListPackageVersionV2(ctx, info)
		require.Error(t, err)
		_, err = p.GetPackageVersionMetadataV2(ctx, info)
		require.Error(t, err)
		_, err = p.CountPackageV2(ctx, info, "x")
		require.Error(t, err)
		_, err = p.CountPackageVersionV2(ctx, info)
		require.Error(t, err)
		_, _, _, _, err = p.DownloadPackage(ctx, info)
		require.Error(t, err)
	})

	t.Run("undecodable upstream payloads", func(t *testing.T) {
		// The service index itself is not JSON, so every v3 call fails on decoding.
		p := rpfNewProxy(srv.URL+"/broken/index.json", &rpfLocalHelper{}, nil)
		_, err := p.ListPackageVersion(ctx, info)
		require.Error(t, err)
		_, err = p.SearchPackage(ctx, info, "x", 1, 0)
		require.Error(t, err)
	})

	t.Run("unsupported auth type", func(t *testing.T) {
		p := rpfNewProxy(srv.URL, &rpfLocalHelper{}, nil)
		p.proxyStore = &rpfProxyStore{proxy: &types.UpstreamProxy{RepoURL: srv.URL, RepoAuthType: "bogus"}}
		_, err := p.ListPackageVersion(ctx, info)
		require.ErrorContains(t, err, "unsupported auth type")
		_, _, _, _, err = p.DownloadPackage(ctx, info)
		require.ErrorContains(t, err, "unsupported auth type")
	})
}

func TestRpfProxyDownloadLocal(t *testing.T) {
	ctx := context.Background()
	srv := rpfUpstream(t)
	info := rpfProxyInfo()

	t.Run("served from the local cache", func(t *testing.T) {
		p := rpfNewProxy(srv.URL+"/v3/index.json", &rpfLocalHelper{exists: true}, nil)
		headers, _, redirect, body, err := p.DownloadPackage(ctx, info)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, headers.Code)
		require.Equal(t, "https://redirect.test/file", redirect)
		require.Nil(t, body)
	})

	t.Run("local failure falls back to upstream", func(t *testing.T) {
		local := &rpfLocalHelper{exists: true, downloadErr: errors.New("corrupt"), uploaded: make(chan string, 1)}
		p := rpfNewProxy(srv.URL+"/v3/index.json", local, nil)
		_, _, _, body, err := p.DownloadPackage(ctx, info)
		require.NoError(t, err)
		data, err := io.ReadAll(body)
		require.NoError(t, err)
		require.Equal(t, "nupkg-bytes", string(data))
		select {
		case <-local.uploaded:
		case <-time.After(5 * time.Second):
			t.Fatal("package was not cached locally")
		}
	})
}

func TestRpfProxyV2(t *testing.T) {
	ctx := context.Background()
	srv := rpfUpstream(t)
	p := rpfNewProxy(srv.URL+"/v2", &rpfLocalHelper{}, nil)
	info := rpfProxyInfo()

	t.Run("counts", func(t *testing.T) {
		n, err := p.CountPackageVersionV2(ctx, info)
		require.NoError(t, err)
		require.Equal(t, int64(2), n)
		n, err = p.CountPackageV2(ctx, info, "demo")
		require.NoError(t, err)
		require.Equal(t, int64(5), n)
	})

	checkFeed := func(t *testing.T, feed *nugettype.FeedResponse) {
		t.Helper()
		require.Equal(t, "https://harness.test/pkg/root/reg/nuget", feed.Base)
		require.Len(t, feed.Links, 1)
		require.Len(t, feed.Entries, 1)
		entry := feed.Entries[0]
		require.Empty(t, entry.ID)
		require.Nil(t, entry.DownloadContent)
		require.Contains(t, entry.Content, "https://harness.test/pkg/root/reg/nuget/Packages(Id='demo',Version='1.0.0')")
		require.Contains(t, entry.Content, "proxy_endpoint="+"http://")
	}

	t.Run("list versions", func(t *testing.T) {
		feed, err := p.ListPackageVersionV2(ctx, info)
		require.NoError(t, err)
		checkFeed(t, feed)
	})

	t.Run("search", func(t *testing.T) {
		feed, err := p.SearchPackageV2(ctx, info, "demo", 10, 0)
		require.NoError(t, err)
		checkFeed(t, feed)
	})

	t.Run("version metadata", func(t *testing.T) {
		entry, err := p.GetPackageVersionMetadataV2(ctx, info)
		require.NoError(t, err)
		require.Empty(t, entry.ID)
		require.True(t, strings.Contains(entry.Content, "https://harness.test/pkg/root/reg/nuget/Packages"))
	})

	t.Run("download from v2 upstream", func(t *testing.T) {
		dl := info
		dl.ProxyEndpoint = srv.URL + "/v2/package/demo/1.0.0"
		_, _, _, body, err := p.DownloadPackage(ctx, dl)
		require.NoError(t, err)
		data, err := io.ReadAll(body)
		require.NoError(t, err)
		require.Equal(t, "v2-nupkg", string(data))
	})

	t.Run("unknown package", func(t *testing.T) {
		missing := info
		missing.Image = "missing"
		_, err := p.GetPackageVersionMetadataV2(ctx, missing)
		require.Error(t, err)
	})
}

func rpfSpaceFinder() refcache.SpaceFinder {
	return refcache.SpaceFinder{}
}
