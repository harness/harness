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

package rpm

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/base"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	"github.com/harness/gitness/registry/app/pkg/types/rpm"
	"github.com/harness/gitness/registry/app/storage"
	rpmutil "github.com/harness/gitness/registry/app/utils/rpm"
)

type Registry interface {
	pkg.Artifact

	UploadPackageFile(
		ctx context.Context,
		info rpm.ArtifactInfo,
		file io.Reader,
		fileName string,
	) (*commons.ResponseHeaders, string, error)

	DownloadPackageFile(ctx context.Context, info rpm.ArtifactInfo) (
		*commons.ResponseHeaders,
		*storage.FileReader,
		io.ReadCloser,
		string,
		error,
	)
	GetRepoData(ctx context.Context, info rpm.ArtifactInfo, fileName string) (
		*commons.ResponseHeaders,
		*storage.FileReader,
		io.ReadCloser,
		string,
		error,
	)
}

func downloadPackageFile(
	ctx context.Context,
	info rpm.ArtifactInfo,
	localBase base.LocalBase,
) (*commons.ResponseHeaders, *storage.FileReader, io.ReadCloser, string, error) {
	var versionWithArch = info.Version
	var epoch string
	var versionPath string
	var version string

	epochSeparatorIndex := strings.Index(info.Version, ":")
	if epochSeparatorIndex != -1 {
		epoch = versionWithArch[:epochSeparatorIndex]
		versionWithArch = versionWithArch[epochSeparatorIndex+1:]
	}
	lastDotIndex := strings.LastIndex(versionWithArch, ".")
	if lastDotIndex == -1 {
		return nil, nil, nil, "", fmt.Errorf("invalid version: %s", info.Version)
	}
	version = versionWithArch[:lastDotIndex]

	if epoch != "" {
		versionPath = version + "/" + info.Arch + "/" + epoch
	} else {
		versionPath = version + "/" + info.Arch
	}

	headers, fileReader, redirectURL, err := localBase.Download(
		ctx, info.ArtifactInfo, versionPath, info.FileName,
	)
	if err != nil {
		return nil, nil, nil, "", err
	}
	return headers, fileReader, nil, redirectURL, nil
}

func getRepoData(
	ctx context.Context,
	info rpm.ArtifactInfo,
	fileName string,
	fileManager filemanager.FileManager,
) (*commons.ResponseHeaders, *storage.FileReader, io.ReadCloser, string, error) {
	responseHeaders := &commons.ResponseHeaders{
		Headers: make(map[string]string),
		Code:    0,
	}
	fileReader, _, redirectURL, err := fileManager.DownloadFileByPath(
		ctx, "/"+rpmutil.RepoDataPrefix+fileName, info.RegistryID, info.RegIdentifier, info.RootIdentifier, true,
	)
	if err != nil {
		return responseHeaders, nil, nil, "", err
	}
	responseHeaders.Code = http.StatusOK
	return responseHeaders, fileReader, nil, redirectURL, nil
}
