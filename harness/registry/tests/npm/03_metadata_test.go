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
	"encoding/json"
	"fmt"

	"github.com/harness/gitness/registry/app/metadata/npm"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var test03Metadata = func() {
	ginkgo.Context("Metadata", func() {
		ginkgo.BeforeEach(func() {
			SkipIfDisabled(TestMetadata)
		})

		ginkgo.It("should retrieve package metadata for non-scoped package", func() {
			packageName := GetUniquePackageName("metadata", 1)
			version := GetUniqueVersion(1)

			// First upload the package
			packageData := generateNpmPackagePayload(packageName, version, false, "")
			uploadReq := client.NewRequest(
				"PUT",
				fmt.Sprintf("/pkg/%s/%s/npm/%s/",
					TestConfig.Namespace, TestConfig.RegistryName, packageName,
				),
			)
			uploadReq.SetHeader("Content-Type", "application/json")
			uploadReq.SetBody(packageData)

			uploadResp, err := client.Do(uploadReq)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(uploadResp.StatusCode).To(gomega.Equal(200))

			// Now retrieve metadata
			metadataReq := client.NewRequest(
				"GET",
				fmt.Sprintf("/pkg/%s/%s/npm/%s/",
					TestConfig.Namespace, TestConfig.RegistryName, packageName,
				),
			)

			metadataResp, err := client.Do(metadataReq)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(metadataResp.StatusCode).To(gomega.Equal(200))

			// Verify metadata structure
			var metadata npm.PackageMetadata
			err = json.Unmarshal(metadataResp.Body, &metadata)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(metadata.Name).To(gomega.Equal(packageName))
			gomega.Expect(metadata.Versions).To(gomega.HaveKey(version))
			gomega.Expect(metadata.DistTags).To(gomega.HaveKeyWithValue("latest", version))
		})

		ginkgo.It("should retrieve package metadata for scoped package", func() {
			scope := TestScope
			packageName := "test-scoped-metadata"
			version := GetUniqueVersion(2)
			fullName := fmt.Sprintf("@%s/%s", scope, packageName)

			// First upload the scoped package
			packageData := generateNpmPackagePayload(packageName, version, true, scope)
			uploadReq := client.NewRequest(
				"PUT",
				fmt.Sprintf("/pkg/%s/%s/npm/@%s/%s/",
					TestConfig.Namespace, TestConfig.RegistryName, scope, packageName,
				),
			)
			uploadReq.SetHeader("Content-Type", "application/json")
			uploadReq.SetBody(packageData)

			uploadResp, err := client.Do(uploadReq)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(uploadResp.StatusCode).To(gomega.Equal(200))

			// Now retrieve metadata
			metadataReq := client.NewRequest(
				"GET",
				fmt.Sprintf("/pkg/%s/%s/npm/@%s/%s/",
					TestConfig.Namespace, TestConfig.RegistryName, scope, packageName,
				),
			)

			metadataResp, err := client.Do(metadataReq)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(metadataResp.StatusCode).To(gomega.Equal(200))

			// Verify metadata structure
			var metadata npm.PackageMetadata
			err = json.Unmarshal(metadataResp.Body, &metadata)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(metadata.Name).To(gomega.Equal(fullName))
			gomega.Expect(metadata.Versions).To(gomega.HaveKey(version))
			gomega.Expect(metadata.DistTags).To(gomega.HaveKeyWithValue("latest", version))
		})
	})
}
