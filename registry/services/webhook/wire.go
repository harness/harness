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

package webhook

import (
	"context"
	"encoding/gob"

	gitnesswebhook "github.com/harness/gitness/app/services/webhook"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/encrypt"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/registry/app/events/artifact"
	registrystore "github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/secret"
	"github.com/harness/gitness/store/database/dbtx"

	"github.com/google/wire"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	ProvideService,
)

func ProvideService(
	ctx context.Context,
	config gitnesswebhook.Config,
	tx dbtx.Transactor,
	artifactsReaderFactory *events.ReaderFactory[*artifact.Reader],
	webhookStore registrystore.WebhooksRepository,
	webhookExecutionStore registrystore.WebhooksExecutionRepository,
	spaceStore store.SpaceStore,
	urlProvider url.Provider,
	principalStore store.PrincipalStore,
	webhookURLProvider gitnesswebhook.URLProvider,
	spacePathStore store.SpacePathStore,
	secretService secret.Service,
	registryRepository registrystore.RegistryRepository,
	encrypter encrypt.Encrypter,
) (*Service, error) {
	gob.Register(&artifact.DockerArtifact{})
	gob.Register(&artifact.HelmArtifact{})
	return NewService(
		ctx,
		config,
		tx,
		artifactsReaderFactory,
		webhookStore,
		webhookExecutionStore,
		spaceStore,
		urlProvider,
		principalStore,
		webhookURLProvider,
		spacePathStore,
		secretService,
		registryRepository,
		encrypter,
	)
}
