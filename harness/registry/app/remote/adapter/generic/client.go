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

package generic

import (
	"context"
	"net/http"

	"github.com/harness/gitness/app/services/refcache"
	commonhttp "github.com/harness/gitness/registry/app/common/http"
	"github.com/harness/gitness/registry/app/remote/adapter/commons"
	"github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/secret"

	"github.com/rs/zerolog/log"
)

type client struct {
	client   *http.Client
	url      string
	username string
	password string
}

// newClient creates a new Generic client.
func newClient(
	ctx context.Context,
	registry types.UpstreamProxy,
	finder refcache.SpaceFinder,
	service secret.Service,
) (*client, error) {
	accessKey, secretKey, _, err := commons.GetCredentials(ctx, finder, service, registry)
	if err != nil {
		log.Ctx(ctx).Err(err).Msgf("error getting credentials for registry: %s %v", registry.RepoKey, err)
		return nil, err
	}

	c := &client{
		url: registry.RepoURL,
		client: &http.Client{
			Transport: commonhttp.GetHTTPTransport(commonhttp.WithInsecure(true)),
		},
		username: accessKey,
		password: secretKey,
	}

	return c, nil
}
