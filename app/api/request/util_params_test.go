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

package request

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetCookie(t *testing.T) {
	req := newTestRequest(t, "/")
	req.AddCookie(&http.Cookie{Name: "session", Value: "abc"})

	val, ok := GetCookie(req, "session")
	require.True(t, ok)
	require.Equal(t, "abc", val)

	val, ok = GetCookie(req, "missing")
	require.False(t, ok)
	require.Empty(t, val)
}

func TestHeaders(t *testing.T) {
	req := newTestRequest(t, "/")
	req.Header.Add("X-Multi", "")
	req.Header.Add("X-Multi", "second")
	req.Header.Add("X-Multi", "third")
	req.Header.Set("X-Empty", "")

	val, ok := GetHeader(req, "X-Multi")
	require.True(t, ok)
	require.Equal(t, "second", val, "the first non-empty value wins")

	_, ok = GetHeader(req, "X-Empty")
	require.False(t, ok)
	_, ok = GetHeader(req, "X-Missing")
	require.False(t, ok)

	require.Equal(t, "second", GetHeaderOrDefault(req, "X-Multi", "default"))
	require.Equal(t, "default", GetHeaderOrDefault(req, "X-Empty", "default"))

	req.Header.Set(HeaderContentEncoding, "gzip")
	req.Header.Set(HeaderIfNoneMatch, "etag-1")
	req.Header.Set(HeaderSignature, "sig")
	require.Equal(t, "gzip", GetContentEncodingFromHeadersOrDefault(req, "identity"))
	require.Equal(t, "identity", GetContentEncodingFromHeadersOrDefault(newTestRequest(t, "/"), "identity"))
	etag, ok := GetIfNoneMatchFromHeader(req)
	require.True(t, ok)
	require.Equal(t, "etag-1", etag)
	require.Equal(t, "sig", GetSignatureFromHeaderOrDefault(req, "none"))
	require.Equal(t, "none", GetSignatureFromHeaderOrDefault(newTestRequest(t, "/"), "none"))
}

func TestPathParams(t *testing.T) {
	req := newTestRequest(t, "/",
		"name", "my%2Frepo", "empty", "", "broken", "%zz", "num", "12", "neg", "-3", "text", "abc")

	val, err := PathParam(req, "name")
	require.NoError(t, err)
	require.Equal(t, "my/repo", val, "values are unescaped")

	val, err = PathParam(req, "empty")
	require.NoError(t, err)
	require.Empty(t, val)

	_, err = PathParam(req, "broken")
	requireBadRequest(t, err, "Failed to decode path parameter 'broken'")

	val, err = PathParamOrError(req, "name")
	require.NoError(t, err)
	require.Equal(t, "my/repo", val)
	_, err = PathParamOrError(req, "empty")
	requireBadRequest(t, err, "'empty' not found")
	_, err = PathParamOrError(req, "broken")
	requireBadRequest(t, err, "Failed to decode")

	require.Equal(t, "my/repo", PathParamOrEmpty(req, "name"))
	require.Empty(t, PathParamOrEmpty(req, "broken"))
	require.Empty(t, PathParamOrEmpty(req, "missing"))

	n, err := PathParamAsPositiveInt64(req, "num")
	require.NoError(t, err)
	require.Equal(t, int64(12), n)
	for _, name := range []string{"neg", "text"} {
		_, err = PathParamAsPositiveInt64(req, name)
		requireBadRequest(t, err, "must be a positive integer")
	}
	_, err = PathParamAsPositiveInt64(req, "missing")
	requireBadRequest(t, err, "not found")
}

