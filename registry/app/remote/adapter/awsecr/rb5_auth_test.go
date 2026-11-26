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
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/harness/gitness/app/services/refcache"
	storecache "github.com/harness/gitness/app/store/cache"
	api "github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/types"
	basestore "github.com/harness/gitness/store"
	coretypes "github.com/harness/gitness/types"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	awsecrapi "github.com/aws/aws-sdk-go/service/ecr"
	"github.com/stretchr/testify/require"
)

var errRb5Boom = errors.New("rb5 boom")

const rb5PrivateURL = "https://123456789012.dkr.ecr.us-east-1.amazonaws.com"

// rb5Password is long enough for the adapter's debug logging and has low entropy.
var rb5Password = strings.Repeat("pw", 20)

type rb5IDCache struct {
	spaces map[int64]*coretypes.SpaceCore
}

func (c rb5IDCache) Get(_ context.Context, id int64) (*coretypes.SpaceCore, error) {
	space, ok := c.spaces[id]
	if !ok {
		return nil, basestore.ErrResourceNotFound
	}
	return space, nil
}
func (rb5IDCache) Stats() (int64, int64)        { return 0, 0 }
func (rb5IDCache) Evict(context.Context, int64) {}
func (rb5IDCache) Map(context.Context, []int64) (map[int64]*coretypes.SpaceCore, error) {
	return map[int64]*coretypes.SpaceCore{}, nil
}

type rb5PathCache struct {
	spaces map[int64]*coretypes.SpaceCore
}

func (c rb5PathCache) Get(_ context.Context, path string) (*coretypes.SpacePath, error) {
	for _, space := range c.spaces {
		if space.Path == path {
			return &coretypes.SpacePath{Value: path, IsPrimary: true, SpaceID: space.ID}, nil
		}
	}
	return nil, basestore.ErrResourceNotFound
}
func (rb5PathCache) Stats() (int64, int64)         { return 0, 0 }
func (rb5PathCache) Evict(context.Context, string) {}

type rb5FoldCache struct{}

func (rb5FoldCache) Get(context.Context, string) (int64, error) {
	return 0, basestore.ErrResourceNotFound
}
func (rb5FoldCache) Stats() (int64, int64)         { return 0, 0 }
func (rb5FoldCache) Evict(context.Context, string) {}

func rb5SpaceFinder() refcache.SpaceFinder {
	spaces := map[int64]*coretypes.SpaceCore{
		1: {ID: 1, Path: "acct", Identifier: "acct"},
		2: {ID: 2, Path: "acct/org", Identifier: "org"},
		3: {ID: 3, Path: "other/org", Identifier: "org"},
	}
	return refcache.NewSpaceFinder(
		rb5IDCache{spaces: spaces}, rb5PathCache{spaces: spaces}, rb5FoldCache{},
		storecache.Evictor[*coretypes.SpaceCore]{},
	)
}

type rb5Secrets struct {
	values map[string]string
}

func (s rb5Secrets) DecryptSecret(_ context.Context, spacePath, identifier string) (string, error) {
	value, ok := s.values[spacePath+"/"+identifier]
	if !ok {
		return "", errRb5Boom
	}
	return value, nil
}

func TestRb5ParseAccountRegion(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		account string
		region  string
		wantErr bool
	}{
		{"private", rb5PrivateURL, "123456789012", "us-east-1", false},
		{"fips", "https://123456789012.dkr.ecr-fips.us-gov-west-1.amazonaws.com", "123456789012", "us-gov-west-1", false},
		{"api endpoint has no account", "https://api.ecr.eu-west-1.amazonaws.com", "", "eu-west-1", false},
		{"china", "https://123456789012.dkr.ecr.cn-north-1.amazonaws.com.cn", "123456789012", "cn-north-1", false},
		{"not ecr", "https://example.com", "", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			account, region, err := parseAccountRegion(tt.url)
			if tt.wantErr {
				require.ErrorContains(t, err, "bad aws url")
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.account, account)
			require.Equal(t, tt.region, region)
		})
	}
	require.Equal(t, "https://h/token?service=s", buildTokenURL("h", "s"))
}

