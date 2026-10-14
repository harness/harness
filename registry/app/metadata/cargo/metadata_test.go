//  Copyright 2026 Harness, Inc.
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

package cargo

import (
	"encoding/json"
	"testing"
)

// Cargo reads a missing `default_features` of an index dependency as true, so a dependency declared with
// `default-features = false` must keep the explicit `"default_features":false` in the index.
func TestIndexDependencyKeepsDefaultFeaturesFalse(t *testing.T) {
	index := IndexMetadata{
		Name:     "consumer",
		Version:  "0.1.0",
		Checksum: "abc",
		Features: map[string][]string{},
		Dependencies: []IndexDependency{
			{Dependency: Dependency{Name: "lean", DefaultFeatures: false}, VersionRequired: "^1"},
			{Dependency: Dependency{Name: "full", DefaultFeatures: true}, VersionRequired: "^1"},
		},
	}

	raw, err := json.Marshal(index)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got struct {
		Deps []map[string]any `json:"deps"`
	}
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	want := map[string]bool{"lean": false, "full": true}
	if len(got.Deps) != len(want) {
		t.Fatalf("got %d deps, want %d: %s", len(got.Deps), len(want), raw)
	}
	for _, dep := range got.Deps {
		name, _ := dep["name"].(string)
		value, present := dep["default_features"]
		if !present {
			t.Errorf("dependency %q: default_features is missing from the index line %s", name, raw)
			continue
		}
		if value != want[name] {
			t.Errorf("dependency %q: default_features = %v, want %v", name, value, want[name])
		}
	}
}

// The metadata of a published version is stored as JSON and the index is regenerated from it, so the value must
// survive the round trip, and a row stored before the fix (no field for `false`) must still read as false.
func TestVersionDependencyDefaultFeaturesRoundTrip(t *testing.T) {
	published := []byte(`{"name":"c","vers":"0.1.0","deps":[` +
		`{"name":"lean","version_req":"^1","default_features":false},` +
		`{"name":"full","version_req":"^1","default_features":true}]}`)

	var fromPublish VersionMetadata
	if err := json.Unmarshal(published, &fromPublish); err != nil {
		t.Fatalf("unmarshal publish payload: %v", err)
	}
	stored, err := json.Marshal(fromPublish)
	if err != nil {
		t.Fatalf("marshal stored metadata: %v", err)
	}
	var fromDB VersionMetadata
	if err := json.Unmarshal(stored, &fromDB); err != nil {
		t.Fatalf("unmarshal stored metadata: %v", err)
	}
	if fromDB.Dependencies[0].DefaultFeatures || !fromDB.Dependencies[1].DefaultFeatures {
		t.Errorf("round trip changed default_features: %+v", fromDB.Dependencies)
	}

	legacy := []byte(`{"name":"c","vers":"0.1.0","deps":[{"name":"lean","version_req":"^1"},` +
		`{"name":"full","version_req":"^1","default_features":true}]}`)
	var fromLegacy VersionMetadata
	if err := json.Unmarshal(legacy, &fromLegacy); err != nil {
		t.Fatalf("unmarshal legacy metadata: %v", err)
	}
	if fromLegacy.Dependencies[0].DefaultFeatures || !fromLegacy.Dependencies[1].DefaultFeatures {
		t.Errorf("legacy row read wrongly: %+v", fromLegacy.Dependencies)
	}
}
