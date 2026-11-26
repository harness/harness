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

package errcode

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRrmCodeErrorBasics(t *testing.T) {
	code := ErrCodeUnauthorized
	require.Equal(t, code, code.ErrorCode())
	require.Equal(t, "UNAUTHORIZED", code.String())
	require.NotEmpty(t, code.Message())
	require.NotEmpty(t, code.Error())
	desc := code.Descriptor()
	require.Equal(t, http.StatusUnauthorized, desc.HTTPStatusCode)
}

func TestRrmCodeErrorWithMessageAndDetail(t *testing.T) {
	err := ErrCodeDenied.WithMessage("access denied").WithDetail("extra")
	require.Equal(t, ErrCodeDenied, err.Code)
	require.Equal(t, "access denied", err.Message)
	require.Equal(t, "extra", err.Detail)

	withArgs := ErrCodeDenied.WithArgs("user")
	require.Equal(t, ErrCodeDenied, withArgs.Code)
	require.Contains(t, withArgs.Message, "denied")
}

func TestRrmCodeErrorMarshalText(t *testing.T) {
	text, err := ErrCodeUnsupported.MarshalText()
	require.NoError(t, err)
	require.Equal(t, "UNSUPPORTED", string(text))

	var parsed CodeError
	require.NoError(t, parsed.UnmarshalText(text))
	require.Equal(t, ErrCodeUnsupported, parsed)
}

func TestRrmParseErrorCode(t *testing.T) {
	require.Equal(t, ErrCodeDigestInvalid, ParseErrorCode("DIGEST_INVALID"))
	require.Equal(t, ErrCodeUnknown, ParseErrorCode("NOT_A_REAL_CODE"))
}

func TestRrmErrorsMarshalUnmarshal(t *testing.T) {
	src := Errors{ErrCodeUnauthorized, ErrCodeDenied.WithMessage("no way")}
	data, err := json.Marshal(src)
	require.NoError(t, err)

	var dst Errors
	require.NoError(t, json.Unmarshal(data, &dst))
	require.Len(t, dst, 2)
}

func TestRrmErrorsErrorString(t *testing.T) {
	require.Equal(t, "<nil>", Errors(nil).Error())
	require.Equal(t, ErrCodeUnknown.Error(), Errors{ErrCodeUnknown}.Error())
	multi := Errors{ErrCodeUnknown, ErrCodeDenied}
	require.Contains(t, multi.Error(), "errors:")
}

func TestRrmServeJSON(t *testing.T) {
	rr := httptest.NewRecorder()
	require.NoError(t, ServeJSON(rr, ErrCodeUnauthorized))
	require.Equal(t, http.StatusUnauthorized, rr.Code)
	require.Contains(t, rr.Header().Get("Content-Type"), "application/json")
}

func TestRrmServeJSONPlainError(t *testing.T) {
	rr := httptest.NewRecorder()
	require.NoError(t, ServeJSON(rr, fmt.Errorf("plain failure")))
	require.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestRrmGetGroupNames(t *testing.T) {
	names := GetGroupNames()
	require.Contains(t, names, "errcode")
}

func TestRrmGetErrorAllDescriptors(t *testing.T) {
	descs := GetErrorAllDescriptors()
	require.NotEmpty(t, descs)
}
