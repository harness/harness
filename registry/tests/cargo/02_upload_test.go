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

// registry/tests/cargo/02_upload_test.go
package cargoconformance

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

type packageIndexMetadata struct {
	Name    string `json:"name"`
	Version string `json:"vers"`
	Cksum   string `json:"cksum"`
	Yanked  bool   `json:"yanked"`
}

var test02Upload = func() {
	ginkgo.Context("Upload", func() {
		ginkgo.BeforeEach(func() {
			if !IsTestEnabled(TestUpload) {
				ginkgo.Skip("Upload tests are disabled")
			}
		})

		ginkgo.It("should upload a Cargo package", func() {
			// Use a unique package name and version for this test
			packageName := GetUniqueArtifactName("cargo", 2)
			version := GetUniqueVersion(2)

			// get package index
			req := client.NewRequest(
				"GET",
				fmt.Sprintf("/pkg/%s/%s/cargo/%s",
					TestConfig.Namespace, TestConfig.RegistryName,
					getIndexFilePathFromImageName(packageName),
				),
			)
			resp, err := client.Do(req)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(resp.StatusCode).To(gomega.Equal(404))

			// Generate the package payload
			body := generateCargoPackagePayload(packageName, version)

			// Set up the upload request
			req = client.NewRequest(
				"PUT",
				fmt.Sprintf("/pkg/%s/%s/cargo/api/v1/crates/new",
					TestConfig.Namespace, TestConfig.RegistryName,
				),
			)
			req.SetHeader("Content-Type", "application/crate")
			req.SetBody(body)

			// Send the request and verify the response
			resp, err = client.Do(req)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(resp.StatusCode).To(gomega.Equal(201))

			time.Sleep(2 * time.Second)
			// get package index
			var indexArr []packageIndexMetadata
			req = client.NewRequest(
				"GET",
				fmt.Sprintf("/pkg/%s/%s/cargo/%s",
					TestConfig.Namespace, TestConfig.RegistryName,
					getIndexFilePathFromImageName(packageName),
				),
			)
			resp, err = client.Do(req)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(resp.StatusCode).To(gomega.Equal(http.StatusOK))
			gomega.Expect(resp.Headers.Get("Content-Type")).To(gomega.Equal("text/plain; charset=utf-8"))
			indexArr, err = getPackageIndexContentFromText(string(resp.Body))
			gomega.Expect(err).To(gomega.BeNil())

			gomega.Expect(len(indexArr)).To(gomega.Equal(1))
			gomega.Expect(indexArr[0].Name).To(gomega.Equal(packageName))
			gomega.Expect(indexArr[0].Version).To(gomega.Equal(version))
			gomega.Expect(indexArr[0].Yanked).To(gomega.Equal(false))

		})
	})
}

func generateCargoPackagePayload(packageName string, version string) []byte {
	// Create a test package file
	packageBytes := []byte("mock package content")

	// Create metadata for the package
	metadata := map[string]string{
		"name": packageName,
		"vers": version,
	}
	metadataBytes, err := json.Marshal(metadata)
	gomega.Expect(err).To(gomega.BeNil())

	metadataLen := uint32(len(metadataBytes)) // #nosec G115
	packageLen := uint32(len(packageBytes))   // #nosec G115
	// Construct the request body according to the Cargo package upload format
	body := make([]byte, 4+metadataLen+4+packageLen)
	binary.LittleEndian.PutUint32(body[:4], metadataLen)
	copy(body[4:], metadataBytes)
	binary.LittleEndian.PutUint32(body[4+metadataLen:4+metadataLen+4], packageLen)
	copy(body[4+metadataLen+4:], packageBytes)

	return body
}

func getIndexFilePathFromImageName(imageName string) string {
	length := len(imageName)
	switch length {
	case 0:
		return imageName
	case 1:
		return fmt.Sprintf("index/1/%s", imageName)
	case 2:
		return fmt.Sprintf("index/2/%s", imageName)
	case 3:
		return fmt.Sprintf("index/3/%c/%s", imageName[0], imageName)
	default:
		return fmt.Sprintf("index/%s/%s/%s", imageName[0:2], imageName[2:4], imageName)
	}
}

func getPackageIndexContentFromText(text string) ([]packageIndexMetadata, error) {
	result := []packageIndexMetadata{}
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var index packageIndexMetadata
		err := json.Unmarshal([]byte(line), &index)
		if err != nil {
			return nil, fmt.Errorf("failed to parse line: %q, error: %w", line, err)
		}
		result = append(result, index)
	}
	return result, nil
}
