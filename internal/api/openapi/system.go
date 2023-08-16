// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package openapi

import (
	"net/http"

	"github.com/harness/gitness/internal/api/handler/system"
	"github.com/harness/gitness/internal/api/usererror"

	"github.com/swaggest/openapi-go/openapi3"
)

// helper function that constructs the openapi specification
// for the system registration config endpoints.
func buildSystem(reflector *openapi3.Reflector) {
	opGetConfig := openapi3.Operation{}
	opGetConfig.WithTags("system")
	opGetConfig.WithMapOfAnything(map[string]interface{}{"operationId": "getSystemConfig"})
	_ = reflector.SetRequest(&opGetConfig, nil, http.MethodGet)
	_ = reflector.SetJSONResponse(&opGetConfig, new(system.ConfigOutput), http.StatusOK)
	_ = reflector.SetJSONResponse(&opGetConfig, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opGetConfig, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/system/config", opGetConfig)
}
