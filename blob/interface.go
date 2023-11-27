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

package blob

import (
	"context"
	"errors"
	"io"
)

var (
	ErrNotFound     = errors.New("resource not found")
	ErrNotSupported = errors.New("not supported")
)

type Store interface {
	// Upload uploads a file to the blob store.
	Upload(ctx context.Context, file io.Reader, filePath string) error

	// GetSignedURL returns the URL for a file in the blob store.
	GetSignedURL(ctx context.Context, filePath string) (string, error)

	// Download returns a reader for a file in the blob store.
	Download(ctx context.Context, filePath string) (io.ReadCloser, error)
}
