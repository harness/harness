// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package gitea

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"code.gitea.io/gitea/modules/git"
)

func (g Adapter) RawDiff(ctx context.Context, repoPath, left, right string, w io.Writer, customArgs ...string) error {
	args := []string{
		"diff",
		"--src-prefix=\\a/",
		"--dst-prefix=\\b/",
		"-M",
	}
	args = append(args, customArgs...)
	args = append(args, left, right)

	cmd := git.NewCommand(ctx, args...)
	cmd.SetDescription(fmt.Sprintf("GetDiffRange [repo_path: %s]", repoPath))
	errbuf := bytes.Buffer{}
	if err := cmd.Run(&git.RunOpts{
		Dir:    repoPath,
		Stderr: &errbuf,
		Stdout: w,
	}); err != nil {
		return fmt.Errorf("git diff [%s base:%s head:%s]: %s", repoPath, left, right, errbuf.String())
	}
	return nil
}
