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

package api

import (
	usercontroller "github.com/harness/gitness/app/api/controller/user"
	"github.com/harness/gitness/app/auth/authn"
	"github.com/harness/gitness/app/auth/authz"
	corestore "github.com/harness/gitness/app/store"
	urlprovider "github.com/harness/gitness/app/url"
	ocihandler "github.com/harness/gitness/registry/app/api/handler/oci"
	"github.com/harness/gitness/registry/app/api/router"
	storagedriver "github.com/harness/gitness/registry/app/driver"
	"github.com/harness/gitness/registry/app/driver/factory"
	"github.com/harness/gitness/registry/app/driver/filesystem"
	"github.com/harness/gitness/registry/app/driver/s3-aws"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/docker"
	"github.com/harness/gitness/registry/app/store/database"
	"github.com/harness/gitness/registry/config"
	"github.com/harness/gitness/registry/gc"
	"github.com/harness/gitness/types"

	"github.com/google/wire"
	"github.com/rs/zerolog/log"
)

type RegistryApp struct {
	Config *types.Config

	AppRouter router.AppRouter
}

func BlobStorageProvider(c *types.Config) (storagedriver.StorageDriver, error) {
	var d storagedriver.StorageDriver
	var err error

	if c.Registry.Storage.StorageType == "filesystem" {
		filesystem.Register()
		d, err = factory.Create("filesystem", config.GetFilesystemParams(c))
		if err != nil {
			log.Fatal().Stack().Err(err).Msgf("")
			panic(err)
		}
	} else {
		s3.Register()
		d, err = factory.Create("s3aws", config.GetS3StorageParameters(c))
		if err != nil {
			log.Error().Stack().Err(err).Msg("failed to init s3 Blob storage ")
			panic(err)
		}
	}
	return d, err
}

func NewHandlerProvider(
	controller *docker.Controller, spaceStore corestore.SpaceStore,
	tokenStore corestore.TokenStore, userCtrl *usercontroller.Controller, authenticator authn.Authenticator,
	urlProvider urlprovider.Provider, authorizer authz.Authorizer, config *types.Config,
) *ocihandler.Handler {
	return ocihandler.NewHandler(
		controller,
		spaceStore,
		tokenStore,
		userCtrl,
		authenticator,
		urlProvider,
		authorizer,
		config.Registry.HTTP.RelativeURL,
	)
}

var WireSet = wire.NewSet(
	BlobStorageProvider,
	NewHandlerProvider,
	database.WireSet,
	pkg.WireSet,
	docker.WireSet,
	router.WireSet,
	gc.WireSet,
)

func Wire(_ *types.Config) (RegistryApp, error) {
	wire.Build(WireSet, wire.Struct(new(RegistryApp), "*"))
	return RegistryApp{}, nil
}
