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

package pypi

import (
	"context"
	"net/http"

	"github.com/harness/gitness/registry/app/dist_temp/errcode"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/registry/types"
)

func (c *controller) DownloadPackageFile(ctx context.Context, info pkg.ArtifactInfo, image, version, filename string) (
	*commons.ResponseHeaders,
	*storage.FileReader,
	string,
	errcode.Error,
) {
	responseHeaders := &commons.ResponseHeaders{
		Headers: make(map[string]string),
		Code:    0,
	}

	path := "/" + image + "/" + version + "/" + filename
	reg, _ := c.registryDao.GetByRootParentIDAndName(ctx, info.RootParentID, info.RegIdentifier)

	fileReader, _, redirectURL, err := c.fileManager.DownloadFile(ctx, path, types.Registry{
		ID:   reg.ID,
		Name: info.RegIdentifier,
	}, info.RootIdentifier)
	if err != nil {
		return responseHeaders, nil, "", errcode.ErrCodeRootNotFound.WithDetail(err)
	}
	responseHeaders.Code = http.StatusOK
	return responseHeaders, fileReader, redirectURL, errcode.Error{}
}
