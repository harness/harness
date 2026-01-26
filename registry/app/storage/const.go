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

import "time"

const (
	HeaderAccept              = "Accept"
	HeaderAuthorization       = "Authorization"
	HeaderCacheControl        = "Cache-Control"
	HeaderContentLength       = "Content-Length"
	HeaderContentRange        = "Content-Range"
	HeaderContentType         = "Content-Type"
	HeaderDockerContentDigest = "Docker-Content-Digest"
	HeaderDockerUploadUUID    = "Docker-Upload-UUID"
	HeaderEtag                = "Etag"
	HeaderIfNoneMatch         = "If-None-Match"
	HeaderLink                = "Link"
	HeaderLocation            = "Location"
	HeaderOCIFiltersApplied   = "OCI-Filters-Applied"
	HeaderOCISubject          = "OCI-Subject"
	HeaderRange               = "Range"
	blobCacheControlMaxAge    = 365 * 24 * time.Hour
)
