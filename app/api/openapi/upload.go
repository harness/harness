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

package openapi

import (
	"net/http"

	"github.com/harness/gitness/app/api/controller/upload"
	"github.com/harness/gitness/app/api/usererror"

	"github.com/swaggest/openapi-go/openapi3"
)

type UploadRequest struct {
	repoRequest
	// Note: Below line won't produce the file upload interface in Swagger UI,
	// ref: https://swagger.io/docs/specification/2-0/file-upload/
	Content string `json:"-" format:"binary" description:"Binary file to upload"`
}

type DownloadRequest struct {
	repoRequest
	FilePathRef string `path:"file_ref"`
}

func uploadOperations(reflector *openapi3.Reflector) {
	opUpload := openapi3.Operation{}
	opUpload.WithTags("upload")
	opUpload.WithMapOfAnything(map[string]interface{}{"operationId": "repoArtifactUpload"})
	_ = reflector.SetRequest(&opUpload, new(UploadRequest), http.MethodPost)
	_ = reflector.SetJSONResponse(&opUpload, new(upload.Result), http.StatusCreated)
	_ = reflector.SetJSONResponse(&opUpload, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opUpload, new(usererror.Error), http.StatusNotFound)
	_ = reflector.SetJSONResponse(&opUpload, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opUpload, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opUpload, new(usererror.Error), http.StatusForbidden)

	_ = reflector.Spec.AddOperation(http.MethodPost, "/repos/{repo_ref}/uploads", opUpload)

	downloadOp := openapi3.Operation{}
	downloadOp.WithTags("upload")
	downloadOp.WithMapOfAnything(map[string]interface{}{"operationId": "repoArtifactDownload"})
	_ = reflector.SetRequest(&downloadOp, new(DownloadRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&downloadOp, nil, http.StatusTemporaryRedirect)
	_ = reflector.SetJSONResponse(&downloadOp, new(usererror.Error), http.StatusNotFound)
	_ = reflector.SetJSONResponse(&downloadOp, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&downloadOp, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&downloadOp, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&downloadOp, new(usererror.Error), http.StatusForbidden)

	_ = reflector.Spec.AddOperation(http.MethodGet, "/repos/{repo_ref}/uploads/{file_ref}", downloadOp)
}
