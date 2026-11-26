// Copyright 2026 Harness, Inc.
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
	"testing"

	"github.com/harness/gitness/app/api/controller/service"
	"github.com/harness/gitness/app/api/controller/user"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/store"
	gitnessstore "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"

	"github.com/stretchr/testify/require"
)

// infPrincipalStore is an in-memory principal store for services and users.
type infPrincipalStore struct {
	store.PrincipalStore

	services map[string]*types.Service
	users    map[string]*types.User

	nextID        int64
	findSvcErr    error
	createSvcErr  error
	findUserErr   error
	createUserErr error
	// createSvcRace simulates another instance creating the service first.
	createSvcRace bool
	createdSvcs   []string
}

const infAdminPassword = "admin-pass-1234"

func infNewStore() *infPrincipalStore {
	return &infPrincipalStore{
		services: map[string]*types.Service{},
		users:    map[string]*types.User{},
		nextID:   10,
	}
}

func (s *infPrincipalStore) FindServiceByUID(_ context.Context, uid string) (*types.Service, error) {
	if s.findSvcErr != nil {
		return nil, s.findSvcErr
	}
	svc, ok := s.services[uid]
	if !ok {
		return nil, gitnessstore.ErrResourceNotFound
	}
	return svc, nil
}

func (s *infPrincipalStore) CreateService(_ context.Context, svc *types.Service) error {
	if s.createSvcErr != nil {
		return s.createSvcErr
	}
	s.nextID++
	svc.ID = s.nextID
	cp := *svc
	if s.createSvcRace {
		// someone else created the service meanwhile; this attempt fails as duplicate.
		s.createSvcRace = false
		s.services[svc.UID] = &cp
		return gitnessstore.ErrDuplicate
	}
	s.services[svc.UID] = &cp
	s.createdSvcs = append(s.createdSvcs, svc.UID)
	return nil
}

func (s *infPrincipalStore) FindUserByUID(_ context.Context, uid string) (*types.User, error) {
	if s.findUserErr != nil {
		return nil, s.findUserErr
	}
	u, ok := s.users[uid]
	if !ok {
		return nil, gitnessstore.ErrResourceNotFound
	}
	return u, nil
}

func (s *infPrincipalStore) CreateUser(_ context.Context, u *types.User) error {
	if s.createUserErr != nil {
		return s.createUserErr
	}
	s.nextID++
	u.ID = s.nextID
	cp := *u
	s.users[u.UID] = &cp
	return nil
}

func (s *infPrincipalStore) CountUsers(context.Context, *types.UserFilter) (int64, error) {
	return int64(len(s.users)), nil
}

func (s *infPrincipalStore) UpdateUser(_ context.Context, u *types.User) error {
	cp := *u
	s.users[u.UID] = &cp
	return nil
}

func infControllers(st *infPrincipalStore) (*user.Controller, *service.Controller) {
	userCtrl := user.NewController(nil, check.PrincipalUIDDefault, nil, st, nil, nil, nil, nil, nil, nil,
		refcache.RepoFinder{}, nil)
	serviceCtrl := service.NewController(check.PrincipalUIDDefault, nil, st)
	return userCtrl, serviceCtrl
}

func infConfig() *types.Config {
	cfg := &types.Config{}
	cfg.Principal.System.UID = "sys-svc"
	cfg.Principal.System.Email = "system@example.invalid"
	cfg.Principal.System.DisplayName = "System"
	cfg.Principal.Pipeline.UID = "pipe-svc"
	cfg.Principal.Pipeline.Email = "pipeline@example.invalid"
	cfg.Principal.Pipeline.DisplayName = "Pipeline"
	cfg.Principal.Gitspace.UID = "space-svc"
	cfg.Principal.Gitspace.Email = "gitspace@example.invalid"
	cfg.Principal.Gitspace.DisplayName = "Gitspace"
	cfg.Principal.Admin.UID = "admin"
	cfg.Principal.Admin.Email = "admin@example.invalid"
	cfg.Principal.Admin.DisplayName = "Admin"
	return cfg
}

