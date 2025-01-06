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
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/harness/gitness/app/store"
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
		endpoint, user, pass, expiresAt, err := a.getAuthorization(req.URL.String())

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
		if t.Before(*expiresAt) {
			a.cacheExpired = &t
		} else {
			a.cacheExpired = expiresAt
		}
	}
	req.Host = a.cacheToken.host
	req.URL.Host = a.cacheToken.host
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
	if rs == nil || len(rs) < 4 {
		return "", "", errors.New("bad aws url")
	}
	return rs[1], rs[3], nil
}

func getCreds(
	ctx context.Context, spacePathStore store.SpacePathStore, secretService secret.Service, reg types.UpstreamProxy,
) (string, string, error) {
	if api.AuthType(reg.RepoAuthType) != api.AuthTypeAccessKeySecretKey {
		log.Debug().Msgf("invalid auth type: %s", reg.RepoAuthType)
		return "", "", nil
	}
	secretKey, err := getSecretValue(ctx, spacePathStore, secretService, reg.SecretSpaceID,
		reg.SecretIdentifier)
	if err != nil {
		return "", "", err
	}
	if reg.UserName != "" {
		return reg.UserName, secretKey, nil
	}
	accessKey, err := getSecretValue(ctx, spacePathStore, secretService, reg.UserNameSecretSpaceID,
		reg.UserNameSecretIdentifier)
	if err != nil {
		return "", "", err
	}
	return accessKey, secretKey, nil
}

func getSecretValue(ctx context.Context, spacePathStore store.SpacePathStore, secretService secret.Service,
	secretSpaceID int64, secretSpacePath string) (string, error) {
	spacePath, err := spacePathStore.FindPrimaryBySpaceID(ctx, secretSpaceID)
	if err != nil {
		log.Error().Msgf("failed to find space path: %v", err)
		return "", err
	}
	decryptSecret, err := secretService.DecryptSecret(ctx, spacePath.Value, secretSpacePath)
	if err != nil {
		log.Error().Msgf("failed to decrypt secret: %v", err)
		return "", err
	}
	return decryptSecret, nil
}

func (a *awsAuthCredential) getAuthorization(url string) (string, string, string, *time.Time, error) {
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
func NewAuth(accessKey string, awssvc *awsecrapi.ECR) Credential {
	return &awsAuthCredential{
		accessKey: accessKey,
		awssvc:    awssvc,
	}
}
