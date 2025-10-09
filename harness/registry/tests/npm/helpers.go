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

package npmconformance

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/harness/gitness/registry/app/metadata/npm"
)

// TestScope is the default scope used for scoped package testing.
const TestScope = "testscope"

// generateNpmPackagePayload creates a mock NPM package payload for testing.
func generateNpmPackagePayload(packageName, version string, isScoped bool, scope string) string {
	var fullName string
	if isScoped {
		fullName = fmt.Sprintf("@%s/%s", scope, packageName)
	} else {
		fullName = packageName
	}

	// Create mock tarball content
	tarballContent := fmt.Sprintf(`{"name":"%s","version":"%s","main":"index.js"}`, fullName, version)
	encodedTarball := base64.StdEncoding.EncodeToString([]byte(tarballContent))

	filename := fmt.Sprintf("%s-%s.tgz", packageName, version)
	if isScoped {
		filename = fmt.Sprintf("%s-%s.tgz", packageName, version)
	}

	payload := npm.PackageUpload{
		PackageMetadata: npm.PackageMetadata{
			ID:          fullName,
			Name:        fullName,
			Description: "Test package for conformance testing",
			DistTags: map[string]string{
				"latest": version,
			},
			Versions: map[string]*npm.PackageMetadataVersion{
				version: {
					Name:        fullName,
					Version:     version,
					Description: "Test package for conformance testing",
					Keywords:    []string{"test", "conformance"},
					Author:      "Harness Test Suite",
					License:     "MIT",
					Dist: npm.PackageDistribution{
						Integrity: "sha512-test",
						Shasum:    "testsha",
						Tarball:   fmt.Sprintf("http://localhost/%s/-/%s", fullName, filename),
					},
				},
			},
		},
		Attachments: map[string]*npm.PackageAttachment{
			filename: {
				ContentType: "application/octet-stream",
				Data:        encodedTarball,
				Length:      len(tarballContent),
			},
		},
	}

	jsonData, _ := json.Marshal(payload)
	return string(jsonData)
}
