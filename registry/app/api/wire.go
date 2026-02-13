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
	"context"

	usercontroller "github.com/harness/gitness/app/api/controller/user"
	"github.com/harness/gitness/app/auth/authn"
	"github.com/harness/gitness/app/auth/authz"
	"github.com/harness/gitness/app/services/refcache"
	corestore "github.com/harness/gitness/app/store"
	urlprovider "github.com/harness/gitness/app/url"
	cargo2 "github.com/harness/gitness/registry/app/api/controller/pkg/cargo"
	generic3 "github.com/harness/gitness/registry/app/api/controller/pkg/generic"
	gopackage2 "github.com/harness/gitness/registry/app/api/controller/pkg/gopackage"
	"github.com/harness/gitness/registry/app/api/controller/pkg/huggingface"
	"github.com/harness/gitness/registry/app/api/controller/pkg/npm"
	nuget2 "github.com/harness/gitness/registry/app/api/controller/pkg/nuget"
	python2 "github.com/harness/gitness/registry/app/api/controller/pkg/python"
	rpm2 "github.com/harness/gitness/registry/app/api/controller/pkg/rpm"
	"github.com/harness/gitness/registry/app/api/handler/cargo"
	"github.com/harness/gitness/registry/app/api/handler/generic"
	"github.com/harness/gitness/registry/app/api/handler/gopackage"
	hf2 "github.com/harness/gitness/registry/app/api/handler/huggingface"
	mavenhandler "github.com/harness/gitness/registry/app/api/handler/maven"
	npm2 "github.com/harness/gitness/registry/app/api/handler/npm"
	nugethandler "github.com/harness/gitness/registry/app/api/handler/nuget"
	ocihandler "github.com/harness/gitness/registry/app/api/handler/oci"
	"github.com/harness/gitness/registry/app/api/handler/packages"
	pypi2 "github.com/harness/gitness/registry/app/api/handler/python"
	"github.com/harness/gitness/registry/app/api/handler/rpm"
	"github.com/harness/gitness/registry/app/api/interfaces"
	"github.com/harness/gitness/registry/app/api/router"
	storagedriver "github.com/harness/gitness/registry/app/driver"
	"github.com/harness/gitness/registry/app/driver/factory"
	"github.com/harness/gitness/registry/app/driver/filesystem"
	"github.com/harness/gitness/registry/app/driver/s3-aws"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/base"
	cargoregistry "github.com/harness/gitness/registry/app/pkg/cargo"
	"github.com/harness/gitness/registry/app/pkg/docker"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	generic2 "github.com/harness/gitness/registry/app/pkg/generic"
	gopackageregistry "github.com/harness/gitness/registry/app/pkg/gopackage"
	hf3 "github.com/harness/gitness/registry/app/pkg/huggingface"
	"github.com/harness/gitness/registry/app/pkg/maven"
	npm22 "github.com/harness/gitness/registry/app/pkg/npm"
	"github.com/harness/gitness/registry/app/pkg/nuget"
	"github.com/harness/gitness/registry/app/pkg/python"
	"github.com/harness/gitness/registry/app/pkg/quarantine"
	rpmregistry "github.com/harness/gitness/registry/app/pkg/rpm"
	publicaccess2 "github.com/harness/gitness/registry/app/services/publicaccess"
	refcache2 "github.com/harness/gitness/registry/app/services/refcache"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/app/store/cache"
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

func DefaultStorageProvider(ctx context.Context, c *types.Config) (storagedriver.StorageDriver, error) {
	var d storagedriver.StorageDriver
	var err error

	if c.Registry.Storage.StorageType == "filesystem" {
		filesystem.Register(ctx)
		d, err = factory.Create(ctx, "filesystem", config.GetFilesystemParams(c))
		if err != nil {
			log.Fatal().Stack().Err(err).Msgf("")
			panic(err)
		}
	} else {
		s3.Register(ctx)
		d, err = factory.Create(ctx, "s3aws", config.GetS3StorageParameters(c))
		if err != nil {
			log.Error().Stack().Err(err).Msg("failed to init s3 Blob storage ")
			panic(err)
		}
	}
	return d, err
}

func NewHandlerProvider(
	controller *docker.Controller,
	spaceFinder refcache.SpaceFinder,
	spaceStore corestore.SpaceStore,
	tokenStore corestore.TokenStore,
	userCtrl *usercontroller.Controller,
	authenticator authn.Authenticator,
	urlProvider urlprovider.Provider,
	authorizer authz.Authorizer,
	config *types.Config,
	registryFinder refcache2.RegistryFinder,
	publicAccessService publicaccess2.CacheService,
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
		registryFinder,
		publicAccessService,
		config.Auth.AnonymousUserSecret,
	)
}

