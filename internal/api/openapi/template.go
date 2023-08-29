// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package openapi

import (
	"net/http"

	"github.com/harness/gitness/internal/api/controller/template"
	"github.com/harness/gitness/internal/api/usererror"
	"github.com/harness/gitness/types"

	"github.com/swaggest/openapi-go/openapi3"
)

type createTemplateRequest struct {
	template.CreateInput
}

type templateRequest struct {
	Ref string `path:"template_ref"`
}

type getTemplateRequest struct {
	templateRequest
}

type updateTemplateRequest struct {
	templateRequest
	template.UpdateInput
}

func templateOperations(reflector *openapi3.Reflector) {
	opCreate := openapi3.Operation{}
	opCreate.WithTags("template")
	opCreate.WithMapOfAnything(map[string]interface{}{"operationId": "createTemplate"})
	_ = reflector.SetRequest(&opCreate, new(createTemplateRequest), http.MethodPost)
	_ = reflector.SetJSONResponse(&opCreate, new(types.Template), http.StatusCreated)
	_ = reflector.SetJSONResponse(&opCreate, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opCreate, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opCreate, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opCreate, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodPost, "/templates", opCreate)

	opFind := openapi3.Operation{}
	opFind.WithTags("template")
	opFind.WithMapOfAnything(map[string]interface{}{"operationId": "findTemplate"})
	_ = reflector.SetRequest(&opFind, new(getTemplateRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&opFind, new(types.Template), http.StatusOK)
	_ = reflector.SetJSONResponse(&opFind, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opFind, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opFind, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opFind, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/templates/{template_ref}", opFind)

	opDelete := openapi3.Operation{}
	opDelete.WithTags("template")
	opDelete.WithMapOfAnything(map[string]interface{}{"operationId": "deleteTemplate"})
	_ = reflector.SetRequest(&opDelete, new(getTemplateRequest), http.MethodDelete)
	_ = reflector.SetJSONResponse(&opDelete, nil, http.StatusNoContent)
	_ = reflector.SetJSONResponse(&opDelete, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opDelete, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opDelete, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opDelete, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodDelete, "/templates/{template_ref}", opDelete)

	opUpdate := openapi3.Operation{}
	opUpdate.WithTags("template")
	opUpdate.WithMapOfAnything(map[string]interface{}{"operationId": "updateTemplate"})
	_ = reflector.SetRequest(&opUpdate, new(updateTemplateRequest), http.MethodPatch)
	_ = reflector.SetJSONResponse(&opUpdate, new(types.Template), http.StatusOK)
	_ = reflector.SetJSONResponse(&opUpdate, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opUpdate, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opUpdate, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opUpdate, new(usererror.Error), http.StatusForbidden)
	_ = reflector.SetJSONResponse(&opUpdate, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodPatch, "/templates/{template_ref}", opUpdate)
}
