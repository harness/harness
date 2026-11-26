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
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRrmNotFoundAndUnknown(t *testing.T) {
	root := fmt.Errorf("db miss")
	nf := NotFoundError(root)
	require.Contains(t, nf.Error(), "resource not found")
	require.True(t, IsNotFoundErr(nf))
	require.Equal(t, NotFoundCode, ErrCode(nf))

	unk := UnknownError(root)
	require.Equal(t, GeneralCode, unk.Code)
}

func TestRrmIsRateLimitError(t *testing.T) {
	err := New("slow down").WithCode(RateLimitCode)
	require.True(t, IsRateLimitError(err))
	require.False(t, IsRateLimitError(fmt.Errorf("other")))
}

func TestRrmErrorWrapAndUnwrap(t *testing.T) {
	root := fmt.Errorf("root cause")
	wrapped := Wrap(root, "outer").WithCode(BadRequestCode)
	require.Equal(t, root, wrapped.Unwrap())
	require.Contains(t, wrapped.Error(), "outer")
	require.Equal(t, BadRequestCode, wrapped.Code)
}

func TestRrmErrorWrapNil(t *testing.T) {
	require.Nil(t, Wrap(nil, "nothing"))
	require.Nil(t, Wrapf(nil, "nothing %s", "x"))
}

func TestRrmErrorNewAndWith(t *testing.T) {
	e := New("hello").WithCode(ConflictCode).WithMessage("custom %s", "msg")
	require.Equal(t, ConflictCode, e.Code)
	require.Equal(t, "custom msg", e.Message)

	caused := e.WithCause(fmt.Errorf("cause"))
	require.NotNil(t, caused.Cause)
}

func TestRrmErrorMarshalJSON(t *testing.T) {
	e := New("msg").WithCode(ForbiddenCode)
	data, err := json.Marshal(e)
	require.NoError(t, err)
	require.Contains(t, string(data), ForbiddenCode)
}

func TestRrmErrorsCollection(t *testing.T) {
	coll := NewErrs(New("one").WithCode(GeneralCode))
	require.Equal(t, 1, coll.Len())
	out := coll.Error()
	require.Contains(t, out, "one")

	coll = Errors{fmt.Errorf("plain")}
	out = coll.Error()
	require.Contains(t, out, "UNKNOWN")
}

func TestRrmIsErrAndErrCode(t *testing.T) {
	e := Errorf("fmt err").WithCode(DENIED)
	require.True(t, IsErr(e, DENIED))
	require.False(t, IsErr(e, NotFoundCode))
	require.Equal(t, DENIED, ErrCode(e))
	require.Empty(t, ErrCode(nil))
}

func TestRrmErrorAsPlainError(t *testing.T) {
	require.False(t, IsErr(fmt.Errorf("plain"), NotFoundCode))
	require.Equal(t, GeneralCode, ErrCode(fmt.Errorf("plain")))
}

func TestRrmErrorsIsAlias(t *testing.T) {
	target := New("x").WithCode(GeneralCode)
	wrapped := fmt.Errorf("wrap: %w", target)
	var herr *Error
	require.True(t, errors.As(wrapped, &herr))
	require.True(t, Is(wrapped, target))
}
