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

package render

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git/api"
	"github.com/harness/gitness/types"

	"github.com/rs/zerolog/log"
)

// indent the json-encoded API responses.
var indent bool

func init() {
	indent, _ = strconv.ParseBool(
		os.Getenv("HTTP_JSON_INDENT"),
	)
}

// DeleteSuccessful writes the header for a successful delete.
func DeleteSuccessful(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// JSON writes the json-encoded value to the response
// with the provides status.
func JSON(w http.ResponseWriter, code int, v any) {
	setCommonHeaders(w)
	w.WriteHeader(code)
	writeJSON(w, v)
}

// Reader reads the content from the provided reader and writes it as is to the response body.
// NOTE: If no content-type header is added beforehand, the content-type will be deduced
// automatically by `http.DetectContentType` (https://pkg.go.dev/net/http#DetectContentType).
func Reader(ctx context.Context, w http.ResponseWriter, code int, reader io.Reader) {
	w.WriteHeader(code)
	_, err := io.Copy(w, reader)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("failed to render data from reader")
	}
}

// JSONArrayDynamic outputs an JSON array whose elements are streamed from a channel.
// Due to the dynamic nature (unknown number of elements) the function will use
// chunked transfer encoding for large files.
func JSONArrayDynamic[T comparable](ctx context.Context, w http.ResponseWriter, stream types.Stream[T]) {
	count := 0
	enc := json.NewEncoder(w)

	for {
		data, err := stream.Next()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			// based on discussion unrelated histories error should return
			// StatusOK on diff.
			if uErr := api.AsUnrelatedHistoriesError(err); uErr != nil {
				JSON(w, http.StatusOK, &usererror.Error{
					Message: uErr.Error(),
					Values:  uErr.Map(),
				})
				return
			}
			// User canceled the request - no need to do anything
			if errors.Is(err, context.Canceled) {
				return
			}

			if count == 0 {
				// Write the error only if no data has been streamed yet.
				TranslatedUserError(ctx, w, err)
				return
			}

			// Array data has been already streamed, it's too late for the output - so just log and quit.
			log.Ctx(ctx).Warn().Err(err).Msgf("Failed to write JSON array response body")
			// close array
			_, _ = w.Write([]byte{']'})
			return
		}

		if count == 0 {
			setCommonHeaders(w)
			_, _ = w.Write([]byte{'['})
		} else {
			_, _ = w.Write([]byte{','})
		}

		count++

		_ = enc.Encode(data)
	}

	if count == 0 {
		setCommonHeaders(w)
		_, _ = w.Write([]byte{'['})
	}

	_, _ = w.Write([]byte{']'})
}

func Unprocessable(w http.ResponseWriter, v any) {
	JSON(w, http.StatusUnprocessableEntity, v)
}

func Violations(w http.ResponseWriter, violations []types.RuleViolations) {
	Unprocessable(w, types.RulesViolations{
		Message:    protection.GenerateErrorMessageForBlockingViolations(violations),
		Violations: violations,
	})
}

// GitBasicAuth renders a response that indicates that the client (GIT) requires basic authentication.
// This is required in order to tell git CLI to query user credentials.
func GitBasicAuth(ctx context.Context, w http.ResponseWriter, urlProvider url.Provider) {
	// Git doesn't seem to handle "realm" - so it doesn't seem to matter for basic user CLI interactions.
	w.Header().Add("WWW-Authenticate", fmt.Sprintf(`Basic realm="%s"`, urlProvider.GetAPIHostname(ctx)))
	w.WriteHeader(http.StatusUnauthorized)
}

func setCommonHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
}

func writeJSON(w http.ResponseWriter, v any) {
	enc := json.NewEncoder(w)
	if indent {
		enc.SetIndent("", "  ")
	}
	if err := enc.Encode(v); err != nil {
		log.Err(err).Msgf("Failed to write json encoding to response body.")
	}
}
