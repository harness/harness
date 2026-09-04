// Copyright 2019 Drone IO, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package repo

import (
	"context"
	"sync"

	"github.com/drone/drone/core"
	"github.com/drone/go-scm/scm"

	"golang.org/x/sync/errgroup"
)

// listPageConcurrency bounds how many repository-list pages are
// fetched from the SCM provider at once, to stay well clear of
// secondary rate limits (e.g. GitHub's abuse-detection mechanism).
const listPageConcurrency = 6

type service struct {
	renew      core.Renewer
	client     *scm.Client
	visibility string
	trusted    bool
}

// New returns a new Repository service, providing access to the
// repository information from the source code management system.
func New(client *scm.Client, renewer core.Renewer, visibility string, trusted bool) core.RepositoryService {
	return &service{
		renew:      renewer,
		client:     client,
		visibility: visibility,
		trusted:    trusted,
	}
}

func (s *service) List(ctx context.Context, user *core.User) ([]*core.Repository, error) {
	err := s.renew.Renew(ctx, user, false)
	if err != nil {
		return nil, err
	}

	ctx = context.WithValue(ctx, scm.TokenKey{}, &scm.Token{
		Token:   user.Token,
		Refresh: user.Refresh,
	})

	opts := scm.ListOptions{Size: 100}
	result, meta, err := s.client.Repositories.List(ctx, opts)
	if err != nil {
		return nil, err
	}
	repos := convertRepositories(result, s.visibility, s.trusted)

	if meta.Page.Last > 1 {
		// the provider told us the total page count up front (e.g.
		// GitHub's Link: rel="last" header, surfaced as meta.Page.Last)
		// -- fetch the remaining pages concurrently instead of walking
		// them one at a time. Order doesn't matter to any caller of
		// List, so pages are appended as they complete.
		var mu sync.Mutex
		group, gctx := errgroup.WithContext(ctx)
		group.SetLimit(listPageConcurrency)
		for page := 2; page <= meta.Page.Last; page++ {
			page := page
			group.Go(func() error {
				pageOpts := scm.ListOptions{Size: opts.Size, Page: page}
				result, _, err := s.client.Repositories.List(gctx, pageOpts)
				if err != nil {
					return err
				}
				converted := convertRepositories(result, s.visibility, s.trusted)
				mu.Lock()
				repos = append(repos, converted...)
				mu.Unlock()
				return nil
			})
		}
		if err := group.Wait(); err != nil {
			return nil, err
		}
		return repos, nil
	}

	// the provider only exposes an opaque next-page cursor with no
	// advance knowledge of how many pages exist -- fall back to the
	// sequential walk.
	for opts.Page, opts.URL = meta.Page.Next, meta.Page.NextURL; opts.Page != 0 || opts.URL != ""; {
		result, meta, err := s.client.Repositories.List(ctx, opts)
		if err != nil {
			return nil, err
		}
		repos = append(repos, convertRepositories(result, s.visibility, s.trusted)...)
		opts.Page, opts.URL = meta.Page.Next, meta.Page.NextURL
	}
	return repos, nil
}

func (s *service) Find(ctx context.Context, user *core.User, repo string) (*core.Repository, error) {
	err := s.renew.Renew(ctx, user, false)
	if err != nil {
		return nil, err
	}

	ctx = context.WithValue(ctx, scm.TokenKey{}, &scm.Token{
		Token:   user.Token,
		Refresh: user.Refresh,
	})
	result, _, err := s.client.Repositories.Find(ctx, repo)
	if err != nil {
		return nil, err
	}
	return convertRepository(result, s.visibility, s.trusted), nil
}

func (s *service) FindPerm(ctx context.Context, user *core.User, repo string) (*core.Perm, error) {
	err := s.renew.Renew(ctx, user, false)
	if err != nil {
		return nil, err
	}

	ctx = context.WithValue(ctx, scm.TokenKey{}, &scm.Token{
		Token:   user.Token,
		Refresh: user.Refresh,
	})
	result, _, err := s.client.Repositories.FindPerms(ctx, repo)
	if err != nil {
		return nil, err
	}
	return &core.Perm{
		Read:  result.Pull,
		Write: result.Push,
		Admin: result.Admin,
	}, nil
}
