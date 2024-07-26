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

package ide

import (
	"context"

	"github.com/harness/gitness/app/gitspace/orchestrator/devcontainer"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type IDE interface {
	// Setup is responsible for doing all the operations for setting up the IDE in the container e.g. installation,
	// copying settings and configurations.
	Setup(
		ctx context.Context,
		devcontainer *devcontainer.Exec,
		gitspaceInstance *types.GitspaceInstance,
	) ([]byte, error)

	// Run runs the IDE and supporting services.
	Run(ctx context.Context, devcontainer *devcontainer.Exec) ([]byte, error)

	// Port provides the port which will be used by this IDE.
	Port() int

	// Type provides the IDE type to which the service belongs.
	Type() enum.IDEType
}
