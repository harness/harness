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

	"github.com/harness/gitness/app/api/controller/user"
	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/types"

	"github.com/gotidy/ptr"
	"github.com/swaggest/openapi-go/openapi3"
)

var queryParameterIncludeCookie = openapi3.ParameterOrRef{
	Parameter: &openapi3.Parameter{
		Name:        request.QueryParamIncludeCookie,
		In:          openapi3.ParameterInQuery,
		Description: ptr.String("If set to true the token is also returned as a cookie."),
		Required:    ptr.Bool(false),
		Schema: &openapi3.SchemaOrRef{
			Schema: &openapi3.Schema{
				Type:    ptrSchemaType(openapi3.SchemaTypeBoolean),
				Default: ptrptr(false),
			},
		},
	},
}

// request to login to an account.
type loginRequest struct {
	user.LoginInput
}

// request to register an account.
type registerRequest struct {
	user.RegisterInput
}

// helper function that constructs the openapi specification
// for the account registration and login endpoints.
func buildAccount(reflector *openapi3.Reflector) {
	onLogin := openapi3.Operation{}
	onLogin.WithTags("account")
	onLogin.WithParameters(queryParameterIncludeCookie)
	onLogin.WithMapOfAnything(map[string]interface{}{"operationId": "onLogin"})
	_ = reflector.SetRequest(&onLogin, new(loginRequest), http.MethodPost)
	_ = reflector.SetJSONResponse(&onLogin, new(types.TokenResponse), http.StatusOK)
	_ = reflector.SetJSONResponse(&onLogin, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&onLogin, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&onLogin, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodPost, "/login", onLogin)

	opLogout := openapi3.Operation{}
	opLogout.WithTags("account")
	opLogout.WithMapOfAnything(map[string]interface{}{"operationId": "opLogout"})
	_ = reflector.SetRequest(&opLogout, nil, http.MethodPost)
	_ = reflector.SetJSONResponse(&opLogout, nil, http.StatusOK)
	_ = reflector.SetJSONResponse(&opLogout, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.SetJSONResponse(&opLogout, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&opLogout, new(usererror.Error), http.StatusNotFound)
	_ = reflector.Spec.AddOperation(http.MethodPost, "/logout", opLogout)

	onRegister := openapi3.Operation{}
	onRegister.WithTags("account")
	onRegister.WithParameters(queryParameterIncludeCookie)
	onRegister.WithMapOfAnything(map[string]interface{}{"operationId": "onRegister"})
	_ = reflector.SetRequest(&onRegister, new(registerRequest), http.MethodPost)
	_ = reflector.SetJSONResponse(&onRegister, new(types.TokenResponse), http.StatusOK)
	_ = reflector.SetJSONResponse(&onRegister, new(usererror.Error), http.StatusInternalServerError)
	_ = reflector.SetJSONResponse(&onRegister, new(usererror.Error), http.StatusBadRequest)
	_ = reflector.Spec.AddOperation(http.MethodPost, "/register", onRegister)
}
