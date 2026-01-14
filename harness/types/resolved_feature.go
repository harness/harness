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
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/Masterminds/semver/v3"
)

type ResolvedFeature struct {
	ResolvedOptions   map[string]string  `json:"options,omitempty"`
	OverriddenOptions map[string]any     `json:"overridden_options,omitempty"`
	Digest            string             `json:"string,omitempty"`
	DownloadedFeature *DownloadedFeature `json:"downloaded_feature,omitempty"`
}

type DownloadedFeature struct {
	FeatureFolderName         string                     `json:"feature_folder_name,omitempty"`
	Source                    string                     `json:"source,omitempty"`
	SourceWithoutTag          string                     `json:"source_without_tag,omitempty"`
	Tag                       string                     `json:"tag,omitempty"`
	CanonicalName             string                     `json:"canonical_name,omitempty"`
	DevcontainerFeatureConfig *DevcontainerFeatureConfig `json:"devcontainer_feature_config,omitempty"`
}

func (r *ResolvedFeature) Print() string {
	options := make([]string, 0, len(r.ResolvedOptions))
	for key, value := range r.ResolvedOptions {
		options = append(options, fmt.Sprintf("%s=%s", key, value))
	}
	return fmt.Sprintf("%s %+v", r.DownloadedFeature.Source, options)
}

// CompareResolvedFeature implements the following comparison rules.
// 1. Compare and sort each Feature lexicographically by their fully qualified resource name
// (For OCI-published Features, that means the ID without version or digest.). If the comparison is equal:
// 2. Compare and sort each Feature from oldest to newest tag (latest being the “most new”). If the comparison is equal:
// 3. Compare and sort each Feature by their options by:
// 3.1 Greatest number of user-defined options (note omitting an option will default that value to the Feature’s
// default value and is not considered a user-defined option). If the comparison is equal:
// 3.2 Sort the provided option keys lexicographically. If the comparison is equal:
// 3.3 Sort the provided option values lexicographically. If the comparison is equal:
// 4. Sort Features by their canonical name (For OCI-published Features, the Feature ID resolved to the digest hash).
// 5. If there is no difference based on these comparator rules, the Features are considered equal.
// Reference: https://containers.dev/implementors/features/#definition-feature-equality (Round Stable Sort).
func CompareResolvedFeature(a, b *ResolvedFeature) int {
	var comparison int

	comparison = strings.Compare(a.DownloadedFeature.SourceWithoutTag, b.DownloadedFeature.SourceWithoutTag)
	if comparison != 0 {
		return comparison
	}

	comparison, _ = compareTags(a.DownloadedFeature.Tag, b.DownloadedFeature.Tag)
	if comparison != 0 {
		return comparison
	}

	comparison = compareOverriddenOptions(a.OverriddenOptions, b.OverriddenOptions)
	if comparison != 0 {
		return comparison
	}

	return strings.Compare(a.DownloadedFeature.CanonicalName, b.DownloadedFeature.CanonicalName)
}

func compareTags(a, b string) (int, error) {
	if a == FeatureDefaultTag && b == FeatureDefaultTag {
		return 0, nil
	}
	if a == FeatureDefaultTag {
		return 1, nil
	}
	if b == FeatureDefaultTag {
		return -1, nil
	}
	versionA, err := semver.NewVersion(a)
	if err != nil {
		return 0, err
	}
	versionB, err := semver.NewVersion(b)
	if err != nil {
		return 0, err
	}
	return versionA.Compare(versionB), nil
}

func compareOverriddenOptions(a, b map[string]any) int {
	if len(a) != len(b) {
		return len(a) - len(b)
	}

	keysA, valuesA := getSortedOptions(a)
	keysB, valuesB := getSortedOptions(b)

	for i := 0; i < len(keysA); i++ {
		if keysA[i] == keysB[i] {
			continue
		}
		return strings.Compare(keysA[i], keysB[i])
	}

	for i := 0; i < len(valuesA); i++ {
		if valuesA[i] == valuesB[i] {
			continue
		}
		return strings.Compare(valuesA[i], valuesB[i])
	}

	return 0
}

// getSortedOptions returns the keys and values of a map sorted lexicographically.
func getSortedOptions(m map[string]any) ([]string, []string) {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	values := make([]string, 0, len(m))
	for _, key := range keys {
		value := m[key]
		switch v := value.(type) {
		case string:
			values = append(values, v)
		case bool:
			values = append(values, strconv.FormatBool(v))
		}
	}
	return keys, values
}
