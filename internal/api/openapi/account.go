// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package openapi

import (
	"net/http"

	"github.com/harness/gitness/internal/api/usererror"
	"github.com/harness/gitness/types"

	"github.com/swaggest/openapi-go/openapi3"
)

// request to login to an account.
type loginRequest struct {
	Username string `formData:"username"`
	Password string `formData:"password"`
}

// request to register an account.
type registerRequest struct {
	Username string `formData:"username"`
	Password string `formData:"password"`
}

// helper function that constructs the openapi specification
// for the account registration and login endpoints.
func buildAccount(reflector *openapi3.Reflector) {
	onLogin := openapi3.Operation{}
	onLogin.WithTags("account")
	onLogin.WithMapOfAnything(map[string]interface{}{"operationId": "onLogin"})
	_ = reflector.SetRequest(&onLogin, new(loginRequest), http.MethodPost)
	_ = reflector.SetJSONResponse(&onLogin, new(types.TokenResponse), http.StatusOK)
	_ = reflector.SetJSONResponse(&onLogin, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&onLogin, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&onLogin, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodPost, "/login", onLogin)

	onRegister := openapi3.Operation{}
	onRegister.WithTags("account")
	onRegister.WithMapOfAnything(map[string]interface{}{"operationId": "onRegister"})
	_ = reflector.SetRequest(&onRegister, new(registerRequest), http.MethodPost)
	_ = reflector.SetJSONResponse(&onRegister, new(types.TokenResponse), http.StatusOK)
	_ = reflector.SetJSONResponse(&onRegister, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&onRegister, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.Spec.AddOperation(http.MethodPost, "/register", onRegister)
}
