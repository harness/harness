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

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var test04TagOperations = func() {
	ginkgo.Context("Tag Operations", func() {
		ginkgo.BeforeEach(func() {
			SkipIfDisabled(TestTagOperations)
		})

		ginkgo.It("should list package tags for non-scoped package", func() {
			packageName := GetUniquePackageName("tags", 1)
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

			// Now list tags
			tagsReq := client.NewRequest(
				"GET",
				fmt.Sprintf("/pkg/%s/%s/npm/-/package/%s/dist-tags",
					TestConfig.Namespace, TestConfig.RegistryName, packageName,
				),
			)

			tagsResp, err := client.Do(tagsReq)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(tagsResp.StatusCode).To(gomega.Equal(200))

			// Verify tags structure
			var tags map[string]string
			err = json.Unmarshal(tagsResp.Body, &tags)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(tags).To(gomega.HaveKeyWithValue("latest", version))
		})

		ginkgo.It("should add a new tag to non-scoped package", func() {
			packageName := GetUniquePackageName("add-tag", 2)
			version := GetUniqueVersion(2)
			newTag := "beta"

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

			// Add new tag
			addTagReq := client.NewRequest(
				"PUT",
				fmt.Sprintf("/pkg/%s/%s/npm/-/package/%s/dist-tags/%s",
					TestConfig.Namespace, TestConfig.RegistryName, packageName, newTag,
				),
			)
			addTagReq.SetHeader("Content-Type", "application/json")
			addTagReq.SetBody(fmt.Sprintf(`"%s"`, version))

			addTagResp, err := client.Do(addTagReq)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(addTagResp.StatusCode).To(gomega.Equal(200))

			// Verify tag was added
			tagsReq := client.NewRequest(
				"GET",
				fmt.Sprintf("/pkg/%s/%s/npm/-/package/%s/dist-tags/",
					TestConfig.Namespace, TestConfig.RegistryName, packageName,
				),
			)

			tagsResp, err := client.Do(tagsReq)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(tagsResp.StatusCode).To(gomega.Equal(200))

			var tags map[string]string
			err = json.Unmarshal(tagsResp.Body, &tags)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(tags).To(gomega.HaveKeyWithValue(newTag, version))
		})

		ginkgo.It("should delete a tag from non-scoped package", func() {
			packageName := GetUniquePackageName("delete-tag", 3)
			version := GetUniqueVersion(3)
			tagToDelete := "beta"

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

			// Add tag first
			addTagReq := client.NewRequest(
				"PUT",
				fmt.Sprintf("/pkg/%s/%s/npm/-/package/%s/dist-tags/%s",
					TestConfig.Namespace, TestConfig.RegistryName, packageName, tagToDelete,
				),
			)
			addTagReq.SetHeader("Content-Type", "application/json")
			addTagReq.SetBody(fmt.Sprintf(`"%s"`, version))

			addTagResp, err := client.Do(addTagReq)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(addTagResp.StatusCode).To(gomega.Equal(200))

			// Now delete the tag
			deleteTagReq := client.NewRequest(
				"DELETE",
				fmt.Sprintf("/pkg/%s/%s/npm/-/package/%s/dist-tags/%s",
					TestConfig.Namespace, TestConfig.RegistryName, packageName, tagToDelete,
				),
			)

			deleteTagResp, err := client.Do(deleteTagReq)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(deleteTagResp.StatusCode).To(gomega.Equal(200))

			// Verify tag was deleted
			tagsReq := client.NewRequest(
				"GET",
				fmt.Sprintf("/pkg/%s/%s/npm/-/package/%s/dist-tags/",
					TestConfig.Namespace, TestConfig.RegistryName, packageName,
				),
			)

			tagsResp, err := client.Do(tagsReq)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(tagsResp.StatusCode).To(gomega.Equal(200))

			var tags map[string]string
			err = json.Unmarshal(tagsResp.Body, &tags)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(tags).ToNot(gomega.HaveKey(tagToDelete))
		})

		ginkgo.It("should handle tag operations for scoped packages", func() {
			scope := TestScope
			packageName := "test-scoped-tags"
			version := GetUniqueVersion(4)
			newTag := "alpha"

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

			// List tags for scoped package
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
			gomega.Expect(tags).To(gomega.HaveKeyWithValue("latest", version))

			// Add new tag to scoped package
			addTagReq := client.NewRequest(
				"PUT",
				fmt.Sprintf("/pkg/%s/%s/npm/-/package/@%s/%s/dist-tags/%s",
					TestConfig.Namespace, TestConfig.RegistryName, scope, packageName, newTag,
				),
			)
			addTagReq.SetHeader("Content-Type", "application/json")
			addTagReq.SetBody(fmt.Sprintf(`"%s"`, version))

			addTagResp, err := client.Do(addTagReq)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(addTagResp.StatusCode).To(gomega.Equal(200))
		})
	})
}
