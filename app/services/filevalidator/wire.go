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

package filevalidator

import (
	"github.com/harness/gitness/app/services/codeowners"

	"github.com/google/wire"
)

// The CODEOWNERS validator must satisfy the Validator interface.
var _ Validator = (*codeowners.ContentValidator)(nil)

var WireSet = wire.NewSet(
	ProvideRegistry,
)

// ProvideRegistry creates the content validator registry with all known validators.
func ProvideRegistry(codeOwners *codeowners.Service) (*Registry, error) {
	return NewRegistry(codeowners.NewContentValidator(codeOwners))
}
