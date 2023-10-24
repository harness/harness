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

package githook

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/githook"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/version"

	"github.com/rs/zerolog/log"
)

var (
	// ExecutionTimeout is the timeout used for githook CLI runs.
	ExecutionTimeout = 3 * time.Minute
)

// GenerateEnvironmentVariables generates the required environment variables for a payload
// constructed from the provided parameters.
// The parameter `internal` should be true if the call is coming from the Gitness
// and therefore protection from rules shouldn't be verified.
func GenerateEnvironmentVariables(
	ctx context.Context,
	apiBaseURL string,
	repoID int64,
	principalID int64,
	disabled bool,
	internal bool,
) (map[string]string, error) {
	// best effort retrieving of requestID - log in case we can't find it but don't fail operation.
	requestID, ok := request.RequestIDFrom(ctx)
	if !ok {
		log.Ctx(ctx).Warn().Msg("operation doesn't have a requestID in the context - generate githook payload without")
	}

	// generate githook base url
	baseURL := strings.TrimLeft(apiBaseURL, "/") + "/v1/internal/git-hooks"

	payload := &types.GithookPayload{
		BaseURL:     baseURL,
		RepoID:      repoID,
		PrincipalID: principalID,
		RequestID:   requestID,
		Disabled:    disabled,
		Internal:    internal,
	}

	if err := payload.Validate(); err != nil {
		return nil, fmt.Errorf("generated payload is invalid: %w", err)
	}

	return githook.GenerateEnvironmentVariables(payload)
}

// LoadFromEnvironment returns a new githook.CLICore created by loading the payload from the environment variable.
func LoadFromEnvironment() (*githook.CLICore, error) {
	payload, err := githook.LoadPayloadFromEnvironment[*types.GithookPayload]()
	if err != nil {
		return nil, fmt.Errorf("failed to load payload from environment: %w", err)
	}

	// ensure we return disabled error in case it's explicitly disabled (will result in no-op)
	if payload.Disabled {
		return nil, githook.ErrDisabled
	}

	if err := payload.Validate(); err != nil {
		return nil, fmt.Errorf("payload validation failed: %w", err)
	}

	return githook.NewCLICore(
		githook.NewClient(
			http.DefaultClient,
			payload.BaseURL,
			func(r *http.Request) *http.Request {
				// add query params
				query := r.URL.Query()
				query.Add(request.QueryParamRepoID, fmt.Sprint(payload.RepoID))
				query.Add(request.QueryParamPrincipalID, fmt.Sprint(payload.PrincipalID))
				if payload.Internal {
					query.Add(request.QueryParamInternal, "true")
				}
				r.URL.RawQuery = query.Encode()

				// add headers
				if len(payload.RequestID) > 0 {
					r.Header.Add(request.HeaderRequestID, payload.RequestID)
				}
				r.Header.Add(request.HeaderUserAgent, fmt.Sprintf("Gitness/%s", version.Version))

				return r
			},
		),
		ExecutionTimeout,
	), nil
}
