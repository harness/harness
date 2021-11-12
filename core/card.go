// Copyright 2019 Drone IO, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package core

import (
	"context"
	"encoding/json"
	"io"
)

type CardInput struct {
	Schema string          `json:"schema"`
	Data   json.RawMessage `json:"data"`
}

// CardStore manages repository cards.
type CardStore interface {
	// Find returns a card data stream from the datastore.
	Find(ctx context.Context, step int64) (io.ReadCloser, error)

	// Create copies the card stream from Reader r to the datastore.
	Create(ctx context.Context, step int64, r io.Reader) error

	// Update copies the card stream from Reader r to the datastore.
	Update(ctx context.Context, step int64, r io.Reader) error

	// Delete purges the card data from the datastore.
	Delete(ctx context.Context, step int64) error
}
