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

package lfs

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/store"
	"github.com/harness/gitness/types/enum"
)

func (c *Controller) LFSTransfer(ctx context.Context,
	session *auth.Session,
	repoRef string,
	in *TransferInput,
) (*TransferOutput, error) {
	reqPermission := enum.PermissionRepoView
	if in.Operation == enum.GitLFSOperationTypeUpload {
		reqPermission = enum.PermissionRepoPush
	}

	var additionalAllowedRepoStates = []enum.RepoState{enum.RepoStateMigrateGitPush}
	repo, err := c.getRepoCheckAccessAndSetting(ctx, session, repoRef,
		reqPermission, additionalAllowedRepoStates...)
	if err != nil {
		return nil, err
	}

	// TODO check if server supports client's transfer adapters
	var objResponses []ObjectResponse
	switch {
	case in.Operation == enum.GitLFSOperationTypeDownload:
		for _, obj := range in.Objects {
			var objResponse = ObjectResponse{
				Pointer: Pointer{
					OId:  obj.OId,
					Size: obj.Size,
				},
			}

			object, err := c.lfsStore.Find(ctx, repo.ID, obj.OId)
			if errors.Is(err, store.ErrResourceNotFound) {
				objResponse.Error = &errNotFound
				objResponses = append(objResponses, objResponse)
				continue
			}
			if err != nil {
				return nil, fmt.Errorf("failed to find object: %w", err)
			}

			// size is not a required query param for download hence nil
			downloadURL := getRedirectRef(ctx, c.urlProvider, repoRef, obj.OId, nil)

			objResponse = ObjectResponse{
				Pointer: Pointer{
					OId:  object.OID,
					Size: object.Size,
				},
				Actions: map[string]Action{
					"download": {
						Href:   downloadURL,
						Header: map[string]string{"Content-Type": "application/octet-stream"},
					},
				},
			}

			objResponses = append(objResponses, objResponse)
		}

	case in.Operation == enum.GitLFSOperationTypeUpload:
		for _, obj := range in.Objects {
			objResponse := ObjectResponse{
				Pointer: Pointer{
					OId:  obj.OId,
					Size: obj.Size,
				},
			}
			// we dont create the object in lfs store here as the upload might fail in blob store.
			_, err := c.lfsStore.Find(ctx, repo.ID, obj.OId)
			if err == nil {
				// no need to re-upload existing LFS objects
				objResponses = append(objResponses, objResponse)
				continue
			}

			if !errors.Is(err, store.ErrResourceNotFound) {
				return nil, fmt.Errorf("failed to find object: %w", err)
			}

			uploadURL := getRedirectRef(ctx, c.urlProvider, repoRef, obj.OId, &obj.Size)

			objResponse.Actions = map[string]Action{
				"upload": {
					Href:   uploadURL,
					Header: map[string]string{"Content-Type": "application/octet-stream"},
				},
			}

			objResponses = append(objResponses, objResponse)
		}

	default:
		return nil, usererror.BadRequestf("Git LFS operation %q is not supported", in.Operation)
	}

	return &TransferOutput{
		Transfer: enum.GitLFSTransferTypeBasic,
		Objects:  objResponses,
	}, nil
}

func getRedirectRef(ctx context.Context, urlProvider url.Provider, repoPath, oID string, size *int64) string {
	baseGitURL := urlProvider.GenerateGITCloneURL(ctx, repoPath)
	queryParams := "oid=" + oID
	if size != nil {
		queryParams += "&size=" + strconv.FormatInt(*size, 10)
	}

	return baseGitURL + "/info/lfs/objects/?" + queryParams
}
