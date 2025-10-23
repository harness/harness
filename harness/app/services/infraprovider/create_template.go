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

package infraprovider

import (
	"context"

	"github.com/harness/gitness/infraprovider"
	"github.com/harness/gitness/types"

	"github.com/rs/zerolog/log"
)

func (c *Service) CreateTemplate(
	ctx context.Context,
	template *types.InfraProviderTemplate,
) error {
	return c.infraProviderTemplateStore.Create(ctx, template)
}

func (c *Service) validateTemplates(
	ctx context.Context,
	infraProvider infraprovider.InfraProvider,
	res types.InfraProviderResource,
) error {
	templateParams := infraProvider.TemplateParams()
	for _, param := range templateParams {
		key := param.Name
		if res.Metadata[key] != "" {
			templateIdentifier := res.Metadata[key]
			_, err := c.infraProviderTemplateStore.FindByIdentifier(
				ctx, res.SpaceID, templateIdentifier)
			if err != nil {
				log.Warn().Msgf("unable to get template params for ID : %s",
					res.Metadata[key])
			}
		}
	}
	return nil
}
