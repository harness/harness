// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package database

import (
	"context"

	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/internal/store/database/mutex"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

var _ store.ServiceAccountStore = (*ServiceAccountStoreSync)(nil)

// NewServiceAccountStoreSync returns a new ServiceAccountStoreSync.
func NewServiceAccountStoreSync(base *ServiceAccountStore) *ServiceAccountStoreSync {
	return &ServiceAccountStoreSync{base}
}

// ServiceAccountStoreSync synchronizes read and write access to the
// service account store. This prevents race conditions when the database
// type is sqlite3.
type ServiceAccountStoreSync struct {
	base *ServiceAccountStore
}

// Find finds the service account by id.
func (s *ServiceAccountStoreSync) Find(ctx context.Context, id int64) (*types.ServiceAccount, error) {
	mutex.RLock()
	defer mutex.RUnlock()
	return s.base.Find(ctx, id)
}

// FindUID finds the service account by uid.
func (s *ServiceAccountStoreSync) FindUID(ctx context.Context, uid string) (*types.ServiceAccount, error) {
	mutex.RLock()
	defer mutex.RUnlock()
	return s.base.FindUID(ctx, uid)
}

// Create saves the service account.
func (s *ServiceAccountStoreSync) Create(ctx context.Context, sa *types.ServiceAccount) error {
	mutex.RLock()
	defer mutex.RUnlock()
	return s.base.Create(ctx, sa)
}

// Update updates the service account details.
func (s *ServiceAccountStoreSync) Update(ctx context.Context, sa *types.ServiceAccount) error {
	mutex.RLock()
	defer mutex.RUnlock()
	return s.base.Update(ctx, sa)
}

// Delete deletes the service account.
func (s *ServiceAccountStoreSync) Delete(ctx context.Context, id int64) error {
	mutex.RLock()
	defer mutex.RUnlock()
	return s.base.Delete(ctx, id)
}

// List returns a list of service accounts for a specific parent.
func (s *ServiceAccountStoreSync) List(ctx context.Context, parentType enum.ParentResourceType,
	parentID int64) ([]*types.ServiceAccount, error) {
	mutex.RLock()
	defer mutex.RUnlock()
	return s.base.List(ctx, parentType, parentID)
}

// Count returns a count of service accounts for a specific parent.
func (s *ServiceAccountStoreSync) Count(ctx context.Context, parentType enum.ParentResourceType,
	parentID int64) (int64, error) {
	mutex.RLock()
	defer mutex.RUnlock()
	return s.base.Count(ctx, parentType, parentID)
}
