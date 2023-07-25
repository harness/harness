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
	onListConfigs := openapi3.Operation{}
	onListConfigs.WithTags("system")
	onListConfigs.WithMapOfAnything(map[string]interface{}{"operationId": "onListConfigs"})
	_ = reflector.SetRequest(&onListConfigs, nil, http.MethodGet)
	_ = reflector.SetJSONResponse(&onListConfigs, new(system.ConfigsOutput), http.StatusOK)
	_ = reflector.SetJSONResponse(&onListConfigs, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&onListConfigs, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.Spec.AddOperation(http.MethodGet, "/system/config", onListConfigs)
}
