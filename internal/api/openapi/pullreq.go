// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package openapi

import (
	"net/http"

	"github.com/harness/gitness/internal/api/controller/pullreq"
	"github.com/harness/gitness/internal/api/usererror"
	"github.com/harness/gitness/types"

	"github.com/swaggest/openapi-go/openapi3"
)

type createPullReqRequest struct {
	repoRequest
	pullreq.CreateInput
}

type pullReqRequest struct {
	repoRequest
	ID int64 `path:"pullreqNumber"`
}

type getPullReqRequest struct {
	pullReqRequest
}

func pullReqOperations(reflector *openapi3.Reflector) {
	createPullReq := openapi3.Operation{}
	createPullReq.WithTags("pullreq")
	createPullReq.WithMapOfAnything(map[string]interface{}{"operationId": "createPullReq"})
	createPullReq.WithParameters(queryParameterSpacePath)
	_ = reflector.SetRequest(&createPullReq, new(createPullReqRequest), http.MethodPost)
	_ = reflector.SetJSONResponse(&createPullReq, new(types.PullReq), http.StatusCreated)
	_ = reflector.SetJSONResponse(&createPullReq, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&createPullReq, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&createPullReq, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&createPullReq, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodPost, "/repos/{repoRef}/pullreq", createPullReq)

	getPullReq := openapi3.Operation{}
	getPullReq.WithTags("pullreq")
	getPullReq.WithMapOfAnything(map[string]interface{}{"operationId": "createPullReq"})
	getPullReq.WithParameters(queryParameterSpacePath)
	_ = reflector.SetRequest(&getPullReq, new(getPullReqRequest), http.MethodPost)
	_ = reflector.SetJSONResponse(&getPullReq, new(types.PullReq), http.StatusOK)
	_ = reflector.SetJSONResponse(&getPullReq, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&getPullReq, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&getPullReq, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&getPullReq, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/repos/{repoRef}/pullreq/{pullreqNumber}", getPullReq)
}
