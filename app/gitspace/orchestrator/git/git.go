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

package git

import (
	"context"

	"github.com/harness/gitness/app/gitspace/orchestrator/devcontainer"
	"github.com/harness/gitness/app/gitspace/scm"
	"github.com/harness/gitness/app/gitspace/types"
)

type Service interface {
	// Install ensures git is installed in the container.
	Install(ctx context.Context,
		exec *devcontainer.Exec,
		gitspaceLogger types.GitspaceLogger,
	) error

	// SetupCredentials sets the user's git credentials inside the container.
	SetupCredentials(
		ctx context.Context,
		exec *devcontainer.Exec,
		resolvedRepoDetails scm.ResolvedDetails,
		gitspaceLogger types.GitspaceLogger,
	) error

	// CloneCode clones the code and ensures devcontainer file is present.
	CloneCode(
		ctx context.Context,
		exec *devcontainer.Exec,
		resolvedRepoDetails scm.ResolvedDetails,
		defaultBaseImage string,
		gitspaceLogger types.GitspaceLogger,
	) error
}
