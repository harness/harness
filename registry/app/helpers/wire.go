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

package helpers

import (
	"github.com/harness/gitness/registry/app/api/interfaces"
	registryevents "github.com/harness/gitness/registry/app/events/artifact"
	registrypostprocessingevents "github.com/harness/gitness/registry/app/events/asyncprocessing"
	"github.com/harness/gitness/registry/app/factory"
	"github.com/harness/gitness/registry/app/helpers/pkg"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/store/database/dbtx"

	"github.com/google/wire"
)

func ProvidePackageWrapperProvider(registryHelper interfaces.RegistryHelper) interfaces.PackageWrapper {
	// create package factory
	packageFactory := factory.NewPackageFactory()
	packageFactory.Register(pkg.NewCargoPackageType(registryHelper))
	packageFactory.Register(pkg.NewDockerPackageType(registryHelper))
	packageFactory.Register(pkg.NewHelmPackageType(registryHelper))
	packageFactory.Register(pkg.NewGenericPackageType(registryHelper))
	packageFactory.Register(pkg.NewMavenPackageType(registryHelper))
	packageFactory.Register(pkg.NewPythonPackageType(registryHelper))
	packageFactory.Register(pkg.NewNugetPackageType(registryHelper))
	packageFactory.Register(pkg.NewRPMPackageType(registryHelper))
	packageFactory.Register(pkg.NewNPMPackageType(registryHelper))
	packageFactory.Register(pkg.NewGoPackageType(registryHelper))
	packageFactory.Register(pkg.NewHuggingFacePackageType(registryHelper))

	return NewPackageWrapper(packageFactory)
}

func ProvideRegistryHelper(
	artifactStore store.ArtifactRepository,
	fileManager filemanager.FileManager,
	imageStore store.ImageRepository,
	artifactEventReporter *registryevents.Reporter,
	postProcessingReporter *registrypostprocessingevents.Reporter,
	tx dbtx.Transactor,
) interfaces.RegistryHelper {
	return NewRegistryHelper(
		artifactStore,
		fileManager,
		imageStore,
		artifactEventReporter,
		postProcessingReporter,
		tx,
	)
}

var WireSet = wire.NewSet(
	ProvidePackageWrapperProvider,
	ProvideRegistryHelper,
)
