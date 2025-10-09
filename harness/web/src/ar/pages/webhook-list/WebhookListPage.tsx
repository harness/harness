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

import React, { useMemo, useRef } from 'react'
import { flushSync } from 'react-dom'
import { useParams } from 'react-router-dom'
import { Expander } from '@blueprintjs/core'
import { useListWebhooksQuery } from '@harnessio/react-har-service-client'
import { Button, ButtonVariation, ExpandingSearchInput, ExpandingSearchInputHandle, Page } from '@harnessio/uicore'

import { DEFAULT_PAGE_INDEX, PreferenceScope } from '@ar/constants'
import { useStrings } from '@ar/frameworks/strings'
import type { RepositoryDetailsTabPathParams } from '@ar/routes/types'
import { useAppStore, useGetSpaceRef, useParentHooks } from '@ar/hooks'
import { PermissionIdentifier, ResourceType } from '@ar/common/permissionTypes'

import WebhookListTable from './components/WebhookListTable/WebhookListTable'
import CreateWebhookButton from './components/CreateWebhookButton/CreateWebhookButton'
import { useListWebhooksQueryParamOptions, type WebhookListPageQueryParams } from './utils'

import css from './WebhookListPage.module.scss'

export default function WebhookListPage() {
  const { getString } = useStrings()
  const registryRef = useGetSpaceRef()
  const searchRef = useRef({} as ExpandingSearchInputHandle)

  const { scope } = useAppStore()
  const { repositoryIdentifier } = useParams<RepositoryDetailsTabPathParams>()
  const { useQueryParams, useUpdateQueryParams, usePermission, usePreferenceStore } = useParentHooks()
  const { updateQueryParams } = useUpdateQueryParams<Partial<WebhookListPageQueryParams>>()

  const queryParamOptions = useListWebhooksQueryParamOptions()
  const queryParams = useQueryParams<WebhookListPageQueryParams>(queryParamOptions)

  const { page, size, searchTerm } = queryParams
  const { accountId, orgIdentifier, projectIdentifier } = scope

  const { preference: sortingPreference, setPreference: setSortingPreference } = usePreferenceStore<string | undefined>(
    PreferenceScope.USER,
    'ArtifactWebhooksSortingPreference'
  )
  const sort = useMemo(
    () => (sortingPreference ? JSON.parse(sortingPreference) : queryParams.sort),
    [queryParams.sort, sortingPreference]
  )

  const [sortField, sortOrder] = sort || []

  const [isEdit] = usePermission(
    {
      resourceScope: {
        accountIdentifier: accountId,
        orgIdentifier,
        projectIdentifier
      },
      resource: {
        resourceType: ResourceType.ARTIFACT_REGISTRY,
        resourceIdentifier: repositoryIdentifier
      },
      permissions: [PermissionIdentifier.EDIT_ARTIFACT_REGISTRY]
    },
    [accountId, projectIdentifier, orgIdentifier, repositoryIdentifier]
  )

  const { isFetching, data, error, refetch } = useListWebhooksQuery({
    registry_ref: registryRef,
    queryParams: {
      page,
      size,
      search_term: searchTerm,
      sort_field: sortField,
      sort_order: sortOrder
    }
  })

  const handleClearFilters = (): void => {
    flushSync(searchRef.current.clear)
    updateQueryParams({
      page: undefined,
      size: undefined,
      searchTerm: undefined
    })
  }

  const response = data?.content.data

  const hasFilter = !!searchTerm
  return (
    <>
      <Page.SubHeader>
        <CreateWebhookButton />
        <Expander />
        <ExpandingSearchInput
          alwaysExpanded
          width={200}
          ref={searchRef}
          placeholder={getString('search')}
          defaultValue={searchTerm ?? ''}
          onChange={text => {
            updateQueryParams({ searchTerm: text || undefined, page: DEFAULT_PAGE_INDEX })
          }}
        />
      </Page.SubHeader>
      <Page.Body
        className={css.pageBody}
        loading={isFetching}
        error={error?.message || error}
        retryOnError={() => refetch()}
        noData={{
          when: () => !response?.itemCount,
          icon: 'code-webhook',
          noIconColor: true,
          messageTitle: hasFilter ? getString('noResultsFound') : getString('webhookList.table.noWebhooksTitle'),
          button: hasFilter ? (
            <Button text={getString('clearFilters')} variation={ButtonVariation.LINK} onClick={handleClearFilters} />
          ) : (
            <CreateWebhookButton />
          )
        }}>
        {response && (
          <WebhookListTable
            data={response}
            refetchList={refetch}
            gotoPage={pageNumber => updateQueryParams({ page: pageNumber })}
            onPageSizeChange={newSize => updateQueryParams({ size: newSize, page: DEFAULT_PAGE_INDEX })}
            sortBy={sort}
            setSortBy={sortArray => {
              setSortingPreference(JSON.stringify(sortArray))
              updateQueryParams({ sort: sortArray, page: DEFAULT_PAGE_INDEX })
            }}
            readonly={!isEdit}
          />
        )}
      </Page.Body>
    </>
  )
}
