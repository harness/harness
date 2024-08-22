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

package dockerhub

import (
	"context"

	store2 "github.com/harness/gitness/app/store"
	"github.com/harness/gitness/encrypt"
	adp "github.com/harness/gitness/registry/app/remote/adapter"
	"github.com/harness/gitness/registry/app/remote/adapter/native"
	"github.com/harness/gitness/registry/types"

	"github.com/rs/zerolog/log"
)

func init() {
	adapterType := "docker"
	if err := adp.RegisterFactory(adapterType, new(factory)); err != nil {
		log.Error().Stack().Err(err).Msgf("Register adapter factory for %s", adapterType)
		return
	}
	log.Info().Msgf("Factory for adapter %s registered", adapterType)
}

func newAdapter(
	ctx context.Context,
	secretStore store2.SecretStore,
	encrypter encrypt.Encrypter,
	registry types.UpstreamProxy,
) (adp.Adapter, error) {
	client, err := NewClient(registry)
	if err != nil {
		return nil, err
	}

	// TODO: get Upstream Credentials
	return &adapter{
		client:  client,
		Adapter: native.NewAdapter(ctx, secretStore, encrypter, registry),
	}, nil
}

type factory struct {
}

// Create ...
func (f *factory) Create(
	ctx context.Context,
	secretStore store2.SecretStore,
	encrypter encrypt.Encrypter,
	record types.UpstreamProxy,
) (adp.Adapter, error) {
	return newAdapter(ctx, secretStore, encrypter, record)
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
