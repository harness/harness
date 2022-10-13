// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package database

import (
	"context"

	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/internal/store/database/mutex"
	"github.com/harness/gitness/types"
)

var _ store.ServiceStore = (*ServiceStoreSync)(nil)

// NewServiceStoreSync returns a new ServiceStoreSync.
func NewServiceStoreSync(base *ServiceStore) *ServiceStoreSync {
	return &ServiceStoreSync{base}
}

// ServiceStoreSync synchronizes read and write access to the
// service store. This prevents race conditions when the database
// type is sqlite3.
type ServiceStoreSync struct {
	base *ServiceStore
}

// Find finds the service by id.
func (s *ServiceStoreSync) Find(ctx context.Context, id int64) (*types.Service, error) {
	mutex.RLock()
	defer mutex.RUnlock()
	return s.base.Find(ctx, id)
}

// FindUID finds the service by uid.
func (s *ServiceStoreSync) FindUID(ctx context.Context, uid string) (*types.Service, error) {
	mutex.RLock()
	defer mutex.RUnlock()
	return s.base.FindUID(ctx, uid)
}

// Create saves the service.
func (s *ServiceStoreSync) Create(ctx context.Context, sa *types.Service) error {
	mutex.RLock()
	defer mutex.RUnlock()
	return s.base.Create(ctx, sa)
}

// Update updates the service.
func (s *ServiceStoreSync) Update(ctx context.Context, sa *types.Service) error {
	mutex.RLock()
	defer mutex.RUnlock()
	return s.base.Update(ctx, sa)
}

// Delete deletes the service.
func (s *ServiceStoreSync) Delete(ctx context.Context, id int64) error {
	mutex.RLock()
	defer mutex.RUnlock()
	return s.base.Delete(ctx, id)
}

// List returns a list of service for a specific parent.
func (s *ServiceStoreSync) List(ctx context.Context) ([]*types.Service, error) {
	mutex.RLock()
	defer mutex.RUnlock()
	return s.base.List(ctx)
}

// Count returns a count of service for a specific parent.
func (s *ServiceStoreSync) Count(ctx context.Context) (int64, error) {
	mutex.RLock()
	defer mutex.RUnlock()
	return s.base.Count(ctx)
}
