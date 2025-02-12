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

package webhook

import (
	"github.com/harness/gitness/app/auth/authz"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/services/webhook"
	"github.com/harness/gitness/encrypt"

	"github.com/google/wire"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	ProvideController,
)

func ProvideController(authorizer authz.Authorizer,
	spaceFinder refcache.SpaceFinder, repoFinder refcache.RepoFinder,
	webhookService *webhook.Service, encrypter encrypt.Encrypter,
	preprocessor Preprocessor,
) *Controller {
	return NewController(
		authorizer, spaceFinder, repoFinder, webhookService, encrypter, preprocessor)
}

func ProvidePreprocessor() Preprocessor {
	return NoopPreprocessor{}
}
