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

package bootstrap

import (
	"context"
	"errors"
	"fmt"

	"github.com/harness/gitness/app/api/controller/service"
	"github.com/harness/gitness/app/api/controller/user"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/store"
	"github.com/harness/gitness/types"

	"github.com/rs/zerolog/log"
)

// systemServicePrincipal is the principal representing gitness.
// It is used for all operations executed by gitness itself.
var systemServicePrincipal *types.Principal

var ErrAdminEmailRequired = errors.New("config.Principal.Admin.Email is required")

func NewSystemServiceSession() *auth.Session {
	return &auth.Session{
		Principal: *systemServicePrincipal,
		Metadata:  &auth.EmptyMetadata{},
	}
}

// pipelineServicePrincipal is the principal that is used during
// pipeline executions for calling gitness APIs.
var pipelineServicePrincipal *types.Principal

func NewPipelineServiceSession() *auth.Session {
	return &auth.Session{
		Principal: *pipelineServicePrincipal,
		Metadata:  &auth.EmptyMetadata{},
	}
}

// gitspaceServicePrincipal is the principal that is used during
// gitspace token injection for calling gitness APIs.
var gitspaceServicePrincipal *types.Principal

func NewGitspaceServiceSession() *auth.Session {
	return &auth.Session{
		Principal: *gitspaceServicePrincipal,
		Metadata:  &auth.EmptyMetadata{},
	}
}

// Bootstrap is an abstraction of a function that bootstraps a system.
type Bootstrap func(context.Context) error

func System(config *types.Config, userCtrl *user.Controller,
	serviceCtrl *service.Controller) func(context.Context) error {
	return func(ctx context.Context) error {
		if err := SystemService(ctx, config, serviceCtrl); err != nil {
			return fmt.Errorf("failed to setup system service: %w", err)
		}

		if err := PipelineService(ctx, config, serviceCtrl); err != nil {
			return fmt.Errorf("failed to setup pipeline service: %w", err)
		}
		if err := GitspaceService(ctx, config, serviceCtrl); err != nil {
			return fmt.Errorf("failed to setup gitspace service: %w", err)
		}

		if err := AdminUser(ctx, config, userCtrl); err != nil {
			return fmt.Errorf("failed to setup admin user: %w", err)
		}

		return nil
	}
}

// AdminUser sets up the admin user based on the config (if provided).
func AdminUser(ctx context.Context, config *types.Config, userCtrl *user.Controller) error {
	if config.Principal.Admin.Password == "" {
		return nil
	}

	if config.Principal.Admin.Email == "" {
		return fmt.Errorf("failed to set up admin user: %w", ErrAdminEmailRequired)
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

func createAdminUser(
	ctx context.Context,
	config *types.Config,
	userCtrl *user.Controller,
) (*types.User, error) {
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
		return nil, fmt.Errorf(
			"failed to find user with uid '%s' (%w) after duplicate error: %w",
			config.Principal.Admin.UID,
			findErr,
			createErr,
		)
	}

	return usr, nil
}

// SystemService sets up the gitness service principal that is used for
// resources that are automatically created by the system.
func SystemService(
	ctx context.Context,
	config *types.Config,
	serviceCtrl *service.Controller,
) error {
	svc, err := serviceCtrl.FindNoAuth(ctx, config.Principal.System.UID)
	if errors.Is(err, store.ErrResourceNotFound) {
		svc, err = createServicePrincipal(
			ctx,
			serviceCtrl,
			config.Principal.System.UID,
			config.Principal.System.Email,
			config.Principal.System.DisplayName,
			true,
		)
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

// PipelineService sets up the pipeline service principal that is used during
// pipeline executions for calling gitness APIs.
func PipelineService(
	ctx context.Context,
	config *types.Config,
	serviceCtrl *service.Controller,
) error {
	svc, err := serviceCtrl.FindNoAuth(ctx, config.Principal.Pipeline.UID)
	if errors.Is(err, store.ErrResourceNotFound) {
		svc, err = createServicePrincipal(
			ctx,
			serviceCtrl,
			config.Principal.Pipeline.UID,
			config.Principal.Pipeline.Email,
			config.Principal.Pipeline.DisplayName,
			false,
		)
	}

	if err != nil {
		return fmt.Errorf("failed to setup pipeline service: %w", err)
	}

	pipelineServicePrincipal = svc.ToPrincipal()

	log.Ctx(ctx).Info().Msgf("Completed setup of pipeline service '%s' (id: %d).", svc.UID, svc.ID)

	return nil
}

// GitspaceService sets up the gitspace service principal that is used during
// gitspace credential injection for calling gitness APIs.
func GitspaceService(
	ctx context.Context,
	config *types.Config,
	serviceCtrl *service.Controller,
) error {
	svc, err := serviceCtrl.FindNoAuth(ctx, config.Principal.Gitspace.UID)
	if errors.Is(err, store.ErrResourceNotFound) {
		svc, err = createServicePrincipal(
			ctx,
			serviceCtrl,
			config.Principal.Gitspace.UID,
			config.Principal.Gitspace.Email,
			config.Principal.Gitspace.DisplayName,
			false,
		)
	}

	if err != nil {
		return fmt.Errorf("failed to setup gitspace service: %w", err)
	}

	gitspaceServicePrincipal = svc.ToPrincipal()

	log.Ctx(ctx).Info().Msgf("Completed setup of gitspace service '%s' (id: %d).", svc.UID, svc.ID)

	return nil
}

func createServicePrincipal(
	ctx context.Context,
	serviceCtrl *service.Controller,
	uid string,
	email string,
	displayName string,
	admin bool,
) (*types.Service, error) {
	in := &service.CreateInput{
		UID:         uid,
		Email:       email,
		DisplayName: displayName,
	}

	svc, createErr := serviceCtrl.CreateNoAuth(ctx, in, admin)
	if createErr == nil || !errors.Is(createErr, store.ErrDuplicate) {
		return svc, createErr
	}

	// service might've been created by another instance in which case we should find it now.
	var findErr error
	svc, findErr = serviceCtrl.FindNoAuth(ctx, uid)
	if findErr != nil {
		return nil, fmt.Errorf(
			"failed to find service with uid '%s' (%w) after duplicate error: %w",
			uid,
			findErr,
			createErr,
		)
	}

	return svc, nil
}