func TestInfServices_CreatedWhenMissing(t *testing.T) {
	ctx := context.Background()
	st := infNewStore()
	_, serviceCtrl := infControllers(st)
	cfg := infConfig()

	require.NoError(t, SystemService(ctx, cfg, serviceCtrl))
	require.NoError(t, PipelineService(ctx, cfg, serviceCtrl))
	require.NoError(t, GitspaceService(ctx, cfg, serviceCtrl))

	require.Equal(t, []string{"sys-svc", "pipe-svc", "space-svc"}, st.createdSvcs)
	require.True(t, st.services["sys-svc"].Admin, "system service must be an admin")
	require.False(t, st.services["pipe-svc"].Admin)
	require.False(t, st.services["space-svc"].Admin)

	require.Equal(t, "sys-svc", NewSystemServiceSession().Principal.UID)
	require.Equal(t, "pipe-svc", NewPipelineServiceSession().Principal.UID)
	require.Equal(t, "space-svc", NewGitspaceServiceSession().Principal.UID)
	require.NotNil(t, NewSystemServiceSession().Metadata)
}

func TestInfServices_ExistingServicesAreReused(t *testing.T) {
	ctx := context.Background()
	st := infNewStore()
	st.services["sys-svc"] = &types.Service{ID: 1, UID: "sys-svc", Admin: true}
	st.services["pipe-svc"] = &types.Service{ID: 2, UID: "pipe-svc"}
	st.services["space-svc"] = &types.Service{ID: 3, UID: "space-svc"}
	_, serviceCtrl := infControllers(st)
	cfg := infConfig()

	require.NoError(t, SystemService(ctx, cfg, serviceCtrl))
	require.NoError(t, PipelineService(ctx, cfg, serviceCtrl))
	require.NoError(t, GitspaceService(ctx, cfg, serviceCtrl))
	require.Empty(t, st.createdSvcs)
	require.EqualValues(t, 1, NewSystemServiceSession().Principal.ID)
	require.EqualValues(t, 2, NewPipelineServiceSession().Principal.ID)
	require.EqualValues(t, 3, NewGitspaceServiceSession().Principal.ID)
}

func TestInfSystemService_NonAdminIsRejected(t *testing.T) {
	st := infNewStore()
	st.services["sys-svc"] = &types.Service{ID: 1, UID: "sys-svc", Admin: false}
	_, serviceCtrl := infControllers(st)
	err := SystemService(context.Background(), infConfig(), serviceCtrl)
	require.ErrorContains(t, err, "is no admin")
}

func TestInfServices_StoreErrors(t *testing.T) {
	ctx := context.Background()
	cfg := infConfig()
	funcs := map[string]func(context.Context, *types.Config, *service.Controller) error{
		"system":   SystemService,
		"pipeline": PipelineService,
		"gitspace": GitspaceService,
	}
	for name, fn := range funcs {
		t.Run(name+" find error", func(t *testing.T) {
			st := infNewStore()
			st.findSvcErr = errors.New("db down")
			_, serviceCtrl := infControllers(st)
			require.ErrorContains(t, fn(ctx, cfg, serviceCtrl), "db down")
		})
		t.Run(name+" create error", func(t *testing.T) {
			st := infNewStore()
			st.createSvcErr = errors.New("insert failed")
			_, serviceCtrl := infControllers(st)
			require.ErrorContains(t, fn(ctx, cfg, serviceCtrl), "insert failed")
		})
		t.Run(name+" duplicate resolved by lookup", func(t *testing.T) {
			st := infNewStore()
			st.createSvcRace = true
			_, serviceCtrl := infControllers(st)
			require.NoError(t, fn(ctx, cfg, serviceCtrl))
		})
	}
}

func TestInfCreateServicePrincipal_DuplicateThenLookupFails(t *testing.T) {
	st := infNewStore()
	st.createSvcErr = gitnessstore.ErrDuplicate
	_, serviceCtrl := infControllers(st)
	_, err := createServicePrincipal(context.Background(), serviceCtrl, "dup-svc", "dup@example.invalid", "Dup", false)
	require.ErrorContains(t, err, "after duplicate error")
	require.ErrorIs(t, err, gitnessstore.ErrDuplicate)
}

