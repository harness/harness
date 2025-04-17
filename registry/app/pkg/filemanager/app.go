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

package filemanager

import (
	"context"
	"crypto/rand"
	"fmt"

	"github.com/harness/gitness/registry/app/dist_temp/dcontext"
	registrystorage "github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/types"

	"github.com/rs/zerolog/log"
)

const randomSecretSize = 32

type App struct {
	context.Context

	Config         *types.Config
	storageService *registrystorage.Service
}

// NewApp takes a configuration and returns a configured app.
func NewApp(
	ctx context.Context,
	cfg *types.Config, storageService *registrystorage.Service,
) *App {
	app := &App{
		Context:        ctx,
		Config:         cfg,
		storageService: storageService,
	}
	app.configureSecret(cfg) //nolint:contextcheck
	return app
}

// configureSecret creates a random secret if a secret wasn't included in the
// configuration.
func (app *App) configureSecret(configuration *types.Config) {
	if configuration.Registry.HTTP.Secret == "" {
		var secretBytes [randomSecretSize]byte
		if _, err := rand.Read(secretBytes[:]); err != nil {
			panic(fmt.Sprintf("could not generate random bytes for HTTP secret: %v", err))
		}
		configuration.Registry.HTTP.Secret = string(secretBytes[:])
		dcontext.GetLogger(app, log.Warn()).
			Msg(
				"No HTTP secret provided - generated random secret. This may cause problems with uploads if" +
					" multiple registries are behind a load-balancer. To provide a shared secret," +
					" set the GITNESS_REGISTRY_HTTP_SECRET environment variable.",
			)
	}
}

// GetBlobsContext context constructs the context object for the application. This only be
// called once per request.
func (app *App) GetBlobsContext(c context.Context, rootIdentifier string) *Context {
	context := &Context{
		App:     app,
		Context: c,
	}
	blobStore := app.storageService.GenericBlobsStore(rootIdentifier)
	context.genericBlobStore = blobStore

	return context
}
