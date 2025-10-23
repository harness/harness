/*
 * Copyright 2024 Harness, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import React from 'react'
import { Page, Text } from '@harnessio/uicore'
import { FontVariation } from '@harnessio/design-system'
import type { Webhook } from '@harnessio/react-har-service-client'

import { useRoutes } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'
import Breadcrumbs from '@ar/components/Breadcrumbs/Breadcrumbs'
import { getIdentifierStringForBreadcrumb } from '@ar/common/utils'
import { RepositoryDetailsTab } from '@ar/pages/repository-details/constants'

interface WebhookDetailsPageHeaderProps {
  data: Webhook
  repositoryIdentifier: string
}

export function WebhookDetailsPageHeader(props: WebhookDetailsPageHeaderProps) {
  const { data, repositoryIdentifier } = props
  const routes = useRoutes()
  const { getString } = useStrings()
  return (
    <Page.Header
      title={
        <Text font={{ variation: FontVariation.H4 }} lineClamp={1}>
          {data.name}
        </Text>
      }
      size="large"
      breadcrumbs={
        <Breadcrumbs
          links={[
            {
              url: routes.toARRepositories(),
              label: getString('breadcrumbs.repositories')
            },
            {
              url: routes.toARRepositoryDetailsTab({
                repositoryIdentifier,
                tab: RepositoryDetailsTab.WEBHOOKS
              }),
              label: getIdentifierStringForBreadcrumb(getString('breadcrumbs.repositories'), repositoryIdentifier)
            }
          ]}
        />
      }
    />
  )
}
