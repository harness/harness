// Source: https://github.com/distribution/distribution

// Copyright 2014 https://github.com/distribution/distribution Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package errcode

import (
	"encoding/json"
	"net/http"
)

// ServeJSON attempts to serve the errcode in a JSON envelope. It marshals err
// and sets the content-type header to 'application/json'. It will handle
// ErrorCoder and Errors, and if necessary will create an envelope.
func ServeJSON(w http.ResponseWriter, err error) error {
	w.Header().Set("Content-Type", "application/json")
	var sc int

	switch errs := err.(type) {
	case Errors:
		if len(errs) < 1 {
			break
		}

		if err, ok := errs[0].(ErrorCoder); ok {
			sc = err.ErrorCode().Descriptor().HTTPStatusCode
		}
	case ErrorCoder:
		sc = errs.ErrorCode().Descriptor().HTTPStatusCode
		err = Errors{err} // create an envelope.
	default:
		// We just have an unhandled error type, so just place in an envelope
		// and move along.
		err = Errors{err}
	}

	if sc == 0 {
		sc = http.StatusInternalServerError
	}

	w.WriteHeader(sc)

	return json.NewEncoder(w).Encode(err)
}
