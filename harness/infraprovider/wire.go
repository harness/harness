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

package infraprovider

import (
	events "github.com/harness/gitness/app/events/gitspaceinfra"

	"github.com/google/wire"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	ProvideDockerProvider,
	ProvideFactory,
	ProvideDockerClientFactory,
)

func ProvideDockerProvider(
	config *DockerConfig,
	dockerClientFactory *DockerClientFactory,
	eventReporter *events.Reporter,
) *DockerProvider {
	return NewDockerProvider(config, dockerClientFactory, eventReporter)
}

func ProvideFactory(dockerProvider *DockerProvider) Factory {
	return NewFactory(dockerProvider)
}

func ProvideDockerClientFactory(config *DockerConfig) *DockerClientFactory {
	return NewDockerClientFactory(config)
}
