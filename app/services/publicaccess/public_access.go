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

package publicaccess

import (
	"context"
	"errors"

	"github.com/harness/gitness/types/enum"
)

var (
	ErrPublicAccessNotAllowed = errors.New("public access is not allowed")
)

// Service is an abstraction of an entity responsible for managing public access to resources.
type Service interface {
	// Get returns whether public access is enabled on the resource.
	Get(
		ctx context.Context,
		resourceType enum.PublicResourceType,
		resourcePath string,
	) (bool, error)

	// Sets the public access mode for the resource based on the value of 'enable'.
	Set(
		ctx context.Context,
		resourceType enum.PublicResourceType,
		resourcePath string,
		enable bool,
	) error

	// Deletes any public access data stored for the resource.
	Delete(
		ctx context.Context,
		resourceType enum.PublicResourceType,
		resourcePath string,
	) error

	// IsPublicAccessSupported return true iff public access is supported under the provided space.
	IsPublicAccessSupported(ctx context.Context, parentSpacePath string) (bool, error)
}