func TestInfAdminUser(t *testing.T) {
	ctx := context.Background()

	t.Run("skipped without password", func(t *testing.T) {
		userCtrl, _ := infControllers(infNewStore())
		require.NoError(t, AdminUser(ctx, infConfig(), userCtrl))
	})
	t.Run("requires an email", func(t *testing.T) {
		cfg := infConfig()
		cfg.Principal.Admin.Password = infAdminPassword
		cfg.Principal.Admin.Email = ""
		userCtrl, _ := infControllers(infNewStore())
		require.ErrorIs(t, AdminUser(ctx, cfg, userCtrl), ErrAdminEmailRequired)
	})
	t.Run("creates the admin user", func(t *testing.T) {
		cfg := infConfig()
		cfg.Principal.Admin.Password = infAdminPassword
		st := infNewStore()
		userCtrl, _ := infControllers(st)
		require.NoError(t, AdminUser(ctx, cfg, userCtrl))
		created := st.users["admin"]
		require.NotNil(t, created)
		require.True(t, created.Admin)
		require.NotEqual(t, infAdminPassword, created.Password, "password must be stored hashed")
	})
	t.Run("existing admin is accepted", func(t *testing.T) {
		cfg := infConfig()
		cfg.Principal.Admin.Password = infAdminPassword
		st := infNewStore()
		st.users["admin"] = &types.User{ID: 1, UID: "admin", Admin: true}
		userCtrl, _ := infControllers(st)
		require.NoError(t, AdminUser(ctx, cfg, userCtrl))
	})
	t.Run("existing non admin is rejected", func(t *testing.T) {
		cfg := infConfig()
		cfg.Principal.Admin.Password = infAdminPassword
		st := infNewStore()
		st.users["admin"] = &types.User{ID: 1, UID: "admin", Admin: false}
		userCtrl, _ := infControllers(st)
		require.ErrorContains(t, AdminUser(ctx, cfg, userCtrl), "exists but is no admin")
	})
	t.Run("lookup error", func(t *testing.T) {
		cfg := infConfig()
		cfg.Principal.Admin.Password = infAdminPassword
		st := infNewStore()
		st.findUserErr = errors.New("db down")
		userCtrl, _ := infControllers(st)
		require.ErrorContains(t, AdminUser(ctx, cfg, userCtrl), "failed to setup admin user")
	})
	t.Run("creation error", func(t *testing.T) {
		cfg := infConfig()
		cfg.Principal.Admin.Password = infAdminPassword
		st := infNewStore()
		st.createUserErr = errors.New("insert failed")
		userCtrl, _ := infControllers(st)
		require.ErrorContains(t, AdminUser(ctx, cfg, userCtrl), "insert failed")
	})
}

func TestInfSystem(t *testing.T) {
	ctx := context.Background()

	t.Run("bootstraps everything", func(t *testing.T) {
		cfg := infConfig()
		cfg.Principal.Admin.Password = infAdminPassword
		st := infNewStore()
		userCtrl, serviceCtrl := infControllers(st)
		require.NoError(t, System(cfg, userCtrl, serviceCtrl)(ctx))
		require.Len(t, st.services, 3)
		require.Len(t, st.users, 1)
	})
	t.Run("service failure is wrapped", func(t *testing.T) {
		st := infNewStore()
		st.createSvcErr = errors.New("insert failed")
		userCtrl, serviceCtrl := infControllers(st)
		err := System(infConfig(), userCtrl, serviceCtrl)(ctx)
		require.ErrorContains(t, err, "failed to setup system service")
	})
	t.Run("admin failure is wrapped", func(t *testing.T) {
		cfg := infConfig()
		cfg.Principal.Admin.Password = infAdminPassword
		st := infNewStore()
		st.createUserErr = errors.New("insert failed")
		userCtrl, serviceCtrl := infControllers(st)
		err := System(cfg, userCtrl, serviceCtrl)(ctx)
		require.ErrorContains(t, err, "failed to setup admin user")
	})
}
