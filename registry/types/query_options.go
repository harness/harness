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

package types

// QueryOptions holds configuration options for database queries.
type QueryOptions struct {
	DeleteFilter DeleteFilter
}

// QueryOption is a function that modifies QueryOptions.
type QueryOption func(o *QueryOptions)

// MakeQueryOptions creates QueryOptions with defaults and applies any provided options.
func MakeQueryOptions(opts ...QueryOption) QueryOptions {
	opt := QueryOptions{
		DeleteFilter: DeleteFilterExcludeDeleted, // Default: exclude soft-deleted entities
	}

	for _, o := range opts {
		o(&opt)
	}

	return opt
}

// WithDeleteFilter sets the soft delete filter option.
func WithDeleteFilter(filter DeleteFilter) QueryOption {
	return func(o *QueryOptions) {
		o.DeleteFilter = filter
	}
}

// WithAllDeleted is a convenience function to include all entities (including soft-deleted).
func WithAllDeleted() QueryOption {
	return WithDeleteFilter(DeleteFilterIncludeDeleted)
}

// WithOnlyDeleted is a convenience function to only include soft-deleted entities.
func WithOnlyDeleted() QueryOption {
	return WithDeleteFilter(DeleteFilterOnlyDeleted)
}

// WithExcludeDeleted is a convenience function to exclude soft-deleted entities (default behavior).
func WithExcludeDeleted() QueryOption {
	return WithDeleteFilter(DeleteFilterExcludeDeleted)
}

// ExtractDeleteFilter extracts the DeleteFilter from QueryOptions.
func ExtractDeleteFilter(opts ...QueryOption) DeleteFilter {
	qo := MakeQueryOptions(opts...)
	return qo.DeleteFilter
}
