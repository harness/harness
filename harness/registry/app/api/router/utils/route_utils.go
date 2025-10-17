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

package utils

import "strings"

type RouteType string

const (
	Manifests           RouteType = "manifests"            // /v2/:registry/:image/manifests/:reference.
	Blobs               RouteType = "blobs"                // /v2/:registry/:image/blobs/:digest.
	BlobsUploadsSession RouteType = "blob-uploads-session" // /v2/:registry/:image/blobs/uploads/:session_id.
	Tags                RouteType = "tags"                 // /v2/:registry/:image/tags/list.
	Referrers           RouteType = "referrers"            // /v2/:registry/:image/referrers/:digest.
	Invalid             RouteType = "invalid"              // Invalid route.
	// Add other route types here.
)

func GetRouteTypeV2(url string) RouteType {
	url = strings.Trim(url, "/")
	segments := strings.Split(url, "/")
	if len(segments) < 4 {
		return Invalid
	}

	typ := segments[len(segments)-2]

	switch typ {
	case "manifests":
		return Manifests
	case "blobs":
		if segments[len(segments)-1] == "uploads" {
			return BlobsUploadsSession
		}
		return Blobs
	case "uploads":
		return BlobsUploadsSession
	case "tags":
		return Tags
	case "referrers":
		return Referrers
	}
	return Invalid
}
