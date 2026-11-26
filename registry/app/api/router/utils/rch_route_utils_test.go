// Copyright 2026 Harness, Inc.
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

package utils_test

import (
	"testing"

	"github.com/harness/gitness/registry/app/api/router/utils"

	"github.com/stretchr/testify/require"
)

func TestRchGetRouteTypeV2(t *testing.T) {
	tests := []struct {
		name string
		path string
		want utils.RouteType
	}{
		{"manifest by tag", "/v2/root/reg/app/manifests/latest", utils.Manifests},
		{"manifest of a nested image", "/v2/root/reg/team/app/manifests/latest", utils.Manifests},
		{"blob", "/v2/root/reg/app/blobs/sha256:abc", utils.Blobs},
		{"blob upload start", "/v2/root/reg/app/blobs/uploads/", utils.BlobsUploadsSession},
		{"blob upload session", "/v2/root/reg/app/blobs/uploads/session-id", utils.BlobsUploadsSession},
		{"tags", "/v2/root/reg/app/tags/list", utils.Tags},
		{"referrers", "/v2/root/reg/app/referrers/sha256:abc", utils.Referrers},
		{"unknown segment", "/v2/root/reg/app/other/thing", utils.Invalid},
		{"too short", "/v2/root/reg", utils.Invalid},
		{"empty", "", utils.Invalid},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, utils.GetRouteTypeV2(tc.path))
		})
	}
}
