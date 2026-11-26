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

package storage

import (
	"context"
	"testing"

	"github.com/harness/gitness/registry/app/driver/factory"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/harness/gitness/registry/app/driver/filesystem"
)

func TestRsg_FactoryCreateFilesystemDriver(t *testing.T) {
	ctx := context.Background()
	d, err := factory.Create(ctx, "filesystem", map[string]any{
		"rootdirectory": t.TempDir(),
	})
	require.NoError(t, err)
	require.NotNil(t, d)
	assert.Equal(t, "filesystem", d.Name())
}
