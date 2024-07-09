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

	"github.com/harness/gitness/app/api/controller/infraprovider"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/types"

	"github.com/swaggest/openapi-go/openapi3"
)

type createInfraProviderConfigRequest struct {
	infraprovider.CreateInput
}

type getInfraProviderRequest struct {
	Ref string `path:"infraprovider_identifier"`
}

func infraProviderOperations(reflector *openapi3.Reflector) {
	opFind := openapi3.Operation{}
	opFind.WithTags("infraproviders")
	opFind.WithSummary("Get infraProviderConfig")
	opFind.WithMapOfAnything(map[string]interface{}{"operationId": "getInfraProvider"})
	_ = reflector.SetRequest(&opFind, new(getInfraProviderRequest), http.MethodGet)
	_ = reflector.SetJSONResponse(&opFind, new(types.InfraProviderConfig), http.StatusOK)
	_ = reflector.SetJSONResponse(&opFind, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opFind, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opFind, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(
		http.MethodGet, "/infraproviders/{infraprovider_identifier}", opFind)

	opCreate := openapi3.Operation{}
	opCreate.WithTags("infraproviders")
	opCreate.WithSummary("Create infraProvider config")
	opCreate.WithMapOfAnything(map[string]interface{}{"operationId": "createInfraProvider"})
	_ = reflector.SetRequest(&opCreate, new(createInfraProviderConfigRequest), http.MethodPost)
	_ = reflector.SetJSONResponse(&opCreate, new(types.InfraProviderConfig), http.StatusCreated)
	_ = reflector.SetJSONResponse(&opCreate, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opCreate, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opCreate, new(usererror.Error), http.StatusUnauthorized)
	_ = reflector.SetJSONResponse(&opCreate, new(usererror.Error), http.StatusForbidden)
	_ = reflector.Spec.AddOperation(http.MethodPost, "/infraproviders", opCreate)
}
