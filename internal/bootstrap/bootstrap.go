// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package bootstrap

import (
	"context"
	"errors"
	"fmt"

	"github.com/harness/gitness/internal/api/controller/user"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types"
	"github.com/rs/zerolog/log"
)

const (
	adminUID = "admin"
)

// Bootstrap is an abstraction of a function that bootstraps a system.
type Bootstrap func(context.Context) error

func System(config *types.Config, userCtrl *user.Controller) func(context.Context) error {
	return func(ctx context.Context) error {
		return Admin(ctx, config, userCtrl)
	}
}

// Admin sets up the admin user based on the config (if provided)
//
// NOTE: We could just call update and ignore any duplicate error
// but then the duplicte might be due to email or name, not uid.
// Futhermore, it would create unnecesary error logs.
func Admin(ctx context.Context, config *types.Config, userCtrl *user.Controller) error {
	if config.Admin.Name == "" {
		return nil
	}

	_, err := userCtrl.FindNoAuth(ctx, adminUID)
	if err == nil || !errors.Is(err, store.ErrResourceNotFound) {
		return err
	}

	in := &user.CreateInput{
		UID:      adminUID,
		Name:     config.Admin.Name,
		Email:    config.Admin.Email,
		Password: config.Admin.Password,
	}

	// create user as admin
	usr, err := userCtrl.CreateNoAuth(ctx, in, true)
	if errors.Is(err, store.ErrDuplicate) {
		// user might've been created by another instance
		// in which case we should find the user now.
		_, err2 := userCtrl.FindNoAuth(ctx, adminUID)
		if err2 != nil {
			// return original error.
			return err
		}

		// user exists - perfect
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to create admin user: %w", err)
	}

	log.Ctx(ctx).Info().Msgf("Created admin user (id: %d).", usr.ID)

	return nil
}
