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

package importer

import (
	"context"
	"fmt"
	"net/url"

	"github.com/harness/gitness/errors"
)

type ConnectorDef struct {
	Path       string `json:"path"`
	Identifier string `json:"identifier"`
	Repo       string `json:"repo"`
}

func ConnectorToURL(
	ctx context.Context,
	s ConnectorService,
	c ConnectorDef,
) (string, error) {
	provider, err := s.AsProvider(ctx, c)
	if err != nil {
		return "", fmt.Errorf("failed to convert linked repo to repository info: %w", err)
	}

	remoteRepository, provider, err := LoadRepositoryFromProvider(ctx, provider, c.Repo)
	if err != nil {
		return "", errors.InvalidArgument("Failed to get access to the remote repository.")
	}

	repoURL, err := url.Parse(remoteRepository.CloneURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse repository clone url, %q: %w",
			remoteRepository.CloneURL, err)
	}

	repoURL.User = url.UserPassword(provider.Username, provider.Password)
	cloneURLWithAuth := repoURL.String()

	return cloneURLWithAuth, nil
}
