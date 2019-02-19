// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

package license

import (
	"context"
	"time"

	"github.com/drone/drone/core"
)

// NewService returns a new License service.
func NewService(
	users core.UserStore,
	repos core.RepositoryStore,
	builds core.BuildStore,
	license *core.License,
) core.LicenseService {
	return &service{
		users:   users,
		repos:   repos,
		builds:  builds,
		license: license,
	}
}

type service struct {
	users   core.UserStore
	repos   core.RepositoryStore
	builds  core.BuildStore
	license *core.License
}

func (s *service) Exceeded(ctx context.Context) (bool, error) {
	if limit := s.license.Builds; limit > 0 {
		count, _ := s.builds.Count(ctx)
		if count > limit {
			return true, core.ErrBuildLimit
		}
	}
	if limit := s.license.Users; limit > 0 {
		count, _ := s.users.Count(ctx)
		if count > limit {
			return true, core.ErrUserLimit
		}
	}
	if limit := s.license.Repos; limit > 0 {
		count, _ := s.repos.Count(ctx)
		if count > limit {
			return true, core.ErrRepoLimit
		}
	}
	return false, nil
}

func (s *service) Expired(ctx context.Context) bool {
	return s.license.Expired()
}

func (s *service) Expires(ctx context.Context) time.Time {
	return s.license.Expires
}
