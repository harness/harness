//nolint:goheader
// Source: https://github.com/distribution/distribution
// Copyright 2014 https://github.com/distribution/distribution Authors
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

package docker

import (
	"context"
	"crypto/rand"
	"fmt"

	corestore "github.com/harness/gitness/app/store"
	"github.com/harness/gitness/registry/app/dist_temp/dcontext"
	"github.com/harness/gitness/registry/app/dist_temp/errcode"
	storagedriver "github.com/harness/gitness/registry/app/driver"
	"github.com/harness/gitness/registry/app/pkg"
	registrystorage "github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/gc"
	"github.com/harness/gitness/types"

	"github.com/opencontainers/go-digest"
	"github.com/rs/zerolog/log"
)

// randomSecretSize is the number of random bytes to generate if no secret
// was specified.
const randomSecretSize = 32

// App is a global registry application object. Shared resources can be placed
// on this object that will be accessible from all requests. Any writable
// fields should be protected.
type App struct {
	context.Context

	Config         *types.Config
	storageService *registrystorage.Service
	bucketService  BucketService
}

// NewApp takes a configuration and returns a configured app.
func NewApp(
	ctx context.Context, storageDeleter storagedriver.StorageDeleter,
	blobRepo store.BlobRepository, spaceStore corestore.SpaceStore,
	cfg *types.Config, storageService *registrystorage.Service,
	gcService gc.Service,
	bucketService BucketService,
) *App {
	app := &App{
		Context:        ctx,
		Config:         cfg,
		storageService: storageService,
		bucketService:  bucketService,
	}
	app.configureSecret(cfg) //nolint:contextcheck
	gcService.Start(ctx, spaceStore, blobRepo, storageDeleter, cfg)
	return app
}

// StorageService returns the storage service for this app.
func (app *App) StorageService() *registrystorage.Service {
	return app.storageService
}

func GetStorageService(cfg *types.Config, driver storagedriver.StorageDriver) *registrystorage.Service {
	options := registrystorage.GetRegistryOptions()
	if cfg.Registry.Storage.S3Storage.Delete {
		options = append(options, registrystorage.EnableDelete)
	}

	if cfg.Registry.Storage.S3Storage.Redirect {
		options = append(options, registrystorage.EnableRedirect)
	} else {
		log.Info().Msg("backend redirection disabled")
	}

	storageService, err := registrystorage.NewStorageService(driver, options...)
	if err != nil {
		panic("could not create storage service: " + err.Error())
	}
	return storageService
}

func LogError(errList errcode.Errors) {
	for _, e1 := range errList {
		log.Error().Err(e1).Msgf("error: %v", e1)
	}
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

// context constructs the context object for the application. This only be
// called once per request.
func (app *App) GetBlobsContext(c context.Context, info pkg.RegistryInfo, blobID interface{}) *Context {
	ctx := &Context{
		App:          app,
		Context:      c,
		UUID:         info.Reference,
		Digest:       digest.Digest(info.Digest),
		URLBuilder:   info.URLBuilder,
		OciBlobStore: nil,
	}

	if result := app.bucketService.GetBlobStore(c, info.RegIdentifier, info.RootIdentifier, blobID,
		digest.Digest(info.Digest).String()); result != nil {
		ctx.OciBlobStore = result.OciStore
	}
	if ctx.OciBlobStore == nil {
		ctx.OciBlobStore = app.storageService.OciBlobsStore(c, info.RegIdentifier, info.RootIdentifier)
	}
	return ctx
}
