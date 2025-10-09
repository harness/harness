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
	"context"
	"fmt"
	"strings"

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git/command"
)

// Config set local git key and value configuration.
func (g *Git) Config(
	ctx context.Context,
	repoPath string,
	key string,
	value string,
) error {
	if repoPath == "" {
		return ErrRepositoryPathEmpty
	}
	if key == "" {
		return errors.InvalidArgument("key cannot be empty")
	}
	var outbuf, errbuf strings.Builder
	cmd := command.New("config",
		command.WithFlag("--local"),
		command.WithArg(key, value),
	)
	err := cmd.Run(ctx, command.WithDir(repoPath),
		command.WithStdout(&outbuf),
		command.WithStderr(&errbuf),
	)
	if err != nil {
		return fmt.Errorf("git config [%s -> <%s> ]: %w\n%s\n%s",
			key, value, err, outbuf.String(), errbuf.String())
	}
	return nil
}
