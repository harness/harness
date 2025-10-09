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

import React, { useMemo } from 'react'
import { useParams } from 'react-router-dom'
import { PageBody } from '@harnessio/uicore'
import {
  ListWebhookExecutionsQueryQueryParams,
  useListWebhookExecutionsQuery
} from '@harnessio/react-har-service-client'

import { useStrings } from '@ar/frameworks/strings'
import { useGetSpaceRef, useParentHooks } from '@ar/hooks'
import { DEFAULT_PAGE_INDEX, PreferenceScope } from '@ar/constants'
import type { RepositoryWebhookDetailsTabPathParams } from '@ar/routes/types'

import WebhookExecutionListTable from './components/WebhookExecutionListTable/WebhookExecutionListTable'
import { useListWebhookExecutionsQueryParamOptions, type WebhookExecutionListPageQueryParams } from './utils'

export default function WebhookExecutionListPage() {
  const registryRef = useGetSpaceRef()
  const { getString } = useStrings()
  const { webhookIdentifier } = useParams<RepositoryWebhookDetailsTabPathParams>()
  const { useQueryParams, useUpdateQueryParams, usePreferenceStore } = useParentHooks()
  const { updateQueryParams } = useUpdateQueryParams<Partial<WebhookExecutionListPageQueryParams>>()

  const queryParamOptions = useListWebhookExecutionsQueryParamOptions()
  const queryParams = useQueryParams<WebhookExecutionListPageQueryParams>(queryParamOptions)
  const { preference: sortingPreference, setPreference: setSortingPreference } = usePreferenceStore<string | undefined>(
    PreferenceScope.USER,
    'ArtifactWebhookExecutionsSortingPreference'
  )
  const sort = useMemo(
    () => (sortingPreference ? JSON.parse(sortingPreference) : queryParams.sort),
    [queryParams.sort, sortingPreference]
  )

  const [sortField, sortOrder] = sort || []

  const { page, size } = queryParams

  const { data, isFetching, error, refetch } = useListWebhookExecutionsQuery({
    registry_ref: registryRef,
    webhook_identifier: webhookIdentifier,
    queryParams: {
      page,
      size,
      sort_field: sortField,
      sort_order: sortOrder
    } as ListWebhookExecutionsQueryQueryParams
  })

  const response = data?.content.data

  return (
    <PageBody
      loading={isFetching}
      error={error}
      retryOnError={() => refetch()}
      noData={{
        when: () => !response?.itemCount,
        icon: 'execution',
        noIconColor: true,
        messageTitle: getString('webhookList.table.noWebhooksTitle')
      }}>
      {response && (
        <WebhookExecutionListTable
          data={response}
          gotoPage={pageNumber => updateQueryParams({ page: pageNumber })}
          onPageSizeChange={newSize => updateQueryParams({ size: newSize, page: DEFAULT_PAGE_INDEX })}
          sortBy={sort}
          setSortBy={sortArray => {
            setSortingPreference(JSON.stringify(sortArray))
            updateQueryParams({ sort: sortArray, page: DEFAULT_PAGE_INDEX })
          }}
        />
      )}
    </PageBody>
  )
}
