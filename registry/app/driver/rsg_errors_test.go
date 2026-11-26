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

package driver

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRsg_DriverErrorMessages(t *testing.T) {
	assert.Equal(t, "fs: unsupported method", UnsupportedMethodError{DriverName: "fs"}.Error())
	assert.Contains(t, PathNotFoundError{DriverName: "fs", Path: "/p"}.Error(), "Path not found")
	assert.Contains(t, InvalidPathError{DriverName: "fs", Path: "/p"}.Error(), "invalid path")
	assert.Contains(t, InvalidOffsetError{DriverName: "fs", Path: "/p", Offset: 1}.Error(), "invalid offset")

	wrapped := Error{DriverName: "fs", Detail: errors.New("detail")}
	assert.Contains(t, wrapped.Error(), "detail")

	data, err := json.Marshal(wrapped)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"driver":"fs"`)
}

func TestRsg_StorageDriverErrorMarshal(t *testing.T) {
	multi := StorageDriverError{DriverName: "fs", Errs: []error{errors.New("a"), errors.New("b")}}
	assert.Contains(t, multi.Error(), "errors:")

	data, err := json.Marshal(multi)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"details"`)

	empty := StorageDriverError{DriverName: "fs"}
	assert.Contains(t, empty.Error(), "<nil>")
	data, err = json.Marshal(empty)
	require.NoError(t, err)
	assert.Contains(t, string(data), `[]`)
}

func TestRsg_FileInfoInternal(t *testing.T) {
	fi := FileInfoInternal{
		FileInfoFields: FileInfoFields{
			Path: "/p", Size: 9, IsDir: false,
		},
	}
	assert.Equal(t, "/p", fi.Path())
	assert.Equal(t, int64(9), fi.Size())
	assert.False(t, fi.IsDir())
}
