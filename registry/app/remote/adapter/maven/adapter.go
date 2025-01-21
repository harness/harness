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

package maven

import (
	"context"
	"net/http"

	store2 "github.com/harness/gitness/app/store"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	commonhttp "github.com/harness/gitness/registry/app/common/http"
	adp "github.com/harness/gitness/registry/app/remote/adapter"
	"github.com/harness/gitness/registry/app/remote/adapter/native"
	"github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/secret"

	"github.com/rs/zerolog/log"
)

const (
	registryURL = "https://repo1.maven.org/maven2/"
)

func init() {
	adapterType := string(artifact.UpstreamConfigSourceMavenCentral)
	if err := adp.RegisterFactory(adapterType, new(factory)); err != nil {
		log.Error().Stack().Err(err).Msgf("Register adapter factory for %s", adapterType)
		return
	}
}

func newAdapter(
	ctx context.Context, spacePathStore store2.SpacePathStore, service secret.Service, registry types.UpstreamProxy,
) (adp.Adapter, error) {
	client, err := NewClient(registry)
	if err != nil {
		return nil, err
	}

	return &adapter{
		client:  client,
		Adapter: native.NewAdapter(ctx, spacePathStore, service, registry),
	}, nil
}

type factory struct {
}

// Create ...
func (f *factory) Create(
	ctx context.Context, spacePathStore store2.SpacePathStore, record types.UpstreamProxy, service secret.Service,
) (adp.Adapter, error) {
	return newAdapter(ctx, spacePathStore, service, record)
}

var (
	_ adp.Adapter          = (*adapter)(nil)
	_ adp.ArtifactRegistry = (*adapter)(nil)
)

type adapter struct {
	*native.Adapter
	client *Client
}

// Ensure '*adapter' implements interface 'Adapter'.
var _ adp.Adapter = (*adapter)(nil)

// Client is a client to interact with DockerHub.
type Client struct {
	client *http.Client
	host   string
}

// NewClient creates a new DockerHub client.
func NewClient(_ types.UpstreamProxy) (*Client, error) {
	client := &Client{
		host: registryURL,
		client: &http.Client{
			Transport: commonhttp.GetHTTPTransport(commonhttp.WithInsecure(true)),
		},
	}

	return client, nil
}
