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
	"context"
	"fmt"
	"io"

	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/base"
	generic2 "github.com/harness/gitness/registry/app/pkg/generic"
	"github.com/harness/gitness/registry/app/pkg/response"
	"github.com/harness/gitness/registry/app/pkg/types/generic"
	registrytypes "github.com/harness/gitness/registry/types"
)

func (c Controller) DownloadFile(
	ctx context.Context,
	info generic.ArtifactInfo,
	filePath string,
) *GetArtifactResponse {
	f := func(registry registrytypes.Registry, a pkg.Artifact) response.Response {
		info.UpdateRegistryInfo(registry)
		genericRegistry, ok := a.(generic2.Registry)
		if !ok {
			return &GetArtifactResponse{
				BaseResponse: BaseResponse{
					Error: fmt.Errorf("invalid registry type: expected generic.Registry, got %T", a),
				},
			}
		}
		headers, fileReader, readCloser, redirectURL, err := genericRegistry.DownloadFile(ctx, info, filePath)
		return &GetArtifactResponse{
			BaseResponse: BaseResponse{
				Error:           err,
				ResponseHeaders: headers,
			},
			RedirectURL: redirectURL,
			Body:        fileReader,
			ReadCloser:  readCloser,
		}
	}

	result, err := base.ProxyWrapper(ctx, c.DBStore.RegistryDao, c.quarantineFinder, f, info, true)
	if err != nil {
		return &GetArtifactResponse{
			BaseResponse: BaseResponse{
				Error: err,
			},
		}
	}
	getResponse, ok := result.(*GetArtifactResponse)
	if !ok {
		return &GetArtifactResponse{
			BaseResponse: BaseResponse{
				Error: fmt.Errorf("invalid response type: expected GetArtifactResponse, got %T", result),
			},
		}
	}
	return getResponse
}

func (c Controller) HeadFile(
	ctx context.Context,
	info generic.ArtifactInfo,
	filePath string,
) *HeadArtifactResponse {
	f := func(registry registrytypes.Registry, a pkg.Artifact) response.Response {
		info.UpdateRegistryInfo(registry)
		genericRegistry, ok := a.(generic2.Registry)
		if !ok {
			return &HeadArtifactResponse{
				BaseResponse: BaseResponse{
					Error: fmt.Errorf("invalid registry type: expected generic.Registry, got %T", a),
				},
			}
		}
		headers, err := genericRegistry.HeadFile(ctx, info, filePath)
		return &HeadArtifactResponse{
			BaseResponse: BaseResponse{
				Error:           err,
				ResponseHeaders: headers,
			},
		}
	}

	result, err := base.ProxyWrapper(ctx, c.DBStore.RegistryDao, c.quarantineFinder, f, info, false)
	if err != nil {
		return &HeadArtifactResponse{
			BaseResponse: BaseResponse{
				Error: err,
			},
		}
	}
	headResponse, ok := result.(*HeadArtifactResponse)
	if !ok {
		return &HeadArtifactResponse{
			BaseResponse: BaseResponse{
				Error: fmt.Errorf("invalid response type: expected HeadArtifactResponse, got %T", result),
			},
		}
	}
	return headResponse
}

func (c Controller) DeleteFile(ctx context.Context, info generic.ArtifactInfo) *DeleteArtifactResponse {
	f := func(registry registrytypes.Registry, a pkg.Artifact) response.Response {
		info.UpdateRegistryInfo(registry)
		genericRegistry, ok := a.(generic2.Registry)
		if !ok {
			return &DeleteArtifactResponse{
				BaseResponse: BaseResponse{
					Error: fmt.Errorf("invalid registry type: expected generic.Registry, got %T", a),
				},
			}
		}
		headers, err := genericRegistry.DeleteFile(ctx, info)
		return &DeleteArtifactResponse{
			BaseResponse: BaseResponse{
				Error:           err,
				ResponseHeaders: headers,
			},
		}
	}

	result, err := base.NoProxyWrapper(ctx, c.DBStore.RegistryDao, f, info)
	if err != nil {
		return &DeleteArtifactResponse{
			BaseResponse: BaseResponse{
				Error: err,
			},
		}
	}
	deleteArtifactResponse, ok := result.(*DeleteArtifactResponse)
	if !ok {
		return &DeleteArtifactResponse{
			BaseResponse: BaseResponse{
				Error: fmt.Errorf("invalid response type: expected DeleteArtifactResponse, got %T", result),
			},
		}
	}
	return deleteArtifactResponse
}

func (c Controller) PutFile(
	ctx context.Context,
	info generic.ArtifactInfo,
	reader io.ReadCloser,
	contentType string,
) *PutArtifactResponse {
	f := func(registry registrytypes.Registry, a pkg.Artifact) response.Response {
		info.UpdateRegistryInfo(registry)
		genericRegistry, ok := a.(generic2.Registry)
		if !ok {
			return &PutArtifactResponse{
				BaseResponse: BaseResponse{
					Error: fmt.Errorf("invalid registry type: expected generic.Registry, got %T", a),
				},
			}
		}
		headers, sha256, err := genericRegistry.PutFile(ctx, info, reader, contentType)
		return &PutArtifactResponse{
			BaseResponse: BaseResponse{
				Error:           err,
				ResponseHeaders: headers,
			},
			Sha256: sha256,
		}
	}

	result, err := base.NoProxyWrapper(ctx, c.DBStore.RegistryDao, f, info)
	if err != nil {
		return &PutArtifactResponse{
			BaseResponse: BaseResponse{
				Error: err,
			},
		}
	}
	putArtifactResponse, ok := result.(*PutArtifactResponse)
	if !ok {
		return &PutArtifactResponse{
			BaseResponse: BaseResponse{
				Error: fmt.Errorf("invalid response type: expected PutArtifactResponse, got %T", result),
			},
		}
	}
	return putArtifactResponse
}
