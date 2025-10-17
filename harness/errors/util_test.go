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

package errors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsType(t *testing.T) {
	err := errors.New("abc")
	valueErr := testValueError{}
	valueErrPtr := &testValueError{}
	pointerErr := &testPointerError{}

	assert.True(t, IsType[error](err))
	assert.True(t, IsType[error](valueErr))
	assert.True(t, IsType[error](valueErrPtr))
	assert.True(t, IsType[error](pointerErr))

	assert.False(t, IsType[testValueError](err))
	assert.True(t, IsType[testValueError](valueErr))
	assert.False(t, IsType[testValueError](valueErrPtr))
	assert.False(t, IsType[testValueError](pointerErr))

	assert.False(t, IsType[*testValueError](err))
	assert.False(t, IsType[*testValueError](valueErr))
	assert.True(t, IsType[*testValueError](valueErrPtr))
	assert.False(t, IsType[*testValueError](pointerErr))

	assert.False(t, IsType[*testPointerError](err))
	assert.False(t, IsType[*testPointerError](valueErr))
	assert.False(t, IsType[*testPointerError](valueErrPtr))
	assert.True(t, IsType[*testPointerError](pointerErr))
}

type testValueError struct{}

func (e testValueError) Error() string { return "value receiver" }

type testPointerError struct{}

func (e *testPointerError) Error() string { return "pointer receiver" }