func NewMavenHandlerProvider(
	controller *maven.Controller, spaceStore corestore.SpaceStore,
	tokenStore corestore.TokenStore, userCtrl *usercontroller.Controller, authenticator authn.Authenticator,
	authorizer authz.Authorizer, spaceFinder refcache.SpaceFinder, registryFinder refcache2.RegistryFinder,
	publicAccessService publicaccess2.CacheService,
) *mavenhandler.Handler {
	return mavenhandler.NewHandler(
		controller,
		spaceStore,
		tokenStore,
		userCtrl,
		authenticator,
		authorizer,
		spaceFinder,
		registryFinder,
		publicAccessService,
	)
}

func NewPackageHandlerProvider(
	registryDao store.RegistryRepository, downloadStatDao store.DownloadStatRepository,
	bandwidthStatDao store.BandwidthStatRepository,
	spaceStore corestore.SpaceStore, tokenStore corestore.TokenStore,
	userCtrl *usercontroller.Controller, authenticator authn.Authenticator,
	urlProvider urlprovider.Provider, authorizer authz.Authorizer, spaceFinder refcache.SpaceFinder,
	regFinder refcache2.RegistryFinder,
	fileManager filemanager.FileManager, quarantineFinder quarantine.Finder,
	packageWrapper interfaces.PackageWrapper,
) packages.Handler {
	return packages.NewHandler(
		registryDao,
		downloadStatDao,
		bandwidthStatDao,
		spaceStore,
		tokenStore,
		userCtrl,
		authenticator,
		urlProvider,
		authorizer,
		spaceFinder,
		regFinder,
		fileManager,
		quarantineFinder,
		packageWrapper,
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

func NewNPMHandlerProvider(
	controller npm.Controller,
	packageHandler packages.Handler,
) npm2.Handler {
	return npm2.NewHandler(controller, packageHandler)
}

func NewRpmHandlerProvider(
	controller rpm2.Controller,
	packageHandler packages.Handler,
) rpm.Handler {
	return rpm.NewHandler(controller, packageHandler)
}

func NewGenericHandlerProvider(
	spaceStore corestore.SpaceStore, controller *generic3.Controller, tokenStore corestore.TokenStore,
	userCtrl *usercontroller.Controller, authenticator authn.Authenticator, urlProvider urlprovider.Provider,
	authorizer authz.Authorizer, packageHandler packages.Handler, spaceFinder refcache.SpaceFinder,
	registryFinder refcache2.RegistryFinder,
) *generic.Handler {
	return generic.NewGenericArtifactHandler(
		spaceStore,
		controller,
		tokenStore,
		userCtrl,
		authenticator,
		urlProvider,
		authorizer,
		packageHandler,
		spaceFinder,
		registryFinder,
	)
}

func NewCargoHandlerProvider(
	controller cargo2.Controller,
	packageHandler packages.Handler,
) cargo.Handler {
	return cargo.NewHandler(controller, packageHandler)
}

func NewGoPackageHandlerProvider(
	controller gopackage2.Controller,
	packageHandler packages.Handler,
) gopackage.Handler {
	return gopackage.NewHandler(controller, packageHandler)
}

var WireSet = wire.NewSet(
	DefaultStorageProvider,
	NewHandlerProvider,
	NewMavenHandlerProvider,
	NewGenericHandlerProvider,
	NewPackageHandlerProvider,
	NewPythonHandlerProvider,
	NewNugetHandlerProvider,
	NewNPMHandlerProvider,
	NewRpmHandlerProvider,
	NewCargoHandlerProvider,
	NewGoPackageHandlerProvider,
	database.WireSet,
	cache.WireSet,
	refcache2.WireSet,
	pkg.WireSet,
	docker.OpenSourceWireSet,
	filemanager.WireSet,
	quarantine.WireSet,
	maven.WireSet,
	nuget.WireSet,
	python.WireSet,
	generic2.WireSet,
	npm22.WireSet,
	router.WireSet,
	gc.WireSet,
	generic3.WireSet,
	python2.ControllerSet,
	nuget2.ControllerSet,
	npm.ControllerSet,
	base.WireSet,
	rpm2.ControllerSet,
	rpmregistry.WireSet,
	cargo2.ControllerSet,
	cargoregistry.WireSet,
	gopackage2.ControllerSet,
	gopackageregistry.WireSet,
	huggingface.WireSet,
	hf2.WireSet,
	hf3.WireSet,
	publicaccess2.WireSet,
)

func Wire(_ *types.Config) (RegistryApp, error) {
	wire.Build(WireSet, wire.Struct(new(RegistryApp), "*"))
	return RegistryApp{}, nil
}
