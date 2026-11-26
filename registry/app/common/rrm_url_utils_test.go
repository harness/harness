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

package common

import (
	"context"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRrmGenerateOciTokenURL(t *testing.T) {
	require.Equal(t, "https://reg.example/v2/token", GenerateOciTokenURL("https://reg.example"))
}

func TestRrmGenerateSetupClientHostnameAndRegistry(t *testing.T) {
	host, ref := GenerateSetupClientHostnameAndRegistry("https://reg.example/org/proj")
	require.Equal(t, "reg.example", host)
	require.Equal(t, "org/proj", ref)
}

func TestRrmGenerateSetupClientHostnameInvalid(t *testing.T) {
	host, ref := GenerateSetupClientHostnameAndRegistry("://missing-scheme-host")
	require.Empty(t, host)
	require.Empty(t, ref)
}

func TestRrmGetHost(t *testing.T) {
	ctx := context.Background()
	require.Equal(t, "reg.example:443", GetHost(ctx, "https://reg.example:443/path"))
	require.Equal(t, "reg.example", GetHost(ctx, "reg.example"))
	require.Empty(t, GetHost(ctx, "://bad"))
}

func TestRrmGetHostName(t *testing.T) {
	ctx := context.Background()
	require.Equal(t, "reg.example", GetHostName(ctx, "https://reg.example:443/path"))
	require.Equal(t, "reg.example", GetHostName(ctx, "reg.example"))
}

func TestRrmTrimURLScheme(t *testing.T) {
	require.Equal(t, "reg.example/path", TrimURLScheme("https://reg.example/path"))
	require.Equal(t, "not%20a%20url", TrimURLScheme("not a url"))
}

func TestRrmExtractFirstQueryParams(t *testing.T) {
	vals := url.Values{"a": []string{"1", "2"}, "b": []string{"x"}}
	got := ExtractFirstQueryParams(vals)
	require.Equal(t, map[string]string{"a": "1", "b": "x"}, got)
}
