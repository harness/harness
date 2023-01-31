// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package repo

import (
	"context"
	"io"
	"strings"

	"github.com/harness/gitness/gitrpc"
	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types/enum"
)

func (c *Controller) RawDiff(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	path string,
	w io.Writer,
) error {
	repo, err := c.repoStore.FindByRef(ctx, repoRef)
	if err != nil {
		return err
	}

	if err = apiauth.CheckRepo(ctx, c.authorizer, session, repo, enum.PermissionRepoView, false); err != nil {
		return err
	}

	info := parseDiffPath(path)

	return c.gitRPCClient.RawDiff(ctx, &gitrpc.DiffParams{
		ReadParams: CreateRPCReadParams(repo),
		BaseRef:    info.BaseRef,
		HeadRef:    info.HeadRef,
		MergeBase:  info.MergeBase,
	}, w)
}

type CompareInfo struct {
	BaseRef   string
	HeadRef   string
	MergeBase bool
}

func parseDiffPath(path string) CompareInfo {
	infos := strings.SplitN(path, "...", 2)
	if len(infos) != 2 {
		infos = strings.SplitN(path, "..", 2)
	}
	if len(infos) != 2 {
		return CompareInfo{
			HeadRef: path,
		}
	}
	return CompareInfo{
		BaseRef:   infos[0],
		HeadRef:   infos[1],
		MergeBase: strings.Contains(path, "..."),
	}
}
