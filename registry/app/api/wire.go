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
	"github.com/harness/gitness/app/services/refcache"
	corestore "github.com/harness/gitness/app/store"
	urlprovider "github.com/harness/gitness/app/url"
	nuget2 "github.com/harness/gitness/registry/app/api/controller/pkg/nuget"
	python2 "github.com/harness/gitness/registry/app/api/controller/pkg/python"
	"github.com/harness/gitness/registry/app/api/handler/generic"
	mavenhandler "github.com/harness/gitness/registry/app/api/handler/maven"
	nugethandler "github.com/harness/gitness/registry/app/api/handler/nuget"
	ocihandler "github.com/harness/gitness/registry/app/api/handler/oci"
	"github.com/harness/gitness/registry/app/api/handler/packages"
	pypi2 "github.com/harness/gitness/registry/app/api/handler/python"
	"github.com/harness/gitness/registry/app/api/router"
	storagedriver "github.com/harness/gitness/registry/app/driver"
	"github.com/harness/gitness/registry/app/driver/factory"
	"github.com/harness/gitness/registry/app/driver/filesystem"
	"github.com/harness/gitness/registry/app/driver/s3-aws"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/base"
	"github.com/harness/gitness/registry/app/pkg/docker"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	generic2 "github.com/harness/gitness/registry/app/pkg/generic"
	"github.com/harness/gitness/registry/app/pkg/maven"
	"github.com/harness/gitness/registry/app/pkg/nuget"
	"github.com/harness/gitness/registry/app/pkg/python"
	"github.com/harness/gitness/registry/app/store"
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
	controller *docker.Controller, spaceFinder refcache.SpaceFinder, spaceStore corestore.SpaceStore,
	tokenStore corestore.TokenStore, userCtrl *usercontroller.Controller, authenticator authn.Authenticator,
	urlProvider urlprovider.Provider, authorizer authz.Authorizer, config *types.Config,
) *ocihandler.Handler {
	return ocihandler.NewHandler(
		controller,
		spaceFinder,
		spaceStore,
		tokenStore,
		userCtrl,
		authenticator,
		urlProvider,
		authorizer,
		config.Registry.HTTP.RelativeURL,
	)
}

func NewMavenHandlerProvider(
	controller *maven.Controller, spaceStore corestore.SpaceStore,
	tokenStore corestore.TokenStore, userCtrl *usercontroller.Controller, authenticator authn.Authenticator,
	authorizer authz.Authorizer,
) *mavenhandler.Handler {
	return mavenhandler.NewHandler(
		controller,
		spaceStore,
		tokenStore,
		userCtrl,
		authenticator,
		authorizer,
	)
}

func NewPackageHandlerProvider(
	registryDao store.RegistryRepository, spaceStore corestore.SpaceStore, tokenStore corestore.TokenStore,
	userCtrl *usercontroller.Controller, authenticator authn.Authenticator,
	urlProvider urlprovider.Provider, authorizer authz.Authorizer,
) packages.Handler {
	return packages.NewHandler(
		registryDao,
		spaceStore,
		tokenStore,
		userCtrl,
		authenticator,
		urlProvider,
		authorizer,
	)
}

func NewPythonHandlerProvider(
	controller python2.Controller,
	packageHandler packages.Handler,
) pypi2.Handler {
	return pypi2.NewHandler(controller, packageHandler)
}

func NewNugetHandlerProvider(
	controller nuget2.Controller,
	packageHandler packages.Handler,
) nugethandler.Handler {
	return nugethandler.NewHandler(controller, packageHandler)
}

func NewGenericHandlerProvider(
	spaceStore corestore.SpaceStore, controller *generic2.Controller, tokenStore corestore.TokenStore,
	userCtrl *usercontroller.Controller, authenticator authn.Authenticator, urlProvider urlprovider.Provider,
	authorizer authz.Authorizer,
) *generic.Handler {
	return generic.NewGenericArtifactHandler(
		spaceStore,
		controller,
		tokenStore,
		userCtrl,
		authenticator,
		urlProvider,
		authorizer,
	)
}

var WireSet = wire.NewSet(
	BlobStorageProvider,
	NewHandlerProvider,
	NewMavenHandlerProvider,
	NewGenericHandlerProvider,
	NewPackageHandlerProvider,
	NewPythonHandlerProvider,
	NewNugetHandlerProvider,
	database.WireSet,
	pkg.WireSet,
	docker.WireSet,
	filemanager.WireSet,
	maven.WireSet,
	nuget.WireSet,
	python.WireSet,
	router.WireSet,
	gc.WireSet,
	generic2.WireSet,
	python2.ControllerSet,
	nuget2.ControllerSet,
	base.WireSet,
)

func Wire(_ *types.Config) (RegistryApp, error) {
	wire.Build(WireSet, wire.Struct(new(RegistryApp), "*"))
	return RegistryApp{}, nil
}
