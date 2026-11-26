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

package base

import (
	"context"
	"testing"

	"github.com/harness/gitness/registry/app/driver"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRsg_RegulatorProxiesCalls(t *testing.T) {
	ctx := context.Background()
	mem := rsgNewMemDriver()
	reg := NewRegulator(mem, 10)

	assert.Equal(t, "rsg-mem", reg.Name())
	require.NoError(t, reg.PutContent(ctx, "/rsg/reg", []byte("v")))
	got, err := reg.GetContent(ctx, "/rsg/reg")
	require.NoError(t, err)
	assert.Equal(t, []byte("v"), got)

	_, err = reg.Stat(ctx, "/rsg/reg")
	require.NoError(t, err)

	_, err = reg.List(ctx, "/")
	require.NoError(t, err)

	require.NoError(t, reg.Move(ctx, "/rsg/reg", "/rsg/reg2"))
	require.NoError(t, reg.Delete(ctx, "/rsg/reg2"))

	_, err = reg.RedirectURL(ctx, "GET", "/x", "f")
	require.NoError(t, err)

	err = reg.Walk(ctx, "/", func(driver.FileInfo) error { return nil })
	require.NoError(t, err)
}
