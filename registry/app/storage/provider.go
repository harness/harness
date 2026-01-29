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

type Mode string

const (
	DefaultKey string = "default"
)

type DriverInfo struct {
	Req       types.DriverRequest
	Driver    driver.StorageDriver
	BucketKey string
}

type DriverProvider interface {
	GetDriver(ctx context.Context, selector types.DriverRequest) (DriverInfo, error)
}

func (r DriverInfo) Default() bool {
	return r.BucketKey == DefaultKey
}

// StaticDriverProvider is a simple implementation of StorageDriverProvider
// that always returns the same Driver.
type StaticDriverProvider struct {
	Driver driver.StorageDriver
}

func NewStaticDriverProvider(d driver.StorageDriver) DriverProvider {
	return &StaticDriverProvider{Driver: d}
}

func (p *StaticDriverProvider) GetDriver(_ context.Context, req types.DriverRequest) (DriverInfo, error) {
	return DriverInfo{
		Req:       req,
		Driver:    p.Driver,
		BucketKey: DefaultKey,
	}, nil
}
