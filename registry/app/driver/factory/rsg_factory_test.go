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

package factory

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/harness/gitness/registry/app/driver"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type rsgStubFactory struct{}

var errRsgStubFactory = errors.New("rsg stub factory")

func (rsgStubFactory) Create(_ context.Context, _ map[string]any) (driver.StorageDriver, error) {
	return nil, errRsgStubFactory
}

func TestRsg_FactoryInvalidDriver(t *testing.T) {
	_, err := Create(context.Background(), "rsg-no-such-driver", nil)
	require.Error(t, err)
	var inv InvalidStorageDriverError
	require.ErrorAs(t, err, &inv)
	assert.Contains(t, inv.Error(), "rsg-no-such-driver")
}

func TestRsg_FactoryRegister(t *testing.T) {
	name := fmt.Sprintf("rsg-factory-%d", time.Now().UnixNano())
	Register(name, rsgStubFactory{})
	_, err := Create(context.Background(), name, nil)
	require.ErrorIs(t, err, errRsgStubFactory)
}
