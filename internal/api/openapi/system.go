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
	onGetConfig := openapi3.Operation{}
	onGetConfig.WithTags("system")
	onGetConfig.WithMapOfAnything(map[string]interface{}{"operationId": "getSystemConfig"})
	_ = reflector.SetRequest(&onGetConfig, nil, http.MethodGet)
	_ = reflector.SetJSONResponse(&onGetConfig, new(system.ConfigOutput), http.StatusOK)
	_ = reflector.SetJSONResponse(&onGetConfig, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&onGetConfig, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/system/config", onGetConfig)
}
