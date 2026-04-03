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

package githook

import (
	"context"
	"fmt"

	"github.com/harness/gitness/git"
	"github.com/harness/gitness/types/enum"
)

// CreateWriteParamsForOperation creates git write parameters together with the
// githook environment payload for the provided operation type.
func CreateWriteParamsForOperation(
	ctx context.Context,
	apiBaseURL string,
	actor git.Identity,
	repoID int64,
	repoUID string,
	principalID int64,
	disabled bool,
	operationType enum.GitOpType,
) (git.WriteParams, error) {
	envVars, err := GenerateEnvironmentVariablesForOperation(
		ctx,
		apiBaseURL,
		repoID,
		principalID,
		disabled,
		operationType,
	)
	if err != nil {
		return git.WriteParams{}, fmt.Errorf("failed to generate git hook environment variables: %w", err)
	}

	return git.WriteParams{
		Actor:   actor,
		RepoUID: repoUID,
		EnvVars: envVars,
	}, nil
}
