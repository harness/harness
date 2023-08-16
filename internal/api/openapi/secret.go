// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package openapi

import (
	"net/http"

	"github.com/harness/gitness/internal/api/controller/secret"
	"github.com/harness/gitness/internal/api/usererror"
	"github.com/harness/gitness/types"

	"github.com/swaggest/openapi-go/openapi3"
)

type createSecretRequest struct {
	secret.CreateInput
}

type secretRequest struct {
	Ref string `path:"secret_ref"`
}

type getSecretRequest struct {
	secretRequest
}

type updateSecretRequest struct {
	secretRequest
	secret.UpdateInput
}

func secretOperations(reflector *openapi3.Reflector) {
	opCreate := openapi3.Operation{}
	opCreate.WithTags("secret")
	opCreate.WithMapOfAnything(map[string]interface{}{"operationId": "createSecret"})
	_ = reflector.SetRequest(&opCreate, new(createSecretRequest), http.MethodPost)
	_ = reflector.SetJSONResponse(&opCreate, new(types.Secret), http.StatusCreated)
	_ = reflector.SetJSONResponse(&opCreate, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opCreate, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opCreate, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opCreate, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodPost, "/secrets", opCreate)

	opFind := openapi3.Operation{}
	opFind.WithTags("secret")
	opFind.WithMapOfAnything(map[string]interface{}{"operationId": "findSecret"})
	_ = reflector.SetRequest(&opFind, new(getSecretRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&opFind, new(types.Secret), http.StatusOK)
	_ = reflector.SetJSONResponse(&opFind, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opFind, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opFind, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opFind, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/secrets/{secret_ref}", opFind)

	opDelete := openapi3.Operation{}
	opDelete.WithTags("secret")
	opDelete.WithMapOfAnything(map[string]interface{}{"operationId": "deleteSecret"})
	_ = reflector.SetRequest(&opDelete, new(getSecretRequest), http.MethodDelete)
	_ = reflector.SetJSONResponse(&opDelete, nil, http.StatusNoContent)
	_ = reflector.SetJSONResponse(&opDelete, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opDelete, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opDelete, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opDelete, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodDelete, "/secrets/{secret_ref}", opDelete)

	opUpdate := openapi3.Operation{}
	opUpdate.WithTags("secret")
	opUpdate.WithMapOfAnything(map[string]interface{}{"operationId": "updateSecret"})
	_ = reflector.SetRequest(&opUpdate, new(updateSecretRequest), http.MethodPatch)
	_ = reflector.SetJSONResponse(&opUpdate, new(types.Secret), http.StatusOK)
	_ = reflector.SetJSONResponse(&opUpdate, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opUpdate, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opUpdate, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opUpdate, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opUpdate, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodPatch, "/secrets/{secret_ref}", opUpdate)
}
