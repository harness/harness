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
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"time"

	"github.com/harness/gitness/registry/app/dist_temp/errcode"
	"github.com/harness/gitness/registry/app/storage"
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

	kind := val.Kind()

	if kind == reflect.Slice || kind == reflect.Array || kind == reflect.Map || kind == reflect.String ||
		kind == reflect.Chan || kind == reflect.Ptr {
		return val.Len() == 0
	}
	return false
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

func ServeContent(
	w http.ResponseWriter,
	r *http.Request,
	body *storage.FileReader,
	fileName string,
	readCloser io.ReadCloser,
) error {
	if body != nil {
		http.ServeContent(w, r, fileName, time.Time{}, body)
		return nil
	}
	if readCloser != nil {
		_, err := io.Copy(w, readCloser)
		if err != nil {
			return fmt.Errorf("failed to copy content: %w", err)
		}
		return nil
	}
	return errors.New("no content to serve")
}
