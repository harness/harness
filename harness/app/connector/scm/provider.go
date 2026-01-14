// Copyright 2023 Harness, Inc.
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

package scm

import (
	"context"
	"fmt"
	"net/http"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/drone/go-scm/scm"
	"github.com/drone/go-scm/scm/driver/github"
	"github.com/drone/go-scm/scm/transport/oauth2"
)

// getSCMProvider returns an SCM client given a connector.
// The SCM client can be used as a common layer for interfacing with any SCM.
func getSCMProvider(
	ctx context.Context,
	connector *types.Connector,
	secretStore store.SecretStore,
) (*scm.Client, error) {
	var client *scm.Client
	var err error
	var transport http.RoundTripper

	switch x := connector.Type; x {
	case enum.ConnectorTypeGithub:
		if connector.Github == nil {
			return nil, fmt.Errorf("github connector is nil")
		}
		if connector.Github.APIURL == "" {
			client = github.NewDefault()
		} else {
			client, err = github.New(connector.Github.APIURL)
			if err != nil {
				return nil, err
			}
		}
		if connector.Github.Auth == nil {
			return nil, fmt.Errorf("github auth needs to be provided")
		}
		if connector.Github.Auth.AuthType == enum.ConnectorAuthTypeBearer {
			creds := connector.Github.Auth.Bearer
			pass, err := resolveSecret(ctx, connector.SpaceID, creds.Token, secretStore)
			if err != nil {
				return nil, err
			}
			transport = oauthTransport(pass, oauth2.SchemeBearer)
		} else {
			return nil, fmt.Errorf("unsupported auth type for github connector: %s", connector.Github.Auth.AuthType)
		}
	default:
		return nil, fmt.Errorf("unsupported scm provider type: %s", x)
	}

	// override default transport if available
	if transport != nil {
		client.Client = &http.Client{Transport: transport}
	}

	return client, nil
}

func oauthTransport(token string, scheme string) http.RoundTripper {
	if token == "" {
		return nil
	}
	return &oauth2.Transport{
		Base:   defaultTransport(),
		Scheme: scheme,
		Source: oauth2.StaticTokenSource(&scm.Token{Token: token}),
	}
}

// defaultTransport provides a default http.Transport.
// This can be extended when needed for things like more advanced TLS config, proxies, etc.
func defaultTransport() http.RoundTripper {
	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,
	}
}

// resolveSecret looks into the secret store to find the value of a secret.
func resolveSecret(
	ctx context.Context,
	spaceID int64,
	ref types.SecretRef,
	secretStore store.SecretStore,
) (string, error) {
	// the secret should be in the same space as the connector
	s, err := secretStore.FindByIdentifier(ctx, spaceID, ref.Identifier)
	if err != nil {
		return "", fmt.Errorf("could not find secret from store: %w", err)
	}
	return s.Data, nil
}
