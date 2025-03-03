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

package pypi

import (
	"context"
	"io"

	"github.com/harness/gitness/registry/app/store"
)

// Controller handles PyPI package operations.
type controller struct {
	artifactStore store.ArtifactRepository
	proxyStore    store.UpstreamProxyConfigRepository
	_             FileManager
}

type Controller interface {
}

// FileManager interface for managing PyPI package files.
type FileManager interface {
	Upload(ctx context.Context, registryID int64, path string, content io.Reader) error
	Download(ctx context.Context, registryID int64, path string) (io.ReadCloser, error)
	Delete(ctx context.Context, registryID int64, path string) error
}

// NewController creates a new PyPI controller.
func NewController(
	artifactStore store.ArtifactRepository,
	proxyStore store.UpstreamProxyConfigRepository,
) Controller {
	return &controller{
		artifactStore: artifactStore,
		proxyStore:    proxyStore,
	}
}
