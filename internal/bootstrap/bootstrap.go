// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package bootstrap

import (
	"context"
	"errors"
	"fmt"

	"github.com/harness/gitness/internal/api/controller/service"
	"github.com/harness/gitness/internal/api/controller/user"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/store"
	"github.com/harness/gitness/types"

	"github.com/rs/zerolog/log"
)

// systemServicePrincipal is the principal representing gitness.
// It is used for all operations executed by gitness itself.
var systemServicePrincipal *types.Principal

func NewSystemServiceSession() *auth.Session {
	return &auth.Session{
		Principal: *systemServicePrincipal,
		Metadata:  &auth.EmptyMetadata{},
	}
}

// Bootstrap is an abstraction of a function that bootstraps a system.
type Bootstrap func(context.Context) error

func System(config *types.Config, userCtrl *user.Controller,
	serviceCtrl *service.Controller) func(context.Context) error {
	return func(ctx context.Context) error {
		if err := SystemService(ctx, config, serviceCtrl); err != nil {
			return err
		}

		if err := AdminUser(ctx, config, userCtrl); err != nil {
			return err
		}

		return nil
	}
}

// AdminUser sets up the admin user based on the config (if provided).
func AdminUser(ctx context.Context, config *types.Config, userCtrl *user.Controller) error {
	if config.Principal.Admin.Password == "" {
		return nil
	}

	usr, err := userCtrl.FindNoAuth(ctx, config.Principal.Admin.UID)
	if errors.Is(err, store.ErrResourceNotFound) {
		usr, err = createAdminUser(ctx, config, userCtrl)
	}

	if err != nil {
		return fmt.Errorf("failed to setup admin user: %w", err)
	}
	if !usr.Admin {
		return fmt.Errorf("user with uid '%s' exists but is no admin (ID: %d)", usr.UID, usr.ID)
	}

	log.Ctx(ctx).Info().Msgf("Completed setup of admin user '%s' (id: %d).", usr.UID, usr.ID)

	return nil
}

func createAdminUser(ctx context.Context, config *types.Config, userCtrl *user.Controller) (*types.User, error) {
	in := &user.CreateInput{
		UID:         config.Principal.Admin.UID,
		DisplayName: config.Principal.Admin.DisplayName,
		Email:       config.Principal.Admin.Email,
		Password:    config.Principal.Admin.Password,
	}

	usr, createErr := userCtrl.CreateNoAuth(ctx, in, true)
	if createErr == nil || !errors.Is(createErr, store.ErrDuplicate) {
		return usr, createErr
	}

	// user might've been created by another instance in which case we should find it now.
	var findErr error
	usr, findErr = userCtrl.FindNoAuth(ctx, config.Principal.Admin.UID)
	if findErr != nil {
		return nil, fmt.Errorf("failed to find user with uid '%s' (%s) after duplicate error: %w",
			config.Principal.Admin.UID, findErr, createErr)
	}

	return usr, nil
}

// SystemService sets up the gitness service principal that is used for
// resources that are automatically created by the system.
func SystemService(ctx context.Context, config *types.Config, serviceCtrl *service.Controller) error {
	svc, err := serviceCtrl.FindNoAuth(ctx, config.Principal.System.UID)
	if errors.Is(err, store.ErrResourceNotFound) {
		svc, err = createSystemService(ctx, config, serviceCtrl)
	}

	if err != nil {
		return fmt.Errorf("failed to setup system service: %w", err)
	}
	if !svc.Admin {
		return fmt.Errorf("service with uid '%s' exists but is no admin (ID: %d)", svc.UID, svc.ID)
	}

	systemServicePrincipal = svc.ToPrincipal()

	log.Ctx(ctx).Info().Msgf("Completed setup of system service '%s' (id: %d).", svc.UID, svc.ID)

	return nil
}

func createSystemService(ctx context.Context, config *types.Config,
	serviceCtrl *service.Controller) (*types.Service, error) {
	in := &service.CreateInput{
		UID:         config.Principal.System.UID,
		Email:       config.Principal.System.Email,
		DisplayName: config.Principal.System.DisplayName,
	}

	svc, createErr := serviceCtrl.CreateNoAuth(ctx, in, true)
	if createErr == nil || !errors.Is(createErr, store.ErrDuplicate) {
		return svc, createErr
	}

	// service might've been created by another instance in which case we should find it now.
	var findErr error
	svc, findErr = serviceCtrl.FindNoAuth(ctx, config.Principal.System.UID)
	if findErr != nil {
		return nil, fmt.Errorf("failed to find service with uid '%s' (%s) after duplicate error: %w",
			config.Principal.System.UID, findErr, createErr)
	}

	return svc, nil
}
