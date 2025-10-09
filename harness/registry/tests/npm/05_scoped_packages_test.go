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

	npm2 "github.com/harness/gitness/registry/app/metadata/npm"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var test05ScopedPackages = func() {
	ginkgo.Context("Scoped Packages", func() {
		ginkgo.BeforeEach(func() {
			SkipIfDisabled(TestScopedPackages)
		})

		ginkgo.It("should handle complete lifecycle of scoped packages", func() {
			scope := "harness"
			packageName := "test-lifecycle"
			version := GetUniqueVersion(1)
			fullName := fmt.Sprintf("@%s/%s", scope, packageName)
			filename := fmt.Sprintf("%s-%s.tgz", packageName, version)

			// 1. Upload scoped package
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

			// 2. Retrieve metadata
			metadataReq := client.NewRequest(
				"GET",
				fmt.Sprintf("/pkg/%s/%s/npm/@%s/%s/",
					TestConfig.Namespace, TestConfig.RegistryName, scope, packageName,
				),
			)

			metadataResp, err := client.Do(metadataReq)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(metadataResp.StatusCode).To(gomega.Equal(200))

			var metadata npm2.PackageMetadata
			err = json.Unmarshal(metadataResp.Body, &metadata)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(metadata.Name).To(gomega.Equal(fullName))

			// 3. Download package file
			downloadReq := client.NewRequest(
				"GET",
				fmt.Sprintf("/pkg/%s/%s/npm/@%s/%s/-/@%s/%s",
					TestConfig.Namespace, TestConfig.RegistryName, scope, packageName, scope, filename,
				),
			)

			downloadResp, err := client.Do(downloadReq)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(downloadResp.StatusCode).To(gomega.Equal(200))

			// 4. Add custom tag
			customTag := "rc"
			addTagReq := client.NewRequest(
				"PUT",
				fmt.Sprintf("/pkg/%s/%s/npm/-/package/@%s/%s/dist-tags/%s",
					TestConfig.Namespace, TestConfig.RegistryName, scope, packageName, customTag,
				),
			)
			addTagReq.SetHeader("Content-Type", "application/json")
			addTagReq.SetBody(fmt.Sprintf(`"%s"`, version))

			addTagResp, err := client.Do(addTagReq)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(addTagResp.StatusCode).To(gomega.Equal(200))

			// 5. Verify tag was added
			tagsReq := client.NewRequest(
				"GET",
				fmt.Sprintf("/pkg/%s/%s/npm/-/package/@%s/%s/dist-tags/",
					TestConfig.Namespace, TestConfig.RegistryName, scope, packageName,
				),
			)

			tagsResp, err := client.Do(tagsReq)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(tagsResp.StatusCode).To(gomega.Equal(200))

			var tags map[string]string
			err = json.Unmarshal(tagsResp.Body, &tags)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(tags).To(gomega.HaveKeyWithValue(customTag, version))
		})

		ginkgo.It("should handle multiple versions of scoped packages", func() {
			scope := "multiversion"
			packageName := "test-versions"
			version1 := GetUniqueVersion(2)
			version2 := GetUniqueVersion(3)
			fullName := fmt.Sprintf("@%s/%s", scope, packageName)

			// Upload first version
			packageData1 := generateNpmPackagePayload(packageName, version1, true, scope)
			uploadReq1 := client.NewRequest(
				"PUT",
				fmt.Sprintf("/pkg/%s/%s/npm/@%s/%s/",
					TestConfig.Namespace, TestConfig.RegistryName, scope, packageName,
				),
			)
			uploadReq1.SetHeader("Content-Type", "application/json")
			uploadReq1.SetBody(packageData1)

			uploadResp1, err := client.Do(uploadReq1)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(uploadResp1.StatusCode).To(gomega.Equal(200))

			// Upload second version
			packageData2 := generateNpmPackagePayload(packageName, version2, true, scope)
			uploadReq2 := client.NewRequest(
				"PUT",
				fmt.Sprintf("/pkg/%s/%s/npm/@%s/%s/",
					TestConfig.Namespace, TestConfig.RegistryName, scope, packageName,
				),
			)
			uploadReq2.SetHeader("Content-Type", "application/json")
			uploadReq2.SetBody(packageData2)

			uploadResp2, err := client.Do(uploadReq2)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(uploadResp2.StatusCode).To(gomega.Equal(200))

			// Retrieve metadata and verify both versions exist
			metadataReq := client.NewRequest(
				"GET",
				fmt.Sprintf("/pkg/%s/%s/npm/@%s/%s/",
					TestConfig.Namespace, TestConfig.RegistryName, scope, packageName,
				),
			)

			metadataResp, err := client.Do(metadataReq)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(metadataResp.StatusCode).To(gomega.Equal(200))

			var metadata npm2.PackageMetadata
			err = json.Unmarshal(metadataResp.Body, &metadata)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(metadata.Name).To(gomega.Equal(fullName))
			gomega.Expect(metadata.Versions).To(gomega.HaveKey(version1))
			gomega.Expect(metadata.Versions).To(gomega.HaveKey(version2))
		})

		ginkgo.It("should handle HEAD requests for scoped package files", func() {
			scope := "headtest"
			packageName := "test-head"
			version := GetUniqueVersion(4)
			filename := fmt.Sprintf("%s-%s.tgz", packageName, version)

			// Upload scoped package
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

			// Perform HEAD request
			headReq := client.NewRequest(
				"HEAD",
				fmt.Sprintf("/pkg/%s/%s/npm/@%s/%s/-/@%s/%s",
					TestConfig.Namespace, TestConfig.RegistryName, scope, packageName, scope, filename,
				),
			)

			headResp, err := client.Do(headReq)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(headResp.StatusCode).To(gomega.Equal(200))
			gomega.Expect(len(headResp.Body)).To(gomega.Equal(0)) // HEAD should not return body
		})
	})
}
