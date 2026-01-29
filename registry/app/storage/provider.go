//  Copyright 2023 Harness, Inc.
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

package storage

import (
	"context"

	"github.com/harness/gitness/registry/app/driver"
	"github.com/harness/gitness/registry/types"
)

const (
	DefaultBucketKey string = "default"
)

type StorageTarget struct {
	Driver    driver.StorageDriver
	BucketKey string
}

func (t StorageTarget) IsDefault() bool {
	return t.BucketKey == DefaultBucketKey
}

// StorageResolver resolves storage backends based on blob and location.
type StorageResolver interface {
	Resolve(ctx context.Context, lookup types.StorageLookup) (StorageTarget, error)
}

// StaticStorageResolver is a simple implementation of StorageResolver
// that always returns the same driver with the default bucket.
type StaticStorageResolver struct {
	Driver driver.StorageDriver
}

// NewStaticStorageResolver creates a resolver that always returns the given driver.
func NewStaticStorageResolver(d driver.StorageDriver) StorageResolver {
	return &StaticStorageResolver{Driver: d}
}

func (r *StaticStorageResolver) Resolve(_ context.Context, _ types.StorageLookup) (StorageTarget, error) {
	return StorageTarget{
		Driver:    r.Driver,
		BucketKey: DefaultBucketKey,
	}, nil
}
