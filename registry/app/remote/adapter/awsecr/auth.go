// Source: https://github.com/goharbor/harbor

// Copyright 2016 Project Harbor Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
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
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/harness/gitness/app/services/refcache"
	api "github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	commonhttp "github.com/harness/gitness/registry/app/common/http"
	"github.com/harness/gitness/registry/app/common/http/modifier"
	"github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/secret"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	awsecrapi "github.com/aws/aws-sdk-go/service/ecr"
	"github.com/rs/zerolog/log"
)

// Credential ...
type Credential modifier.Modifier

// Implements interface Credential.
type awsAuthCredential struct {
	accessKey string
	awssvc    *awsecrapi.ECR

	cacheToken   *cacheToken
	cacheExpired *time.Time
	isPublic     bool
}

type cacheToken struct {
	endpoint string
	user     string
	password string
	host     string
}

// DefaultCacheExpiredTime is expired timeout for aws auth token.
const DefaultCacheExpiredTime = time.Hour * 1

func (a *awsAuthCredential) Modify(req *http.Request) error {
	// url maybe redirect to s3
	if !strings.Contains(req.URL.Host, ".ecr.") {
		return nil
	}
	if !a.isTokenValid() {
		endpoint, user, pass, expiresAt, err := a.getAuthorization(req.URL.String(), req.URL.Host)

		if err != nil {
			return err
		}
		u, err := url.Parse(endpoint)
		if err != nil {
			return err
		}
		a.cacheToken = &cacheToken{}
		a.cacheToken.host = u.Host
		a.cacheToken.user = user
		a.cacheToken.password = pass
		a.cacheToken.endpoint = endpoint
		t := time.Now().Add(DefaultCacheExpiredTime)
		if expiresAt == nil || t.Before(*expiresAt) {
			a.cacheExpired = &t
		} else {
			a.cacheExpired = expiresAt
		}
	}
	req.Host = a.cacheToken.host
	req.URL.Host = a.cacheToken.host
	if a.isPublic {
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", a.cacheToken.password))
		return nil
	}
	req.SetBasicAuth(a.cacheToken.user, a.cacheToken.password)
	return nil
}

func getAwsSvc(accessKey, secretKey string, reg types.UpstreamProxy) (*awsecrapi.ECR, error) {
	_, region, err := parseAccountRegion(reg.RepoURL)
	if err != nil {
		return nil, err
	}
	sess, err := session.NewSession()
	if err != nil {
		return nil, err
	}
	var cred *credentials.Credentials
	log.Info().Msgf("Aws Ecr getAuthorization %s", accessKey)
	if accessKey != "" {
		cred = credentials.NewStaticCredentials(
			accessKey,
			secretKey,
			"")
	}

	config := &aws.Config{
		Credentials: cred,
		Region:      &region,
		HTTPClient: &http.Client{
			Transport: commonhttp.GetHTTPTransport(commonhttp.WithInsecure(false)),
		},
	}

	svc := awsecrapi.New(sess, config)
	return svc, nil
}

func parseAccountRegion(url string) (string, string, error) {
	rs := ecrRegexp.FindStringSubmatch(url)
	if len(rs) < 4 {
		return "", "", errors.New("bad aws url")
	}
	return rs[1], rs[3], nil
}

func getCreds(
	ctx context.Context, spaceFinder refcache.SpaceFinder, secretService secret.Service, reg types.UpstreamProxy,
) (string, string, bool, error) {
	if api.AuthType(reg.RepoAuthType) == api.AuthTypeAnonymous {
		return "", "", true, nil
	}
	if api.AuthType(reg.RepoAuthType) != api.AuthTypeAccessKeySecretKey {
		log.Debug().Msgf("invalid auth type: %s", reg.RepoAuthType)
		return "", "", false, nil
	}
	secretKey, err := getSecretValue(ctx, spaceFinder, secretService, reg.SecretSpaceID,
		reg.SecretIdentifier)
	if err != nil {
		return "", "", false, err
	}
	if reg.UserName != "" {
		return reg.UserName, secretKey, false, nil
	}
	accessKey, err := getSecretValue(ctx, spaceFinder, secretService, reg.UserNameSecretSpaceID,
		reg.UserNameSecretIdentifier)
	if err != nil {
		return "", "", false, err
	}
	return accessKey, secretKey, false, nil
}

