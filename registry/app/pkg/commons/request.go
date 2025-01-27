//  Copyright 2023 Harness, Inc.
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

package commons

import (
	"net/http"
	"reflect"

	"github.com/harness/gitness/registry/app/dist_temp/errcode"
)

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
)

type ResponseHeaders struct {
	Headers map[string]string
	Code    int
}

func IsEmpty(slice interface{}) bool {
	if slice == nil {
		return true
	}
	val := reflect.ValueOf(slice)

	// Check if the input is a pointer
	if val.Kind() == reflect.Ptr {
		// Dereference the pointer
		val = val.Elem()
	}

	// Check if the dereferenced value is nil
	if !val.IsValid() {
		return true
	}

	return val.Len() == 0
}

func IsEmptyError(err errcode.Error) bool {
	return err.Code == 0
}

func (r *ResponseHeaders) WriteToResponse(w http.ResponseWriter) {
	if w == nil || r == nil {
		return
	}

	if r.Headers != nil {
		for key, value := range r.Headers {
			w.Header().Set(key, value)
		}
	}

	if r.Code != 0 {
		w.WriteHeader(r.Code)
	}
}

func (r *ResponseHeaders) WriteHeadersToResponse(w http.ResponseWriter) {
	if w == nil || r == nil {
		return
	}

	if r.Headers != nil {
		for key, value := range r.Headers {
			w.Header().Set(key, value)
		}
	}
}
