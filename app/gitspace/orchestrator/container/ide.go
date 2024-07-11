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

package container

import (
	"context"

	"github.com/harness/gitness/types/enum"
)

type IDE interface {
	// Setup is responsible for doing all the operations for setting up the IDE in the container e.g. installation,
	// copying settings and configurations, ensuring SSH server is running etc.
	Setup(ctx context.Context, containerParams *Devcontainer) ([]byte, error)

	// PortAndProtocol provides the port with protocol which will be used by this IDE.
	PortAndProtocol() string

	// Type provides the IDE type to which the service belongs.
	Type() enum.IDEType
}
