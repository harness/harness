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

package awsecr

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/types"

	"github.com/stretchr/testify/require"
)

func TestW2rParseAccountRegion(t *testing.T) {
	account, region, err := parseAccountRegion("https://123456789012.dkr.ecr.eu-west-1.amazonaws.com")
	require.NoError(t, err)
	require.Equal(t, "123456789012", account)
	require.Equal(t, "eu-west-1", region)

	account, region, err = parseAccountRegion("https://api.ecr.us-east-1.amazonaws.com")
	require.NoError(t, err)
	require.Empty(t, account)
	require.Equal(t, "us-east-1", region)

	_, _, err = parseAccountRegion("https://example.com")
	require.Error(t, err)
}

func TestW2rGetCredsByAuthType(t *testing.T) {
	ctx := context.Background()
	finder := refcache.SpaceFinder{}

	user, secret, anonymous, err := getCreds(ctx, finder, nil, types.UpstreamProxy{
		RepoAuthType: string(artifact.AuthTypeAnonymous),
	})
	require.NoError(t, err)
	require.True(t, anonymous)
	require.Empty(t, user+secret)

	user, secret, anonymous, err = getCreds(ctx, finder, nil, types.UpstreamProxy{RepoAuthType: "UserPassword"})
	require.NoError(t, err)
	require.False(t, anonymous)
	require.Empty(t, user+secret, "only access key and secret key auth is supported")

	// Without secret identifiers there is nothing to look up.
	user, secret, anonymous, err = getCreds(ctx, finder, nil, types.UpstreamProxy{
		RepoAuthType: string(artifact.AuthTypeAccessKeySecretKey), UserName: "AKIAEXAMPLE",
	})
	require.NoError(t, err)
	require.False(t, anonymous)
	require.Equal(t, "AKIAEXAMPLE", user)
	require.Empty(t, secret)

	user, _, _, err = getCreds(ctx, finder, nil, types.UpstreamProxy{
		RepoAuthType: string(artifact.AuthTypeAccessKeySecretKey),
	})
	require.NoError(t, err)
	require.Empty(t, user)

	value, err := getSecretValue(ctx, finder, nil, "parent", 0, "")
	require.NoError(t, err)
	require.Empty(t, value)
}

func TestW2rGetAwsSvc(t *testing.T) {
	ctx := context.Background()
	reg := types.UpstreamProxy{RepoURL: "https://123456789012.dkr.ecr.us-east-1.amazonaws.com"}

	svc, err := getAwsSvc(ctx, "AKIAEXAMPLE", "secret-value", reg)
	require.NoError(t, err)
	require.NotNil(t, svc)
	require.Equal(t, "us-east-1", *svc.Config.Region)

	anonymous, err := getAwsSvc(ctx, "", "", reg)
	require.NoError(t, err)
	require.NotNil(t, anonymous)

	_, err = getAwsSvc(ctx, "", "", types.UpstreamProxy{RepoURL: "https://example.com"})
	require.Error(t, err)
}

func TestW2rAuthModifyUsesCachedToken(t *testing.T) {
	future := time.Now().Add(time.Hour)
	cred := &awsAuthCredential{
		cacheToken:   &cacheToken{host: "registry.example.com", user: "AWS", password: "token", endpoint: "https://x"},
		cacheExpired: &future,
	}

	const ecrURL = "https://123.dkr.ecr.us-east-1.amazonaws.com/v2/"
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, ecrURL, nil)
	require.NoError(t, err)
	require.NoError(t, cred.Modify(req))
	user, pass, ok := req.BasicAuth()
	require.True(t, ok)
	require.Equal(t, "AWS", user)
	require.Equal(t, "token", pass)
	require.Equal(t, "registry.example.com", req.URL.Host)

	public := &awsAuthCredential{cacheToken: cred.cacheToken, cacheExpired: &future, isPublic: true}
	req, err = http.NewRequestWithContext(context.Background(), http.MethodGet, "https://public.ecr.aws/v2/", nil)
	require.NoError(t, err)
	require.NoError(t, public.Modify(req))
	require.Equal(t, "Bearer token", req.Header.Get("Authorization"))

	// Requests to other hosts are not touched.
	other, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://example.com/v2/", nil)
	require.NoError(t, err)
	require.NoError(t, cred.Modify(other))
	require.Empty(t, other.Header.Get("Authorization"))
	require.Equal(t, "example.com", other.URL.Host)
}

func TestW2rAuthTokenValidity(t *testing.T) {
	require.False(t, (&awsAuthCredential{}).isTokenValid())
	require.False(t, (&awsAuthCredential{cacheToken: &cacheToken{}}).isTokenValid())

	past := time.Now().Add(-time.Minute)
	expired := &awsAuthCredential{cacheToken: &cacheToken{}, cacheExpired: &past}
	require.False(t, expired.isTokenValid())
	require.Nil(t, expired.cacheToken, "an expired token is dropped")

	future := time.Now().Add(time.Minute)
	require.True(t, (&awsAuthCredential{cacheToken: &cacheToken{}, cacheExpired: &future}).isTokenValid())
}

func TestW2rAuthHelpers(t *testing.T) {
	tokenURL := buildTokenURL("public.ecr.aws", "public.ecr.aws")
	require.Equal(t, "https://public.ecr.aws/token?service=public.ecr.aws", tokenURL)

	cred := NewAuth("AKIAEXAMPLE", nil, true)
	require.NotNil(t, cred)

	a := &adapter{}
	status, err := a.HealthCheck()
	require.NoError(t, err)
	require.Equal(t, "Not implemented", status)
	name, err := a.GetImageName("library/busybox")
	require.NoError(t, err)
	require.Equal(t, "library/busybox", name)
}
