// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package gitea

import (
	"context"
	"fmt"
	"strings"

	"code.gitea.io/gitea/modules/git"
)

// Config set local git key and value configuration.
func (g Adapter) Config(ctx context.Context, repoPath, key, value string) error {
	var outbuf, errbuf strings.Builder
	if err := git.NewCommand(ctx, "config", "--local").AddArguments(key, value).
		Run(&git.RunOpts{
			Dir:    repoPath,
			Stdout: &outbuf,
			Stderr: &errbuf,
		}); err != nil {
		return fmt.Errorf("git config [%s -> <%s> ]: %w\n%s\n%s",
			key, value, err, outbuf.String(), errbuf.String())
	}
	return nil
}
