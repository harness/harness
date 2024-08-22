// Source: https://github.com/goharbor/harbor

// Copyright 2016 Project Harbor Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package null

import (
	"net/http"

	"github.com/harness/gitness/registry/app/common/lib"
)

// NewAuthorizer returns a null authorizer.
func NewAuthorizer() lib.Authorizer {
	return &authorizer{}
}

type authorizer struct{}

func (a *authorizer) Modify(_ *http.Request) error {
	// do nothing
	return nil
}
