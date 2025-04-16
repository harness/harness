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

package utils

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/types"
)

const (
	extensionXML         = ".xml"
	extensionMD5         = ".md5"
	extensionSHA1        = ".sha1"
	extensionSHA256      = ".sha256"
	extensionSHA512      = ".sha512"
	extensionPom         = ".pom"
	extensionJar         = ".jar"
	contentTypeJar       = "application/java-archive"
	contentTypeXML       = "text/xml"
	contentTypePlainText = "text/plain"
)

const (
	Jar   = ".jar"
	War   = ".war"
	Ear   = ".ear"
	Zip   = ".zip"
	TarGz = ".tar.gz"
	So    = ".so"
	Dll   = ".dll"
	Dylib = ".dylib"
	Rpm   = ".rpm"
	Deb   = ".deb"
	Exe   = ".exe"
)

var MainArtifactFileExtensions = []string{
	Jar,
	War,
	Ear,
	Zip,
	TarGz,
	So,
	Dll,
	Dylib,
	Rpm,
	Deb,
	Exe,
}

func GetFilePath(info pkg.MavenArtifactInfo) string {
	groupIDPath := strings.ReplaceAll(info.GroupID, ".", "/")
	return "/" + groupIDPath + "/" + info.ArtifactID + "/" + info.Version + "/" + info.FileName
}

func IsMainArtifactFile(info pkg.MavenArtifactInfo) bool {
	filePath := GetFilePath(info)
	fileExtension := strings.ToLower(filepath.Ext(filePath))
	for _, ext := range MainArtifactFileExtensions {
		if ext == fileExtension {
			return true
		}
	}
	return false
}

func SetHeaders(
	info pkg.MavenArtifactInfo,
	fileInfo types.FileInfo,
) *commons.ResponseHeaders {
	responseHeaders := &commons.ResponseHeaders{
		Headers: map[string]string{},
		Code:    http.StatusOK,
	}
	filePath := GetFilePath(info)
	ext := strings.ToLower(filepath.Ext(filePath))
	responseHeaders.Code = http.StatusOK
	responseHeaders.Headers["Content-Length"] = fmt.Sprintf("%d", fileInfo.Size)
	responseHeaders.Headers["LastModified"] = fmt.Sprintf("%d", fileInfo.CreatedAt.Unix())
	responseHeaders.Headers["FileName"] = fileInfo.Filename
	switch ext {
	case extensionJar:
		responseHeaders.Headers["Content-Type"] = contentTypeJar
	case extensionPom, extensionXML:
		responseHeaders.Headers["Content-Type"] = contentTypeXML
	case extensionMD5, extensionSHA1, extensionSHA256, extensionSHA512:
		responseHeaders.Headers["Content-Type"] = contentTypePlainText
	}
	return responseHeaders
}

func IsSnapshotVersion(info pkg.MavenArtifactInfo) bool {
	return strings.HasSuffix(info.Version, "-SNAPSHOT")
}

func AddLikeBeforeExtension(info pkg.MavenArtifactInfo) string {
	filePath := GetFilePath(info)
	ext := strings.ToLower(filepath.Ext(filePath))
	extIndex := len(filePath) - len(ext)
	return filePath[:extIndex] + "%" + ext
}

func ParseResponseHeaders(resp *http.Response) *commons.ResponseHeaders {
	headers := make(map[string]string)
	if resp.Header != nil {
		for key, values := range resp.Header {
			headers[key] = values[0]
		}
	}
	return &commons.ResponseHeaders{
		Headers: headers,
		Code:    http.StatusOK,
	}
}