func TestRb5GetSecretValue(t *testing.T) {
	ctx := context.Background()
	finder := rb5SpaceFinder()
	secrets := rb5Secrets{values: map[string]string{"acct/org/key": "s3cret"}}

	value, err := getSecretValue(ctx, finder, secrets, "acct", 0, "key")
	require.NoError(t, err)
	require.Empty(t, value)
	value, err = getSecretValue(ctx, finder, secrets, "acct", 2, "")
	require.NoError(t, err)
	require.Empty(t, value)

	value, err = getSecretValue(ctx, finder, secrets, "acct", 2, "key")
	require.NoError(t, err)
	require.Equal(t, "s3cret", value)

	_, err = getSecretValue(ctx, finder, secrets, "acct", 99, "key")
	require.Error(t, err)

	// A secret from another account is refused.
	_, err = getSecretValue(ctx, finder, secrets, "acct", 3, "key")
	require.Error(t, err)

	_, err = getSecretValue(ctx, finder, secrets, "acct", 2, "unknown")
	require.ErrorIs(t, err, errRb5Boom)
}

func TestRb5GetCreds(t *testing.T) {
	ctx := context.Background()
	finder := rb5SpaceFinder()
	secrets := rb5Secrets{values: map[string]string{
		"acct/org/secret": "sk", "acct/org/access": "ak",
	}}
	base := types.UpstreamProxy{
		ParentID: "acct", RepoAuthType: string(api.AuthTypeAccessKeySecretKey),
		SecretSpaceID: 2, SecretIdentifier: "secret",
	}

	anonymous := base
	anonymous.RepoAuthType = string(api.AuthTypeAnonymous)
	user, pass, public, err := getCreds(ctx, finder, secrets, anonymous)
	require.NoError(t, err)
	require.True(t, public)
	require.Empty(t, user+pass)

	unsupported := base
	unsupported.RepoAuthType = string(api.AuthTypeUserPassword)
	user, pass, public, err = getCreds(ctx, finder, secrets, unsupported)
	require.NoError(t, err)
	require.False(t, public)
	require.Empty(t, user+pass)

	withUser := base
	withUser.UserName = "plain-access"
	user, pass, public, err = getCreds(ctx, finder, secrets, withUser)
	require.NoError(t, err)
	require.False(t, public)
	require.Equal(t, "plain-access", user)
	require.Equal(t, "sk", pass)

	withSecretUser := base
	withSecretUser.UserNameSecretSpaceID = 2
	withSecretUser.UserNameSecretIdentifier = "access"
	user, pass, _, err = getCreds(ctx, finder, secrets, withSecretUser)
	require.NoError(t, err)
	require.Equal(t, "ak", user)
	require.Equal(t, "sk", pass)

	brokenSecret := base
	brokenSecret.SecretIdentifier = "unknown"
	_, _, _, err = getCreds(ctx, finder, secrets, brokenSecret)
	require.Error(t, err)

	brokenUser := withSecretUser
	brokenUser.UserNameSecretIdentifier = "unknown"
	_, _, _, err = getCreds(ctx, finder, secrets, brokenUser)
	require.Error(t, err)
}

// rb5ECR starts a fake ECR API and returns a client pointed at it together with the call counter.
func rb5ECR(t *testing.T, handler func(w http.ResponseWriter)) (*awsecrapi.ECR, *atomic.Int32) {
	t.Helper()
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/x-amz-json-1.1")
		handler(w)
	}))
	t.Cleanup(srv.Close)

	sess, err := session.NewSession()
	require.NoError(t, err)
	svc := awsecrapi.New(sess, aws.NewConfig().
		WithEndpoint(srv.URL).WithRegion("us-east-1").WithDisableSSL(true).
		WithCredentials(credentials.NewStaticCredentials("ak", "sk", "")))
	return svc, &calls
}

func rb5TokenResponse(proxyEndpoint string, expiresAt *int64) func(w http.ResponseWriter) {
	return func(w http.ResponseWriter) {
		entry := map[string]any{
			"authorizationToken": base64.StdEncoding.EncodeToString([]byte("AWS:" + rb5Password)),
			"proxyEndpoint":      proxyEndpoint,
		}
		if expiresAt != nil {
			entry["expiresAt"] = *expiresAt
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"authorizationData": []any{entry}})
	}
}

