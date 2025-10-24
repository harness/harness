//  Copyright 2023 Harness, Inc.
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

package generic

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Helper to create multipart request.
func createMultipartRequest(t *testing.T, filename, content string) *http.Request {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatal(err)
	}
	_, err = part.Write([]byte(content))
	if err != nil {
		t.Fatal(err)
	}
	err = writer.WriteField("filename", filename)
	if err != nil {
		t.Fatal(err)
	}
	err = writer.WriteField("description", "Test description")
	if err != nil {
		t.Fatal(err)
	}
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/test", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

func TestCreateMultipartRequest(t *testing.T) {
	req := createMultipartRequest(t, "test.txt", "content")

	assert.Equal(t, http.MethodPost, req.Method)
	assert.Contains(t, req.Header.Get("Content-Type"), "multipart/form-data")
	assert.NotNil(t, req.Body)
}

func TestHandler_Struct(t *testing.T) {
	handler := &Handler{}

	assert.NotNil(t, handler)
	assert.Nil(t, handler.Controller)
	assert.Nil(t, handler.SpaceStore)
}
