// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package githook

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/harness/gitness/githook"
	"github.com/harness/gitness/internal/api/request"
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
func GenerateEnvironmentVariables(
	ctx context.Context,
	apiBaseURL string,
	repoID int64,
	principalID int64,
	disabled bool,
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
