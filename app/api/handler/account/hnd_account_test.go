// Copyright 2026 Harness, Inc.
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

package account

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	systemctrl "github.com/harness/gitness/app/api/controller/system"
	userctrl "github.com/harness/gitness/app/api/controller/user"

	"github.com/stretchr/testify/require"
)

func TestHndAccountHandlers_badBody(t *testing.T) {
	ctrl := &userctrl.Controller{}
	sysCtrl := &systemctrl.Controller{}
	for _, h := range []http.HandlerFunc{
		HandleLogin(ctrl, ""),
		HandleRegister(ctrl, sysCtrl, ""),
	} {
		r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{`))
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		require.Equal(t, http.StatusBadRequest, w.Code)
	}
}
