// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

// +build !oss

package admission

import (
	"context"
	"errors"
	"strings"

	"github.com/drone/drone/core"
)

// ErrMembership is returned when attempting to create a new
// user account for a user that is not a member of an approved
// organization.
var ErrMembership = errors.New("User must be a member of an approved organization")

// Membership limits user access by organization membership.
func Membership(service core.OrganizationService, accounts []string) core.AdmissionService {
	lookup := map[string]struct{}{}
	for _, account := range accounts {
		account = strings.TrimSpace(account)
		account = strings.ToLower(account)
		lookup[account] = struct{}{}
	}
	return &membership{service: service, account: lookup}
}

type membership struct {
	service core.OrganizationService
	account map[string]struct{}
}

func (s *membership) Admit(ctx context.Context, user *core.User) error {
	// this admission policy is only enforced for
	// new users. Existing users are always admitted.
	if user.ID != 0 {
		return nil
	}

	// if the membership whitelist is empty assume the system
	// is open admission.
	if len(s.account) == 0 {
		return nil
	}
	// if the username is in the whitelist when can admin
	// the user without making an API call to fetch the
	// organization list.
	_, ok := s.account[strings.ToLower(user.Login)]
	if ok {
		return nil
	}
	// make an API call to retrive the list of organizations
	// to which the user belongs.
	orgs, err := s.service.List(ctx, user)
	if err != nil {
		return err
	}
	// if the user is a member of an organization in the
	// account whitelist we can admit the user.
	for _, org := range orgs {
		_, ok := s.account[strings.ToLower(org.Name)]
		if ok {
			return nil
		}
	}
	// else deny access
	return ErrMembership
}
