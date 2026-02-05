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

package dotrange

import (
	"context"
	"fmt"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/controller"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/auth/authz"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git"
	gitenum "github.com/harness/gitness/git/enum"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type Service struct {
	git         git.Interface
	repoFinder  refcache.RepoFinder
	urlProvider url.Provider
	authorizer  authz.Authorizer
}

func (s *Service) FetchUpstreamBranch(
	ctx context.Context,
	session *auth.Session,
	repoForkCore *types.RepositoryCore,
	branchName string,
) (sha.SHA, *types.RepositoryCore, error) {
	return s.fetchUpstreamObjects(
		ctx,
		session,
		repoForkCore,
		func(readParams git.ReadParams) (sha.SHA, error) {
			result, err := s.git.GetRef(ctx, git.GetRefParams{
				ReadParams: readParams,
				Name:       branchName,
				Type:       gitenum.RefTypeBranch,
			})
			if err != nil {
				return sha.SHA{}, fmt.Errorf("failed to fetch branch %s: %w", branchName, err)
			}

			return result.SHA, nil
		})
}

func (s *Service) FetchCommitDivergenceObjectsFromUpstream(
	ctx context.Context,
	session *auth.Session,
	repo *types.RepositoryCore,
	div *git.CommitDivergenceRequest,
) error {
	dot, err := Make(div.To, div.From, true)
	if err != nil {
		return fmt.Errorf("failed to make dot range: %w", err)
	}

	err = s.FetchDotRangeObjectsFromUpstream(ctx, session, repo, &dot)
	if err != nil {
		return fmt.Errorf("failed to fetch dot range objects: %w", err)
	}

	div.To = dot.BaseRef
	div.From = dot.HeadRef

	return nil
}

func (s *Service) FetchDotRangeObjectsFromUpstream(
	ctx context.Context,
	session *auth.Session,
	repoForkCore *types.RepositoryCore,
	dotRange *DotRange,
) error {
	if dotRange.BaseUpstream {
		refSHA, _, err := s.fetchUpstreamRevision(ctx, session, repoForkCore, dotRange.BaseRef)
		if err != nil {
			return fmt.Errorf("failed to fetch upstream objects: %w", err)
		}

		dotRange.BaseUpstream = false
		dotRange.BaseRef = refSHA.String()
	}

	if dotRange.HeadUpstream {
		refSHA, _, err := s.fetchUpstreamRevision(ctx, session, repoForkCore, dotRange.HeadRef)
		if err != nil {
			return fmt.Errorf("failed to fetch upstream objects: %w", err)
		}

		dotRange.HeadUpstream = false
		dotRange.HeadRef = refSHA.String()
	}

	return nil
}

func (s *Service) fetchUpstreamRevision(
	ctx context.Context,
	session *auth.Session,
	repoForkCore *types.RepositoryCore,
	revision string,
) (sha.SHA, *types.RepositoryCore, error) {
	return s.fetchUpstreamObjects(
		ctx,
		session,
		repoForkCore,
		func(readParams git.ReadParams) (sha.SHA, error) {
			result, err := s.git.ResolveRevision(ctx, git.ResolveRevisionParams{
				ReadParams: readParams,
				Revision:   revision,
			})
			if err != nil {
				return sha.SHA{}, fmt.Errorf("failed to resolve revision %s: %w", revision, err)
			}

			return result.SHA, nil
		})
}

func (s *Service) fetchUpstreamObjects(
	ctx context.Context,
	session *auth.Session,
	repoForkCore *types.RepositoryCore,
	getSHA func(params git.ReadParams) (sha.SHA, error),
) (sha.SHA, *types.RepositoryCore, error) {
	if repoForkCore.ForkID == 0 {
		return sha.None, nil, errors.InvalidArgument("Repository is not a fork.")
	}

	repoUpstreamCore, err := s.repoFinder.FindByID(ctx, repoForkCore.ForkID)
	if err != nil {
		return sha.None, nil, fmt.Errorf("failed to find upstream repo: %w", err)
	}

	if err = apiauth.CheckRepo(
		ctx,
		s.authorizer,
		session,
		repoUpstreamCore,
		enum.PermissionRepoView,
	); errors.Is(err, apiauth.ErrForbidden) {
		return sha.None, nil, usererror.BadRequest(
			"Not enough permissions to view the upstream repository.",
		)
	} else if err != nil {
		return sha.None, nil, fmt.Errorf("failed to check access to upstream repo: %w", err)
	}

	upstreamSHA, err := getSHA(git.CreateReadParams(repoUpstreamCore))
	if err != nil {
		return sha.None, nil, fmt.Errorf("failed to get upstream repo SHA: %w", err)
	}

	writeParams, err := controller.CreateRPCSystemReferencesWriteParams(ctx, s.urlProvider, session, repoForkCore)
	if err != nil {
		return sha.None, nil, fmt.Errorf("failed to create RPC write params: %w", err)
	}

	_, err = s.git.FetchObjects(ctx, &git.FetchObjectsParams{
		WriteParams: writeParams,
		Source:      repoUpstreamCore.GitUID,
		ObjectSHAs:  []sha.SHA{upstreamSHA},
	})
	if err != nil {
		return sha.None, nil, fmt.Errorf("failed to fetch commit from upstream repo: %w", err)
	}

	return upstreamSHA, repoUpstreamCore, nil
}