func TestRb5GetAuthorization(t *testing.T) {
	ctx := context.Background()

	svc, calls := rb5ECR(t, rb5TokenResponse(rb5PrivateURL, nil))
	auth := &awsAuthCredential{accessKey: "ak", awssvc: svc}
	endpoint, user, pass, expires, err := auth.getAuthorization(ctx, rb5PrivateURL, "host")
	require.NoError(t, err)
	require.Equal(t, rb5PrivateURL, endpoint)
	require.Equal(t, "AWS", user)
	require.Equal(t, rb5Password, pass)
	require.Nil(t, expires)
	require.Equal(t, int32(1), calls.Load())

	// The account is optional for the api endpoint.
	_, _, _, _, err = auth.getAuthorization(ctx, "https://api.ecr.us-east-1.amazonaws.com", "host")
	require.NoError(t, err)

	_, _, _, _, err = auth.getAuthorization(ctx, "https://example.com", "host")
	require.ErrorContains(t, err, "bad aws url")

	empty, _ := rb5ECR(t, func(w http.ResponseWriter) { _, _ = w.Write([]byte(`{"authorizationData":[]}`)) })
	_, _, _, _, err = (&awsAuthCredential{awssvc: empty}).getAuthorization(ctx, rb5PrivateURL, "host")
	require.ErrorContains(t, err, "no authorization token returned")
}

func TestRb5GetPublicToken(t *testing.T) {
	ctx := context.Background()

	respond := func(status int, body string) *httptest.Server {
		srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/token" {
				http.NotFound(w, r)
				return
			}
			w.WriteHeader(status)
			_, _ = w.Write([]byte(body))
		}))
		t.Cleanup(srv.Close)
		return srv
	}
	hostOf := func(srv *httptest.Server) string { return strings.TrimPrefix(srv.URL, "https://") }
	auth := &awsAuthCredential{isPublic: true}

	ok := respond(http.StatusOK, `{"token":"public-token"}`)
	token, err := auth.getPublicECRToken(ctx, hostOf(ok))
	require.NoError(t, err)
	require.Equal(t, "public-token", token)

	endpoint, user, pass, expires, err := auth.getAuthorization(ctx, "https://public.example/v2/", hostOf(ok))
	require.NoError(t, err)
	require.Equal(t, "https://public.example/v2/", endpoint)
	require.Empty(t, user)
	require.Equal(t, "public-token", pass)
	require.Nil(t, expires)

	_, err = auth.getPublicECRToken(ctx, hostOf(respond(http.StatusForbidden, "")))
	require.ErrorContains(t, err, "non-200 response: 403 Forbidden")

	_, err = auth.getPublicECRToken(ctx, hostOf(respond(http.StatusOK, "not json")))
	require.ErrorContains(t, err, "failed to parse token response")

	_, _, _, _, err = auth.getAuthorization(ctx, "https://public.example/v2/", hostOf(respond(http.StatusBadGateway, "")))
	require.Error(t, err)

	closed := respond(http.StatusOK, "{}")
	host := hostOf(closed)
	closed.Close()
	_, err = auth.getPublicECRToken(ctx, host)
	require.ErrorContains(t, err, "failed to fetch token")

	_, err = auth.getPublicECRToken(ctx, "bad host/with space")
	require.Error(t, err)
}

func rb5Request(t *testing.T, rawURL string) *http.Request {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, rawURL, nil)
	require.NoError(t, err)
	return req
}

func TestRb5ModifyPrivate(t *testing.T) {
	proxy := "https://123456789012.dkr.ecr.us-east-1.amazonaws.com"
	svc, calls := rb5ECR(t, rb5TokenResponse(proxy, nil))
	cred, ok := NewAuth("ak", svc, false).(*awsAuthCredential)
	require.True(t, ok)

	// Hosts outside of ECR are left alone.
	other := rb5Request(t, "https://example.com/v2/")
	require.NoError(t, cred.Modify(other))
	require.Empty(t, other.Header.Get("Authorization"))
	require.Equal(t, int32(0), calls.Load())

	req := rb5Request(t, rb5PrivateURL+"/v2/")
	require.NoError(t, cred.Modify(req))
	user, pass, hasAuth := req.BasicAuth()
	require.True(t, hasAuth)
	require.Equal(t, "AWS", user)
	require.Equal(t, rb5Password, pass)
	require.Equal(t, "123456789012.dkr.ecr.us-east-1.amazonaws.com", req.URL.Host)
	require.Equal(t, int32(1), calls.Load())
	require.NotNil(t, cred.cacheExpired)

	// The cached token is reused.
	require.NoError(t, cred.Modify(rb5Request(t, rb5PrivateURL+"/v2/")))
	require.Equal(t, int32(1), calls.Load())

	// An expired token is fetched again.
	past := time.Now().Add(-time.Minute)
	cred.cacheExpired = &past
	require.NoError(t, cred.Modify(rb5Request(t, rb5PrivateURL+"/v2/")))
	require.Equal(t, int32(2), calls.Load())
}

