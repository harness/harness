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

package netrc

import (
	"context"

	"github.com/drone/drone/core"
	"github.com/drone/go-scm/scm"
)

var _ core.NetrcService = (*Service)(nil)

// Service implements a netrc file generation service.
type Service struct {
	client   *scm.Client
	renewer  core.Renewer
	private  bool
	username string
	password string
}

// New returns a new Netrc service.
func New(
	client *scm.Client,
	renewer core.Renewer,
	private bool,
	username string,
	password string,
) core.NetrcService {
	return &Service{
		client:   client,
		renewer:  renewer,
		private:  private,
		username: username,
		password: password,
	}
}

// Create creates a netrc file for the user and repository.
func (s *Service) Create(ctx context.Context, user *core.User, repo *core.Repository) (*core.Netrc, error) {
	// if the repository is public and private mode is disabled,
	// authentication is not required.
	if repo.Private == false && s.private == false {
		return nil, nil
	}

	netrc := new(core.Netrc)
	err := netrc.SetMachine(repo.HTTPURL)
	if err != nil {
		return nil, err
	}

	if s.username != "" && s.password != "" {
		netrc.Password = s.password
		netrc.Login = s.username
		return netrc, nil
	}

	// force refresh the authorization token to prevent
	// it from expiring during pipeline execution.
	err = s.renewer.Renew(ctx, user, true)
	if err != nil {
		return nil, err
	}

	switch s.client.Driver {
	case scm.DriverGitlab, scm.DriverGitee:
		netrc.Login = "oauth2"
		netrc.Password = user.Token
	case scm.DriverBitbucket:
		netrc.Login = "x-token-auth"
		netrc.Password = user.Token
	case scm.DriverGithub, scm.DriverGogs, scm.DriverGitea:
		netrc.Password = "x-oauth-basic"
		netrc.Login = user.Token
	}
	return netrc, nil
}
