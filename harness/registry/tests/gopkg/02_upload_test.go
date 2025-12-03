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

// registry/tests/gopkg/02_upload_test.go
package gopkgconformance

import (
	"bytes"
	"fmt"
	"mime/multipart"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

type UploadFormData struct {
	body        []byte
	contentType string
}

type PackageInfo struct {
	Version string `json:"version"`
	Time    string `json:"time"`
}

var test02Upload = func() {
	ginkgo.Context("Upload", func() {
		ginkgo.BeforeEach(func() {
			if !IsTestEnabled(TestUpload) {
				ginkgo.Skip("Upload tests are disabled")
			}
		})

		ginkgo.It("should upload a Go package", func() {
			// Use a unique package name and version for this test
			packageName := GetUniqueArtifactName("go", 2)
			version := GetUniqueVersion(2)

			// Generate the package payload
			formData := generateGoPackagePayload(packageName, version)

			// Set up the upload request
			req := client.NewRequest(
				"PUT",
				fmt.Sprintf("/pkg/%s/%s/go/upload",
					TestConfig.Namespace, TestConfig.RegistryName,
				),
			)
			req.SetHeader("Content-Type", formData.contentType)
			req.SetBody(formData.body)

			// Send the request and verify the response
			resp, err := client.Do(req)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(resp.StatusCode).To(gomega.Equal(200))
		})
	})
}

func generateGoPackagePayload(packageName string, version string) *UploadFormData {
	// create form and add 3 files with form keys mod, info and .zip
	// Buffer to hold multipart body
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	// Add dummy .mod file
	addFile(writer, "mod", version+".mod", "module "+packageName+"\n")

	// Add dummy .info file
	addFile(writer, "info", version+".info", `{"Version":"`+version+`","Time":"2023-01-01T00:00:00Z"}`)

	// Add dummy .zip file
	addFile(writer, "zip", version+".zip", "mock package content") // Zip header or dummy content

	// Close the multipart writer to set final boundary
	writer.Close()

	return &UploadFormData{
		body:        body.Bytes(),
		contentType: writer.FormDataContentType(),
	}
}

// Helper to add a form file with string content.
func addFile(writer *multipart.Writer, fieldName, fileName, content string) {
	part, err := writer.CreateFormFile(fieldName, fileName)
	gomega.Expect(err).To(gomega.BeNil())
	_, err = part.Write([]byte(content))
	gomega.Expect(err).To(gomega.BeNil())
}
