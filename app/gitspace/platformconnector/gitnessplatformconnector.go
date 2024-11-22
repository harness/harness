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

package platformconnector

import (
	"context"

	"github.com/harness/gitness/types"
)

var _ PlatformConnector = (*GitnessPlatformConnector)(nil)

type GitnessPlatformConnector struct{}

func NewGitnessPlatformConnector() *GitnessPlatformConnector {
	return &GitnessPlatformConnector{}
}

func (g *GitnessPlatformConnector) FetchConnectors(
	_ context.Context,
	ids []string,
) ([]types.PlatformConnector, error) {
	result := make([]types.PlatformConnector, len(ids))
	for i, id := range ids {
		result[i] = types.PlatformConnector{ID: id}
	}
	return result, nil
}
