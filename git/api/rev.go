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

package api

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git/command"
)

func (g *Git) ResolveRev(ctx context.Context,
	repoPath string,
	rev string,
) (string, error) {
	cmd := command.New("rev-parse", command.WithArg(rev))
	output := &bytes.Buffer{}
	err := cmd.Run(ctx, command.WithDir(repoPath), command.WithStdout(output))
	if err != nil {
		if strings.Contains(err.Error(), "ambiguous argument") {
			return "", errors.InvalidArgument("could not resolve git revision: %s", rev)
		}
		return "", fmt.Errorf("failed to resolve git revision: %w", err)
	}
	commitSHA := strings.TrimSpace(output.String())
	return commitSHA, nil
}