func TestRb5ModifyTokenExpiry(t *testing.T) {
	soon := time.Now().Add(5 * time.Minute).Unix()
	svc, _ := rb5ECR(t, rb5TokenResponse(rb5PrivateURL, &soon))
	cred := &awsAuthCredential{awssvc: svc}
	require.NoError(t, cred.Modify(rb5Request(t, rb5PrivateURL+"/v2/")))
	require.NotNil(t, cred.cacheExpired)
	require.WithinDuration(t, time.Unix(soon, 0), *cred.cacheExpired, 2*time.Second)

	later := time.Now().Add(48 * time.Hour).Unix()
	svc, _ = rb5ECR(t, rb5TokenResponse(rb5PrivateURL, &later))
	cred = &awsAuthCredential{awssvc: svc}
	require.NoError(t, cred.Modify(rb5Request(t, rb5PrivateURL+"/v2/")))
	require.WithinDuration(t, time.Now().Add(DefaultCacheExpiredTime), *cred.cacheExpired, 2*time.Second)
}

func TestRb5ModifyFailures(t *testing.T) {
	// The ECR host does not match the AWS pattern.
	cred := &awsAuthCredential{}
	require.ErrorContains(t, cred.Modify(rb5Request(t, "https://x.ecr.example.com/v2/")), "bad aws url")

	// The proxy endpoint returned by AWS cannot be parsed.
	svc, _ := rb5ECR(t, rb5TokenResponse("http://[::1", nil))
	cred = &awsAuthCredential{awssvc: svc}
	require.Error(t, cred.Modify(rb5Request(t, rb5PrivateURL+"/v2/")))
}

func TestRb5ModifyPublicUsesBearer(t *testing.T) {
	future := time.Now().Add(time.Hour)
	cred, ok := NewAuth("", nil, true).(*awsAuthCredential)
	require.True(t, ok)
	cred.cacheToken = &cacheToken{host: "public.ecr.aws", password: "public-token"}
	cred.cacheExpired = &future

	req := rb5Request(t, "https://public.ecr.aws/v2/")
	require.NoError(t, cred.Modify(req))
	require.Equal(t, "Bearer public-token", req.Header.Get("Authorization"))
}

func TestRb5IsTokenValid(t *testing.T) {
	cred := &awsAuthCredential{}
	require.False(t, cred.isTokenValid())

	cred.cacheToken = &cacheToken{}
	require.False(t, cred.isTokenValid())

	future := time.Now().Add(time.Hour)
	cred.cacheExpired = &future
	require.True(t, cred.isTokenValid())

	past := time.Now().Add(-time.Hour)
	cred.cacheExpired = &past
	require.False(t, cred.isTokenValid())
	require.Nil(t, cred.cacheToken)
	require.Nil(t, cred.cacheExpired)
}

func TestRb5GetAwsSvc(t *testing.T) {
	ctx := context.Background()

	svc, err := getAwsSvc(ctx, "ak", "sk", types.UpstreamProxy{RepoURL: rb5PrivateURL})
	require.NoError(t, err)
	require.NotNil(t, svc)
	require.Equal(t, "us-east-1", *svc.Config.Region)

	// Without an access key the default credential chain is used.
	svc, err = getAwsSvc(ctx, "", "", types.UpstreamProxy{RepoURL: rb5PrivateURL})
	require.NoError(t, err)
	require.NotNil(t, svc)

	_, err = getAwsSvc(ctx, "ak", "sk", types.UpstreamProxy{RepoURL: "https://example.com"})
	require.ErrorContains(t, err, "bad aws url")
}