func TestQueryParams(t *testing.T) {
	req := newTestRequest(t, "/?a=1&b=&list=x&list=y&n=7&zero=0&neg=-1&text=abc&flag=true&bad=maybe")

	val, ok := QueryParam(req, "a")
	require.True(t, ok)
	require.Equal(t, "1", val)
	val, ok = QueryParam(req, "b")
	require.True(t, ok, "empty values still count as present")
	require.Empty(t, val)
	_, ok = QueryParam(req, "missing")
	require.False(t, ok)

	list, ok := QueryParamList(req, "list")
	require.True(t, ok)
	require.Equal(t, []string{"x", "y"}, list)
	_, ok = QueryParamList(req, "missing")
	require.False(t, ok)

	require.Equal(t, "1", QueryParamOrDefault(req, "a", "d"))
	require.Equal(t, "d", QueryParamOrDefault(req, "missing", "d"))

	val, err := QueryParamOrError(req, "a")
	require.NoError(t, err)
	require.Equal(t, "1", val)
	_, err = QueryParamOrError(req, "missing")
	requireBadRequest(t, err, "'missing' not found in query")
}

func TestQueryParamAsPositiveInt64(t *testing.T) {
	req := newTestRequest(t, "/?n=7&zero=0&neg=-1&text=abc")

	n, err := QueryParamAsPositiveInt64OrDefault(req, "n", 99)
	require.NoError(t, err)
	require.Equal(t, int64(7), n)
	n, err = QueryParamAsPositiveInt64OrDefault(req, "missing", 99)
	require.NoError(t, err)
	require.Equal(t, int64(99), n)
	for _, name := range []string{"zero", "neg", "text"} {
		_, err = QueryParamAsPositiveInt64OrDefault(req, name, 99)
		requireBadRequest(t, err, "must be a positive integer")
	}

	n, err = QueryParamAsPositiveInt64OrError(req, "n")
	require.NoError(t, err)
	require.Equal(t, int64(7), n)
	_, err = QueryParamAsPositiveInt64OrError(req, "missing")
	requireBadRequest(t, err, "not found")
	_, err = QueryParamAsPositiveInt64OrError(req, "zero")
	requireBadRequest(t, err, "must be a positive integer")

	n, ok, err := QueryParamAsPositiveInt64(req, "n")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, int64(7), n)
	_, ok, err = QueryParamAsPositiveInt64(req, "missing")
	require.NoError(t, err)
	require.False(t, ok)
	_, _, err = QueryParamAsPositiveInt64(req, "neg")
	requireBadRequest(t, err, "must be a positive integer")

	require.Equal(t, int64(7), QueryParamAsUnixMillisOrDefault(req, "n", 1))
	require.Equal(t, int64(1), QueryParamAsUnixMillisOrDefault(req, "missing", 1))
	require.Equal(t, int64(1), QueryParamAsUnixMillisOrDefault(req, "text", 1), "invalid values fall back")
}

func TestQueryParamAsBoolAndList(t *testing.T) {
	req := newTestRequest(t, "/?flag=true&off=false&empty=&bad=maybe&ids=1&ids=2&badids=1&badids=x&zeroids=0")

	got, err := QueryParamAsBoolOrDefault(req, "flag", false)
	require.NoError(t, err)
	require.True(t, got)
	got, err = QueryParamAsBoolOrDefault(req, "off", true)
	require.NoError(t, err)
	require.False(t, got)
	got, err = QueryParamAsBoolOrDefault(req, "empty", true)
	require.NoError(t, err)
	require.True(t, got, "an empty value uses the default")
	got, err = QueryParamAsBoolOrDefault(req, "missing", true)
	require.NoError(t, err)
	require.True(t, got)
	_, err = QueryParamAsBoolOrDefault(req, "bad", false)
	requireBadRequest(t, err, "must be a boolean")

	ids, err := QueryParamListAsPositiveInt64(req, "ids")
	require.NoError(t, err)
	require.Equal(t, []int64{1, 2}, ids)
	ids, err = QueryParamListAsPositiveInt64(req, "missing")
	require.NoError(t, err)
	require.NotNil(t, ids)
	require.Empty(t, ids)
	for _, name := range []string{"badids", "zeroids"} {
		_, err = QueryParamListAsPositiveInt64(req, name)
		requireBadRequest(t, err, "must be a positive integer")
	}
}
