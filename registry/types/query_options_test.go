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

import (
	"testing"
)

func TestMakeQueryOptions_Default(t *testing.T) {
	opts := MakeQueryOptions()
	if opts.DeleteFilter != DeleteFilterExcludeDeleted {
		t.Errorf("expected default DeleteFilter to be EXCLUDE, got %s", opts.DeleteFilter)
	}
}

func TestMakeQueryOptions_WithDeleteFilter(t *testing.T) {
	opts := MakeQueryOptions(WithDeleteFilter(DeleteFilterOnlyDeleted))
	if opts.DeleteFilter != DeleteFilterOnlyDeleted {
		t.Errorf("expected ONLY, got %s", opts.DeleteFilter)
	}
}

func TestWithAllDeleted(t *testing.T) {
	opts := MakeQueryOptions(WithAllDeleted())
	if opts.DeleteFilter != DeleteFilterIncludeDeleted {
		t.Errorf("expected INCLUDE, got %s", opts.DeleteFilter)
	}
}

func TestWithOnlyDeleted(t *testing.T) {
	opts := MakeQueryOptions(WithOnlyDeleted())
	if opts.DeleteFilter != DeleteFilterOnlyDeleted {
		t.Errorf("expected ONLY, got %s", opts.DeleteFilter)
	}
}

func TestWithExcludeDeleted(t *testing.T) {
	opts := MakeQueryOptions(WithExcludeDeleted())
	if opts.DeleteFilter != DeleteFilterExcludeDeleted {
		t.Errorf("expected EXCLUDE, got %s", opts.DeleteFilter)
	}
}

func TestExtractDeleteFilter_Default(t *testing.T) {
	f := ExtractDeleteFilter()
	if f != DeleteFilterExcludeDeleted {
		t.Errorf("expected EXCLUDE, got %s", f)
	}
}

func TestExtractDeleteFilter_WithOption(t *testing.T) {
	f := ExtractDeleteFilter(WithAllDeleted())
	if f != DeleteFilterIncludeDeleted {
		t.Errorf("expected INCLUDE, got %s", f)
	}
}
