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

package orgs

import (
	"context"
	"time"

	"github.com/drone/drone/core"
	"github.com/drone/go-scm/scm"
)

// New returns a new OrganizationService.
func New(client *scm.Client, renewer core.Renewer) core.OrganizationService {
	return &service{
		client:  client,
		renewer: renewer,
	}
}

type service struct {
	renewer core.Renewer
	client  *scm.Client
}

func (s *service) List(ctx context.Context, user *core.User) ([]*core.Organization, error) {
	err := s.renewer.Renew(ctx, user, false)
	if err != nil {
		return nil, err
	}
	token := &scm.Token{
		Token:   user.Token,
		Refresh: user.Refresh,
	}
	if user.Expiry != 0 {
		token.Expires = time.Unix(user.Expiry, 0)
	}
	ctx = context.WithValue(ctx, scm.TokenKey{}, token)
	out, _, err := s.client.Organizations.List(ctx, scm.ListOptions{Size: 100})
	if err != nil {
		return nil, err
	}
	var orgs []*core.Organization
	for _, org := range out {
		orgs = append(orgs, &core.Organization{
			Name:   org.Name,
			Avatar: org.Avatar,
		})
	}
	return orgs, nil
}

func (s *service) Membership(ctx context.Context, user *core.User, name string) (bool, bool, error) {
	err := s.renewer.Renew(ctx, user, false)
	if err != nil {
		return false, false, err
	}
	token := &scm.Token{
		Token:   user.Token,
		Refresh: user.Refresh,
	}
	if user.Expiry != 0 {
		token.Expires = time.Unix(user.Expiry, 0)
	}
	ctx = context.WithValue(ctx, scm.TokenKey{}, token)
	out, _, err := s.client.Organizations.FindMembership(ctx, name, user.Login)
	if err != nil {
		return false, false, err
	}
	switch {
	case out.Active == false:
		return false, false, nil
	case out.Role == scm.RoleUndefined:
		return false, false, nil
	case out.Role == scm.RoleAdmin:
		return true, true, nil
	default:
		return true, false, nil
	}
}
