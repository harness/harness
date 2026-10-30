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

package gitspace

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/gitspace/scm"
	"github.com/harness/gitness/netpolicy"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

type LookupRepoInput struct {
	SpaceRef string                    `json:"space_ref"` // Ref of the parent space
	URL      string                    `json:"url"`
	RepoType enum.GitspaceCodeRepoType `json:"repo_type"`
}

var (
	ErrInvalidURL = usererror.BadRequest(
		"The URL specified is not valid format.")
	ErrRepoMissing = usererror.BadRequest(
		"There must be URL or Ref specified fir repo.")
	ErrBadURLScheme = usererror.BadRequest("The URL is missing scheme, it must start with http or https")
	// ErrURLNotAllowed is returned for every URL rejected by the network
	// policy, whatever the reason, so the lookup can't be used to tell
	// internal hosts and ports apart.
	ErrURLNotAllowed = usererror.BadRequest("The URL is not allowed.")
)

func (c *Controller) LookupRepo(
	ctx context.Context,
	session *auth.Session,
	in *LookupRepoInput,
) (*scm.CodeRepositoryResponse, error) {
	if err := c.sanitizeLookupRepoInput(in); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}
	space, err := c.spaceFinder.FindByRef(ctx, in.SpaceRef)
	if err != nil {
		return nil, fmt.Errorf("failed to find space: %w", err)
	}
	err = apiauth.CheckInfraProvider(ctx, c.authorizer, session, space.Path,
		"", enum.PermissionInfraProviderView)
	if err != nil {
		return nil, fmt.Errorf("failed to authorize: %w", err)
	}
	if err := c.checkLookupURLAllowed(ctx, in.URL); err != nil {
		return nil, err
	}
	repositoryRequest := scm.CodeRepositoryRequest{
		URL:            in.URL,
		UserIdentifier: session.Principal.UID,
		SpacePath:      space.Path,
		RepoType:       in.RepoType,
		UserID:         session.Principal.ID,
	}
	codeRepositoryResponse, err := c.scm.CheckValidCodeRepo(ctx, repositoryRequest)
	if err != nil {
		return nil, err
	}
	return codeRepositoryResponse, nil
}

func (c *Controller) sanitizeLookupRepoInput(in *LookupRepoInput) error {
	if in.RepoType == "" && in.URL == "" {
		return ErrRepoMissing
	}
	parsedURL, err := url.Parse(in.URL)
	if err != nil {
		return ErrInvalidURL
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return ErrBadURLScheme
	}
	if _, err := url.ParseRequestURI(parsedURL.RequestURI()); err != nil {
		return err
	}
	return nil
}

// checkLookupURLAllowed verifies that the host of the user provided URL only
// resolves to addresses the network policy allows. The lookup runs
// git ls-remote against the URL, so without this check any user with
// infraprovider view permission could make the server connect to internal
// hosts (including cloud metadata endpoints) and probe them through the
// error it returns.
func (c *Controller) checkLookupURLAllowed(ctx context.Context, rawURL string) error {
	err := c.networkPolicy.CheckURLHost(ctx, c.resolver, rawURL)
	if err == nil {
		return nil
	}
	if !errors.Is(err, netpolicy.ErrAddressNotAllowed) {
		// NOTE: a resolution failure is reported like a rejected address, so an
		// internal-only name can't be told apart from a non-existing one.
		log.Ctx(ctx).Debug().Err(err).Str("url", rawURL).Msg("gitspace repo lookup url could not be checked")
	}
	return ErrURLNotAllowed
}
