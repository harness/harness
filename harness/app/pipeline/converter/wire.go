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

package converter

import (
	"github.com/harness/gitness/app/pipeline/file"
	"github.com/harness/gitness/app/services/publicaccess"

	"github.com/google/wire"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	ProvideService,
)

// ProvideService provides a service which can convert templates.
func ProvideService(fileService file.Service, publicAccess publicaccess.Service) Service {
	return newConverter(fileService, publicAccess)
}
