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

package metadata

import (
	"testing"
	"time"

	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"

	"github.com/stretchr/testify/assert"
)

func TestValidatePackageTypes_ValidTypes(t *testing.T) {
	err := ValidatePackageTypes([]string{"DOCKER", "HELM"})
	assert.NoError(t, err)
}

func TestValidatePackageTypes_InvalidTypes(t *testing.T) {
	err := ValidatePackageTypes([]string{"INVALID"})
	assert.Error(t, err)
	assert.Equal(t, "invalid package type", err.Error())
}

func TestValidatePackageType_ValidType(t *testing.T) {
	err := ValidatePackageType("DOCKER")
	assert.NoError(t, err)
}

func TestValidatePackageType_InvalidType(t *testing.T) {
	err := ValidatePackageType("INVALID")
	assert.Error(t, err)
	assert.Equal(t, "invalid package type", err.Error())
}

func TestValidateIdentifier_ValidIdentifier(t *testing.T) {
	err := ValidateIdentifier("valid-identifier")
	assert.NoError(t, err)
}

func TestValidateIdentifier_InvalidIdentifier(t *testing.T) {
	err := ValidateIdentifier("Invalid Identifier")
	assert.Error(t, err)
	assert.Equal(t, RegistryIdentifierErrorMsg, err.Error())
}

func TestCleanURLPath_ValidURL(t *testing.T) {
	input := "https://example.com/path/"
	expected := "https://example.com/path"
	CleanURLPath(&input)
	assert.Equal(t, expected, input)
}

func TestCleanURLPath_InvalidURL(t *testing.T) {
	input := "://invalid-url"
	expected := "://invalid-url"
	CleanURLPath(&input)
	assert.Equal(t, expected, input)
}

func TestGetTimeInMs_ValidTime(t *testing.T) {
	tm := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	expected := "1672531200000"
	result := GetTimeInMs(tm)
	assert.Equal(t, expected, result)
}

func TestGetErrorResponse_ValidError(t *testing.T) {
	code := 404
	message := "Not Found"
	expected := &artifact.Error{
		Code:    "404",
		Message: message,
	}
	result := GetErrorResponse(code, message)
	assert.Equal(t, expected, result)
}

func TestGetSortByOrder_ValidOrder(t *testing.T) {
	assert.Equal(t, "ASC", GetSortByOrder(""))
	assert.Equal(t, "DESC", GetSortByOrder("DESC"))
	assert.Equal(t, "ASC", GetSortByOrder("INVALID"))
}

func TestGetSortByField_ValidField(t *testing.T) {
	assert.Equal(t, "name", GetSortByField("identifier", RepositoryResource))
	assert.Equal(t, "created_at", GetSortByField("invalid", RepositoryResource))
}

func TestGetPageLimit_ValidPageSize(t *testing.T) {
	pageSize := artifact.PageSize(20)
	assert.Equal(t, 20, GetPageLimit(&pageSize))
	assert.Equal(t, 10, GetPageLimit(nil))
}

func TestGetOffset_ValidOffset(t *testing.T) {
	pageSize := artifact.PageSize(20)
	pageNumber := artifact.PageNumber(2)
	assert.Equal(t, 40, GetOffset(&pageSize, &pageNumber))
	assert.Equal(t, 0, GetOffset(nil, nil))
}

func TestGetPageNumber_ValidPageNumber(t *testing.T) {
	pageNumber := artifact.PageNumber(2)
	assert.Equal(t, int64(2), GetPageNumber(&pageNumber))
	assert.Equal(t, int64(1), GetPageNumber(nil))
}

func TestGetSuccessResponse_ValidResponse(t *testing.T) {
	expected := &artifact.Success{
		Status: artifact.StatusSUCCESS,
	}
	result := GetSuccessResponse()
	assert.Equal(t, expected, result)
}

func TestGetPageCount_ValidCount(t *testing.T) {
	assert.Equal(t, int64(5), GetPageCount(50, 10))
	assert.Equal(t, int64(0), GetPageCount(0, 10))
}

func TestGetRegistryRef_ValidRef(t *testing.T) {
	assert.Equal(t, "root/registry", GetRegistryRef("root", "registry"))
}

func TestGetRepoURLWithoutProtocol_ValidURL(t *testing.T) {
	assert.Equal(t, "example.com/path", GetRepoURLWithoutProtocol("https://example.com/path"))
	assert.Equal(t, "", GetRepoURLWithoutProtocol("://invalid-url"))
}

func TestGetTagURL_ValidURL(t *testing.T) {
	assert.Equal(t, "https://example.com/artifact/version", GetTagURL("artifact", "version", "https://example.com"))
}

func TestGetPullCommand_ValidCommand(t *testing.T) {
	assert.Equal(t, "docker pull example.com/image:tag",
		GetPullCommand("image", "tag", "DOCKER", "https://example.com"))
	assert.Equal(t, "helm pull oci://example.com/image:tag",
		GetPullCommand("image", "tag", "HELM", "https://example.com"))
	assert.Equal(t, "", GetPullCommand("image", "tag", "INVALID", "https://example.com"))
}

func TestGetDockerPullCommand_ValidCommand(t *testing.T) {
	assert.Equal(t, "docker pull example.com/image:tag", GetDockerPullCommand("image", "tag", "https://example.com"))
}

func TestGetHelmPullCommand_ValidCommand(t *testing.T) {
	assert.Equal(t, "helm pull oci://example.com/image:tag", GetHelmPullCommand("image", "tag", "https://example.com"))
}
