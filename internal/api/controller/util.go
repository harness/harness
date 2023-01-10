// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package controller

import (
	"context"
	"fmt"

	"github.com/harness/gitness/gitrpc"
	"github.com/harness/gitness/internal/api/request"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/internal/githook"
	"github.com/harness/gitness/internal/url"
	"github.com/harness/gitness/types"

	"github.com/rs/zerolog/log"
)

// CreateRPCWriteParams creates base write parameters for gitrpc write operations.
// IMPORTANT: session & repo are assumed to be not nil!
// TODO: this is duplicate function from repo controller, we need to see where this
// function will be best fit.
func CreateRPCWriteParams(ctx context.Context, urlProvider *url.Provider,
	session *auth.Session, repo *types.Repository) (gitrpc.WriteParams, error) {
	requestID, ok := request.RequestIDFrom(ctx)
	if !ok {
		// best effort retrieving of requestID - log in case we can't find it but don't fail operation.
		log.Ctx(ctx).Warn().Msg("operation doesn't have a requestID in the context.")
	}

	// generate envars (add everything githook CLI needs for execution)
	envVars, err := githook.GenerateEnvironmentVariables(&githook.Payload{
		APIBaseURL:  urlProvider.GetAPIBaseURLInternal(),
		RepoID:      repo.ID,
		PrincipalID: session.Principal.ID,
		RequestID:   requestID,
	})
	if err != nil {
		return gitrpc.WriteParams{}, fmt.Errorf("failed to generate git hook environment variables: %w", err)
	}

	return gitrpc.WriteParams{
		Actor: gitrpc.Identity{
			Name:  session.Principal.DisplayName,
			Email: session.Principal.Email,
		},
		RepoUID: repo.GitUID,
		EnvVars: envVars,
	}, nil
}
