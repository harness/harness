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

package handlers

import (
	"context"
	"fmt"
	"strings"

	"github.com/harness/gitness/app/services/importer"
	"github.com/harness/gitness/app/services/keywordsearch"
	"github.com/harness/gitness/app/services/linkedsync"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/store"
	gitnessurl "github.com/harness/gitness/app/url"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/types"

	"github.com/rs/zerolog/log"
)

const (
	branchRefPrefix = "refs/heads/"
	tagRefPrefix    = "refs/tags/"
)

// PushHandler mirrors a push webhook's branch/tag tip into the linked repo's
// mirror. Unlike PullRequestHandler, a push carries no PR-namespace rename,
// so the sync logic needs no per-provider RefSyncSpec and the handler stays
// fully provider-agnostic.
type PushHandler struct {
	repoStore        store.RepoStore
	indexer          keywordsearch.Indexer
	gitClient        git.Interface
	repoFinder       refcache.RepoFinder
	urlProvider      gitnessurl.Provider
	connectorService importer.ConnectorService
}

func NewPushHandler(
	repoStore store.RepoStore,
	indexer keywordsearch.Indexer,
	gitClient git.Interface,
	repoFinder refcache.RepoFinder,
	urlProvider gitnessurl.Provider,
	connectorService importer.ConnectorService,
) *PushHandler {
	return &PushHandler{
		repoStore:        repoStore,
		indexer:          indexer,
		gitClient:        gitClient,
		repoFinder:       repoFinder,
		urlProvider:      urlProvider,
		connectorService: connectorService,
	}
}

// isNamedBranchOrTagRef reports whether ref is a branch or tag ref carrying a
// non-empty name. The name check is not cosmetic: a bare prefix such as
// "refs/heads/" refers to nothing, so SyncRefs hard-fails on it and the
// resulting error would poison-pill the consumer into endless retry.
// PullRequestHandler rejects the equivalent payload for the same reason.
func isNamedBranchOrTagRef(ref string) bool {
	for _, prefix := range []string{branchRefPrefix, tagRefPrefix} {
		if strings.HasPrefix(ref, prefix) && len(ref) > len(prefix) {
			return true
		}
	}
	return false
}

// Handle mirrors the pushed ref, whether or not a PR ever pointed at it, then
// (best-effort, like linked_sync.go) re-indexes the repo if the default
// branch moved.
func (h *PushHandler) Handle(
	ctx context.Context,
	_ *linkedsync.Event,
	payload linkedsync.PushPayload,
	linkedRepo *types.LinkedRepo,
) error {
	if !isNamedBranchOrTagRef(payload.Ref) {
		log.Ctx(ctx).Debug().Str("ref", payload.Ref).
			Msg("linkedsync: push ref is not a named branch or tag; ack-skipping")
		return nil
	}

	afterSHA, err := sha.NewOrEmpty(payload.After)
	if err != nil {
		// A malformed sha is a producer bug, not a transient failure: retrying
		// can never make it parse, so ack-skip instead of poison-pilling.
		log.Ctx(ctx).Warn().Err(err).Str("ref", payload.Ref).Str("after", payload.After).
			Msg("linkedsync: push payload has malformed after-sha; ack-skipping")
		return nil //nolint:nilerr
	}
	if afterSHA.IsEmpty() || afterSHA.IsNil() {
		// Branch/tag deletion: the ref is gone upstream, so SyncRefs would
		// hard-fail trying to fetch it. Mirroring deletions is out of scope.
		log.Ctx(ctx).Debug().Str("ref", payload.Ref).
			Msg("linkedsync: push after-sha is empty/nil (deletion); ack-skipping")
		return nil
	}

	if err := RunSyncRefs(
		ctx, h.gitClient, h.repoFinder, h.urlProvider, h.connectorService,
		linkedRepo, []RefSyncEntry{{RemoteRef: payload.Ref}},
	); err != nil {
		return fmt.Errorf("sync refs: %w", err)
	}

	repo, err := h.repoStore.Find(ctx, linkedRepo.RepoID)
	if err != nil {
		return fmt.Errorf("find repo %d: %w", linkedRepo.RepoID, err)
	}

	if payload.Ref != branchRefPrefix+repo.DefaultBranch {
		return nil
	}

	if err := h.indexer.Index(ctx, repo); err != nil {
		log.Ctx(ctx).Warn().Err(err).Int64("repo_id", repo.ID).
			Msg("linkedsync: failed to index repository after default branch push")
	}
	return nil
}
