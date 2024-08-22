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

package bearer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/harness/gitness/registry/app/common/lib"
	"github.com/harness/gitness/registry/app/common/lib/errors"
)

const (
	cacheCapacity = 100
	cacheLatency  = 10 // second
)

// NewAuthorizer return a bearer token authorizer
// The parameter "a" is an authorizer used to fetch the token.
func NewAuthorizer(realm, service string, a lib.Authorizer, transport http.RoundTripper) lib.Authorizer {
	authorizer := &authorizer{
		realm:      realm,
		service:    service,
		authorizer: a,
		cache:      newCache(cacheCapacity, cacheLatency),
	}

	authorizer.client = &http.Client{Transport: transport}
	return authorizer
}

type authorizer struct {
	realm      string
	service    string
	authorizer lib.Authorizer
	cache      *cache
	client     *http.Client
}

func (a *authorizer) Modify(req *http.Request) error {
	// parse scopes from request
	scopes := parseScopes(req)

	// get token
	token, err := a.getToken(scopes)
	if err != nil {
		return err
	}

	// set authorization header
	if token != nil && len(token.Token) > 0 {
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token.Token))
	}
	return nil
}

func (a *authorizer) getToken(scopes []*scope) (*token, error) {
	// get token from cache first
	token := a.cache.get(scopes)
	if token != nil {
		return token, nil
	}

	// get no token from cache, fetch it from the token service
	token, err := a.fetchToken(scopes)
	if err != nil {
		return nil, err
	}

	// set the token into the cache
	a.cache.set(scopes, token)
	return token, nil
}

type token struct {
	Token       string `json:"token"`
	AccessToken string `json:"access_token"` // the token returned by azure container registry is called "access_token"
	ExpiresIn   int    `json:"expires_in"`
	IssuedAt    string `json:"issued_at"`
}

func (a *authorizer) fetchToken(scopes []*scope) (*token, error) {
	url, err := url.Parse(a.realm)
	if err != nil {
		return nil, err
	}
	query := url.Query()
	query.Add("service", a.service)
	for _, scope := range scopes {
		query.Add("scope", scope.String())
	}
	url.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(context.TODO(), http.MethodGet, url.String(), nil)
	if err != nil {
		return nil, err
	}
	if a.authorizer != nil {
		if err = a.authorizer.Modify(req); err != nil {
			return nil, err
		}
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	token := &token{}
	switch resp.StatusCode {
	case http.StatusOK:
		return getToken(body, token)
	case http.StatusUnauthorized:
		return nil, fmt.Errorf("request with body :%s : %s", string(body), errors.UnAuthorizedCode)
	case http.StatusForbidden:
		return nil, fmt.Errorf("request with body :%s : %s", string(body), errors.ForbiddenCode)
	default:
		return nil, fmt.Errorf(
			"failed to fetch token for request with body %s, status code %d",
			string(body),
			resp.StatusCode,
		)
	}
}

// getToken unmarshals the provided JSON-encoded body into the given token struct.
// If the "Token" field is empty but the "AccessToken" field is populated, it assigns "AccessToken" to "Token".
// It returns the updated token struct and any error encountered during unmarshalling.
func getToken(body []byte, t *token) (*token, error) {
	// Unmarshal the JSON body into the token struct
	if err := json.Unmarshal(body, t); err != nil {
		return nil, fmt.Errorf("failed to unmarshal token: %w", err)
	}

	// If Token is empty and AccessToken is provided, assign AccessToken to Token
	if t.Token == "" && t.AccessToken != "" {
		t.Token = t.AccessToken
	}

	return t, nil
}
