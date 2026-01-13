//  Copyright 2023 Harness, Inc.
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.

package storage

import (
	"context"

	"github.com/harness/gitness/registry/app/driver"
)

type DriverSelector struct {
	BucketID     string
	RootParentID int64
	gBlobID      string
}

// DriverProvider interface is for provider storage drivers dynamically.
type DriverProvider interface {
	GetDriver(ctx context.Context, selector DriverSelector) (driver.StorageDriver, error)
	GetDeleteDriver(ctx context.Context, selector DriverSelector) (driver.StorageDeleter, error)
}

// StaticDriverProvider is a simple implementation of StorageDriverProvider
// that always returns the same driver.
type StaticDriverProvider struct {
	driver driver.StorageDriver
}

func NewStaticDriverProvider(d driver.StorageDriver) DriverProvider {
	return &StaticDriverProvider{driver: d}
}

// GetDriver returns the static driver.
func (p *StaticDriverProvider) GetDriver(_ context.Context, _ DriverSelector) (
	driver.StorageDriver,
	error,
) {
	return p.driver, nil
}

func (p *StaticDriverProvider) GetDeleteDriver(_ context.Context, _ DriverSelector) (driver.StorageDeleter, error) {
	return p.driver, nil
}
