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

package util

import (
	"errors"
	"fmt"
)

var (
	// ErrNotFound is returned when a row is not found on the metadata database.
	ErrNotFound = errors.New("not found")
	// ErrManifestNotFound is returned when a manifest is not found on the metadata database.
	ErrManifestNotFound = fmt.Errorf("manifest %w", ErrNotFound)
	// ErrRefManifestNotFound is returned when a manifest referenced by a list/index is not found on the metadata database.
	ErrRefManifestNotFound = fmt.Errorf("referenced %w", ErrManifestNotFound)
	// ErrManifestReferencedInList is returned when attempting to delete a manifest referenced in at least one list.
	ErrManifestReferencedInList = errors.New("manifest referenced by manifest list")
)

// UnknownMediaTypeError is returned when attempting to save a manifest containing references with unknown media types.
type UnknownMediaTypeError struct {
	// MediaType is the offending media type
	MediaType string
}

// Error implements error.
func (err UnknownMediaTypeError) Error() string {
	return fmt.Sprintf("unknown media type: %s", err.MediaType)
}
