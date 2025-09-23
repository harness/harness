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

package metadata_test

import (
	"testing"
	"time"

	"github.com/harness/gitness/registry/app/api/controller/metadata"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"

	"github.com/stretchr/testify/assert"
)

func TestValidatePackageTypes(t *testing.T) {
	tests := []struct {
		name    string
		types   []string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid_types",
			types:   []string{"DOCKER", "HELM"},
			wantErr: false,
		},
		{
			name:    "invalid_type",
			types:   []string{"INVALID"},
			wantErr: true,
			errMsg:  "invalid package type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := metadata.ValidatePackageTypes(tt.types)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Equal(t, tt.errMsg, err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidatePackageType(t *testing.T) {
	tests := []struct {
		name    string
		pkgType string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid_type",
			pkgType: "DOCKER",
			wantErr: false,
		},
		{
			name:    "invalid_type",
			pkgType: "INVALID",
			wantErr: true,
			errMsg:  "invalid package type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := metadata.ValidatePackageType(tt.pkgType)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Equal(t, tt.errMsg, err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateIdentifier(t *testing.T) {
	tests := []struct {
		name       string
		identifier string
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "valid_identifier",
			identifier: "valid-identifier",
			wantErr:    false,
		},
		{
			name:       "invalid_identifier",
			identifier: "Invalid Identifier",
			wantErr:    true,
			errMsg:     metadata.RegistryIdentifierErrorMsg,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := metadata.ValidateIdentifier(tt.identifier)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Equal(t, tt.errMsg, err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCleanURLPath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid_url",
			input:    "https://example.com/path/",
			expected: "https://example.com/path",
		},
		{
			name:     "invalid_url",
			input:    "://invalid-url",
			expected: "://invalid-url",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := tt.input
			metadata.CleanURLPath(&input)
			assert.Equal(t, tt.expected, input)
		})
	}
}

func TestGetTimeInMs(t *testing.T) {
	tests := []struct {
		name     string
		time     time.Time
		expected string
	}{
		{
			name:     "valid_time",
			time:     time.Unix(1234567890, 0),
			expected: "1234567890000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := metadata.GetTimeInMs(tt.time)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetPullCommand(t *testing.T) {
	tests := []struct {
		name     string
		image    string
		tag      string
		pkgType  string
		baseURL  string
		expected string
	}{
		{
			name:     "docker_command",
			image:    "image",
			tag:      "tag",
			pkgType:  "DOCKER",
			baseURL:  "https://example.com",
			expected: "docker pull example.com/image:tag",
		},
		{
			name:     "helm_command",
			image:    "image",
			tag:      "tag",
			pkgType:  "HELM",
			baseURL:  "https://example.com",
			expected: "helm pull oci://example.com/image --version tag",
		},
		{
			name:     "invalid_type",
			image:    "image",
			tag:      "tag",
			pkgType:  "INVALID",
			baseURL:  "https://example.com",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := metadata.GetPullCommand(tt.image, tt.tag, tt.pkgType, tt.baseURL,
				"Authorization: Bearer", nil, true)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetDockerPullCommand(t *testing.T) {
	tests := []struct {
		name     string
		image    string
		tag      string
		baseURL  string
		expected string
	}{
		{
			name:     "valid_command",
			image:    "image",
			tag:      "tag",
			baseURL:  "https://example.com",
			expected: "docker pull example.com/image:tag",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := metadata.GetDockerPullCommand(tt.image, tt.tag, tt.baseURL, true)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetHelmPullCommand(t *testing.T) {
	tests := []struct {
		name     string
		image    string
		tag      string
		baseURL  string
		expected string
	}{
		{
			name:     "valid_command",
			image:    "image",
			tag:      "tag",
			baseURL:  "https://example.com",
			expected: "helm pull oci://example.com/image --version tag",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := metadata.GetHelmPullCommand(tt.image, tt.tag, tt.baseURL, true)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetErrorResponse(t *testing.T) {
	tests := []struct {
		name     string
		code     int
		message  string
		expected *artifact.Error
	}{
		{
			name:    "valid_error",
			code:    404,
			message: "Not Found",
			expected: &artifact.Error{
				Code:    "404",
				Message: "Not Found",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := metadata.GetErrorResponse(tt.code, tt.message)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetSortByOrder(t *testing.T) {
	tests := []struct {
		name     string
		order    string
		expected string
	}{
		{
			name:     "empty_order",
			order:    "",
			expected: "ASC",
		},
		{
			name:     "desc_order",
			order:    "DESC",
			expected: "DESC",
		},
		{
			name:     "invalid_order",
			order:    "INVALID",
			expected: "ASC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := metadata.GetSortByOrder(tt.order)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetSortByField(t *testing.T) {
	tests := []struct {
		name     string
		field    string
		resource string
		expected string
	}{
		{
			name:     "identifier_field",
			field:    "identifier",
			resource: "repository",
			expected: "name",
		},
		{
			name:     "invalid_field",
			field:    "invalid",
			resource: "repository",
			expected: "created_at",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := metadata.GetSortByField(tt.field, tt.resource)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetPageLimit(t *testing.T) {
	tests := []struct {
		name     string
		pageSize *artifact.PageSize
		expected int
	}{
		{
			name:     "valid_page_size",
			pageSize: func() *artifact.PageSize { ps := artifact.PageSize(20); return &ps }(),
			expected: 20,
		},
		{
			name:     "nil_page_size",
			pageSize: nil,
			expected: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := metadata.GetPageLimit(tt.pageSize)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetOffset(t *testing.T) {
	tests := []struct {
		name       string
		pageSize   *artifact.PageSize
		pageNumber *artifact.PageNumber
		expected   int
	}{
		{
			name:       "valid_offset",
			pageSize:   func() *artifact.PageSize { ps := artifact.PageSize(20); return &ps }(),
			pageNumber: func() *artifact.PageNumber { pn := artifact.PageNumber(2); return &pn }(),
			expected:   40,
		},
		{
			name:       "nil_values",
			pageSize:   nil,
			pageNumber: nil,
			expected:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := metadata.GetOffset(tt.pageSize, tt.pageNumber)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetPageNumber(t *testing.T) {
	tests := []struct {
		name       string
		pageNumber *artifact.PageNumber
		expected   int64
	}{
		{
			name:       "valid_page_number",
			pageNumber: func() *artifact.PageNumber { pn := artifact.PageNumber(2); return &pn }(),
			expected:   2,
		},
		{
			name:       "nil_page_number",
			pageNumber: nil,
			expected:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := metadata.GetPageNumber(tt.pageNumber)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetSuccessResponse(t *testing.T) {
	tests := []struct {
		name     string
		expected *artifact.Success
	}{
		{
			name: "valid_response",
			expected: &artifact.Success{
				Status: artifact.StatusSUCCESS,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := metadata.GetSuccessResponse()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetPageCount(t *testing.T) {
	tests := []struct {
		name     string
		count    int64
		limit    int
		expected int64
	}{
		{
			name:     "valid_count",
			count:    50,
			limit:    10,
			expected: 5,
		},
		{
			name:     "zero_count",
			count:    0,
			limit:    10,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := metadata.GetPageCount(tt.count, tt.limit)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetRegistryRef(t *testing.T) {
	tests := []struct {
		name     string
		root     string
		registry string
		expected string
	}{
		{
			name:     "valid_ref",
			root:     "root",
			registry: "registry",
			expected: "root/registry",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := metadata.GetRegistryRef(tt.root, tt.registry)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetRepoURLWithoutProtocol(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "valid_url",
			url:      "https://example.com/path",
			expected: "example.com/path",
		},
		{
			name:     "invalid_url",
			url:      "://invalid-url",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := metadata.GetRepoURLWithoutProtocol(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetTagURL(t *testing.T) {
	tests := []struct {
		name     string
		artifact string
		version  string
		baseURL  string
		expected string
	}{
		{
			name:     "valid_url",
			artifact: "artifact",
			version:  "version",
			baseURL:  "https://example.com",
			expected: "https://example.com/artifact/version",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := metadata.GetTagURL(tt.artifact, tt.version, tt.baseURL)
			assert.Equal(t, tt.expected, result)
		})
	}
}
