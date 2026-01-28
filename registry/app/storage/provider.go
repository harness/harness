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

	"github.com/google/uuid"
)

type Mode string

const (
	ModeWrite  Mode   = "WRITE"
	ModeRead   Mode   = "READ"
	DefaultKey string = "default"
)

type DriverSelector struct {
	BucketID      uuid.UUID
	RootParentID  int64
	BlobID        int64
	GenericBlobID uuid.UUID
	GlobalBlobID  uuid.UUID
	Mode          Mode
}

// DriverResult is an extensible interface for GetDriver responses.
// Implementers can embed BaseDriverResult and add custom fields/methods.
type DriverResult interface {
	GetDriver() driver.StorageDriver
	GetBucketKey() string
	IsDefault() bool
}

// BaseDriverResult provides a default implementation of DriverResult.
type BaseDriverResult struct {
	Driver driver.StorageDriver
	Key    string
}

func (r *BaseDriverResult) GetDriver() driver.StorageDriver {
	return r.Driver
}

func (r *BaseDriverResult) GetBucketKey() string {
	return r.Key
}

func (r *BaseDriverResult) IsDefault() bool {
	return r.Key == DefaultKey
}

// DriverProvider interface is for provider storage drivers dynamically.
type DriverProvider interface {
	GetDriver(ctx context.Context, selector DriverSelector) (DriverResult, error)
}

// StaticDriverProvider is a simple implementation of StorageDriverProvider
// that always returns the same Driver.
type StaticDriverProvider struct {
	Driver driver.StorageDriver
}

func NewStaticDriverProvider(d driver.StorageDriver) DriverProvider {
	return &StaticDriverProvider{Driver: d}
}

// GetDriver returns the static Driver.
func (p *StaticDriverProvider) GetDriver(_ context.Context, _ DriverSelector) (DriverResult, error) {
	return &BaseDriverResult{
		Driver: p.Driver,
		Key:    "random",
	}, nil
}