func getSecretValue(ctx context.Context, spaceFinder refcache.SpaceFinder, secretService secret.Service,
	secretSpaceID int64, secretSpacePath string) (string, error) {
	spacePath, err := spaceFinder.FindByID(ctx, secretSpaceID)
	if err != nil {
		log.Error().Msgf("failed to find space path: %v", err)
		return "", err
	}
	decryptSecret, err := secretService.DecryptSecret(ctx, spacePath.Path, secretSpacePath)
	if err != nil {
		log.Error().Msgf("failed to decrypt secret: %v", err)
		return "", err
	}
	return decryptSecret, nil
}

func (a *awsAuthCredential) getAuthorization(url, host string) (string, string, string, *time.Time, error) {
	if a.isPublic {
		token, err := a.getPublicECRToken(host)
		if err != nil {
			return "", "", "", nil, err
		}
		return url, "", token, nil, nil
	}
	id, _, err := parseAccountRegion(url)
	if err != nil {
		return "", "", "", nil, err
	}

	var input *awsecrapi.GetAuthorizationTokenInput
	if id != "" {
		input = &awsecrapi.GetAuthorizationTokenInput{RegistryIds: []*string{&id}}
	}
	svc := a.awssvc
	result, err := svc.GetAuthorizationToken(input)
	if err != nil {
		var awsErr *awserr.Error

		if errors.As(err, awsErr) {
			return "", "", "", nil, fmt.Errorf("%s", err.Error())
		}

		return "", "", "", nil, err
	}

	// Double check
	if len(result.AuthorizationData) == 0 {
		return "", "", "", nil, errors.New("no authorization token returned")
	}

	theOne := result.AuthorizationData[0]
	expiresAt := theOne.ExpiresAt
	payload, _ := base64.StdEncoding.DecodeString(*theOne.AuthorizationToken)
	pair := strings.SplitN(string(payload), ":", 2)

	log.Debug().Msgf("Aws Ecr getAuthorization %s result: %d %s...", a.accessKey, len(pair[1]), pair[1][:25])

	return *(theOne.ProxyEndpoint), pair[0], pair[1], expiresAt, nil
}

func (a *awsAuthCredential) isTokenValid() bool {
	if a.cacheToken == nil {
		return false
	}
	if a.cacheExpired == nil {
		return false
	}
	if time.Now().After(*a.cacheExpired) {
		a.cacheExpired = nil
		a.cacheToken = nil
		return false
	}
	return true
}

// NewAuth new aws auth.
func NewAuth(accessKey string, awssvc *awsecrapi.ECR, isPublic bool) Credential {
	return &awsAuthCredential{
		accessKey: accessKey,
		awssvc:    awssvc,
		isPublic:  isPublic,
	}
}

func (a *awsAuthCredential) getPublicECRToken(host string) (string, error) {
	c := &http.Client{
		Transport: commonhttp.GetHTTPTransport(commonhttp.WithInsecure(true)),
	}
	req, err := http.NewRequestWithContext(context.TODO(), http.MethodGet, buildTokenURL(host, host), nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := c.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("non-200 response: %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	// Parse the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	// Unmarshal JSON
	var tokenResponse TokenResponse
	if err := json.Unmarshal(body, &tokenResponse); err != nil {
		return "", fmt.Errorf("failed to parse token response: %w", err)
	}

	return tokenResponse.Token, nil
}

type TokenResponse struct {
	Token string `json:"token"`
}

func buildTokenURL(host, service string) string {
	return fmt.Sprintf("https://%s/token?service=%s", host, service)
}
