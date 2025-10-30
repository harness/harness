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
	"fmt"
	"net/url"
	"time"

	"github.com/harness/gitness/types"
)

func (c *Service) UpdateInstance(
	ctx context.Context,
	gitspaceInstance *types.GitspaceInstance,
) error {
	gitspaceInstance.Updated = time.Now().UnixMilli()

	if gitspaceInstance.URL != nil {
		formatURLStr, err := formatURL(*gitspaceInstance.URL)
		if err != nil {
			return fmt.Errorf("cannot parse ide url: %w", err)
		}
		gitspaceInstance.URL = &formatURLStr
	}
	err := c.gitspaceInstanceStore.Update(ctx, gitspaceInstance)
	if err != nil {
		return fmt.Errorf("failed to update gitspace instance: %w", err)
	}
	return nil
}

func formatURL(rawURL string) (string, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	// Parse query parameters
	q := parsedURL.Query()

	// Remove token parameter
	q.Del("token")

	// Set updated query back
	parsedURL.RawQuery = q.Encode()

	return parsedURL.String(), nil
}
