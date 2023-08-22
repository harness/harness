// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package openapi

import (
	"net/http"

	"github.com/harness/gitness/internal/api/controller/connector"
	"github.com/harness/gitness/internal/api/usererror"
	"github.com/harness/gitness/types"

	"github.com/swaggest/openapi-go/openapi3"
)

type createConnectorRequest struct {
	connector.CreateInput
}

type connectorRequest struct {
	Ref string `path:"connector_ref"`
}

type getConnectorRequest struct {
	connectorRequest
}

type updateConnectorRequest struct {
	connectorRequest
	connector.UpdateInput
}

func connectorOperations(reflector *openapi3.Reflector) {
	opCreate := openapi3.Operation{}
	opCreate.WithTags("connector")
	opCreate.WithMapOfAnything(map[string]interface{}{"operationId": "createConnector"})
	_ = reflector.SetRequest(&opCreate, new(createConnectorRequest), http.MethodPost)
	_ = reflector.SetJSONResponse(&opCreate, new(types.Connector), http.StatusCreated)
	_ = reflector.SetJSONResponse(&opCreate, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opCreate, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opCreate, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opCreate, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodPost, "/connectors", opCreate)

	opFind := openapi3.Operation{}
	opFind.WithTags("connector")
	opFind.WithMapOfAnything(map[string]interface{}{"operationId": "findConnector"})
	_ = reflector.SetRequest(&opFind, new(getConnectorRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&opFind, new(types.Connector), http.StatusOK)
	_ = reflector.SetJSONResponse(&opFind, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opFind, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opFind, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opFind, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/connectors/{connector_ref}", opFind)

	opDelete := openapi3.Operation{}
	opDelete.WithTags("connector")
	opDelete.WithMapOfAnything(map[string]interface{}{"operationId": "deleteConnector"})
	_ = reflector.SetRequest(&opDelete, new(getConnectorRequest), http.MethodDelete)
	_ = reflector.SetJSONResponse(&opDelete, nil, http.StatusNoContent)
	_ = reflector.SetJSONResponse(&opDelete, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opDelete, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opDelete, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opDelete, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodDelete, "/connectors/{connector_ref}", opDelete)

	opUpdate := openapi3.Operation{}
	opUpdate.WithTags("connector")
	opUpdate.WithMapOfAnything(map[string]interface{}{"operationId": "updateConnector"})
	_ = reflector.SetRequest(&opUpdate, new(updateConnectorRequest), http.MethodPatch)
	_ = reflector.SetJSONResponse(&opUpdate, new(types.Connector), http.StatusOK)
	_ = reflector.SetJSONResponse(&opUpdate, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opUpdate, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opUpdate, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opUpdate, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opUpdate, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodPatch, "/connectors/{connector_ref}", opUpdate)
}
