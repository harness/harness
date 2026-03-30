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

package util

import (
	"testing"
)

type baseDB struct {
	ID   int64  `db:"id"`
	Name string `db:"name"`
}

type extendedDB struct {
	baseDB
	Extra string `db:"extra"`
}

type noTags struct {
	Foo string
}

func TestGetDBTagsFromStruct_Simple(t *testing.T) {
	tags := GetDBTagsFromStruct(baseDB{})
	if len(tags) != 2 {
		t.Fatalf("expected 2 tags, got %d: %v", len(tags), tags)
	}
	if tags[0] != "id" || tags[1] != "name" {
		t.Errorf("unexpected tags: %v", tags)
	}
}

func TestGetDBTagsFromStruct_Embedded(t *testing.T) {
	tags := GetDBTagsFromStruct(extendedDB{})
	if len(tags) != 3 {
		t.Fatalf("expected 3 tags, got %d: %v", len(tags), tags)
	}
	expected := map[string]bool{"id": true, "name": true, "extra": true}
	for _, tag := range tags {
		if !expected[tag] {
			t.Errorf("unexpected tag: %s", tag)
		}
	}
}

func TestGetDBTagsFromStruct_NoTags(t *testing.T) {
	tags := GetDBTagsFromStruct(noTags{})
	if len(tags) != 0 {
		t.Errorf("expected 0 tags, got %d: %v", len(tags), tags)
	}
}
