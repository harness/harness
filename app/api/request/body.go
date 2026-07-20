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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// MaxBodySize is the maximum accepted size of a JSON request body decoded by DecodeBody.
const MaxBodySize = 10 << 20 // 10 MiB

// ErrBodyTooLarge is returned by DecodeBody when the request body exceeds MaxBodySize.
var ErrBodyTooLarge = errors.New("http request body too large")

// DecodeBody decodes JSON request body from the provided HTTP request.
// It also discards the rest of the body (up to drainSize) and closes it. This enables
// request canceling. Requests with a body larger than MaxBodySize are rejected with
// ErrBodyTooLarge instead of a misleading decode error.
func DecodeBody(r *http.Request, in any) error {
	// +1 sentinel byte: if the reader gets consumed, the body exceeded MaxBodySize.
	lr := &io.LimitedReader{R: r.Body, N: MaxBodySize + 1}

	defer func() {
		_ = r.Body.Close()
	}()

	err := json.NewDecoder(lr).Decode(in)
	if lr.N == 0 {
		return fmt.Errorf("%w (max %d bytes)", ErrBodyTooLarge, MaxBodySize)
	}
	if err != nil {
		return err
	}

	return nil
}
