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

package store

import (
	"context"
	"io"
)

// LogStore provides an interface for the persistent log store backend.
type LogStore interface {
	// Find returns a log stream from the datastore.
	Find(ctx context.Context, stepID int64) (io.ReadCloser, error)

	// Create writes copies the log stream from Reader r to the datastore.
	Create(ctx context.Context, stepID int64, r io.Reader) error

	// Update copies the log stream from Reader r to the datastore.
	Update(ctx context.Context, stepID int64, r io.Reader) error

	// Delete purges the log stream from the datastore.
	Delete(ctx context.Context, stepID int64) error
}
