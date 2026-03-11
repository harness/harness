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

import React, { useRef } from 'react'
import { flushSync } from 'react-dom'
import classNames from 'classnames'
import {
  ExpandingSearchInput,
  ExpandingSearchInputHandle,
  Layout,
  Page,
  Button,
  ButtonVariation
} from '@harnessio/uicore'

import { useListFilesV3Query, type ListFilesResponseBodyV3 } from '@harnessio/react-har-service-client'

import { useParentHooks } from '@ar/hooks'
import { useAppStore } from '@ar/hooks/useAppStore'
import { DEFAULT_PAGE_INDEX } from '@ar/constants'
import { useStrings } from '@ar/frameworks/strings'

import { useArtifactFileListQueryParamOptions, type ArtifactFileListPageQueryParams } from './utils'
import ArtifactFileListV3Table from './ArtifactFileListV3Table'

export interface ArtifactFileListV3PageProps {
  registryId?: string
  packageId?: string
  versionId?: string
  repositoryIdentifier: string
  pageBodyClassName?: string
  wrapperClassName?: string
}

export default function ArtifactFileListV3Page(props: ArtifactFileListV3PageProps): React.ReactElement {
  const { registryId, packageId, versionId, repositoryIdentifier, pageBodyClassName } = props
  const { getString } = useStrings()
  const searchRef = useRef<ExpandingSearchInputHandle | null>(null)
  const { scope } = useAppStore()
  const { useQueryParams, useUpdateQueryParams } = useParentHooks()
  const { updateQueryParams, replaceQueryParams } = useUpdateQueryParams<Partial<ArtifactFileListPageQueryParams>>()
  const queryParamOptions = useArtifactFileListQueryParamOptions()
  const queryParams = useQueryParams<ArtifactFileListPageQueryParams>(queryParamOptions)
  const { page, size, sort, searchTerm } = queryParams

  const {
    data,
    isFetching: loading,
    error,
    refetch
  } = useListFilesV3Query(
    {
      queryParams: {
        account_identifier: scope?.accountId ?? '',
        org_identifier: scope?.orgIdentifier,
        project_identifier: scope?.projectIdentifier,
        registry_id: registryId,
        package_id: packageId,
        version_id: versionId,
        page: page,
        size: size,
        sort: sort?.join(':'),
        search_term: searchTerm
      }
    },
    { enabled: !!registryId }
  )

  const resolvedData: ListFilesResponseBodyV3 | undefined = data?.content

  const hasFilter = !!searchTerm

  const handleClearFilters = (): void => {
    flushSync(() => searchRef.current?.clear?.())
    replaceQueryParams({ page: DEFAULT_PAGE_INDEX })
  }

  return (
    <Layout.Vertical margin="large" spacing="medium">
      <ExpandingSearchInput
        ref={searchRef}
        alwaysExpanded
        width={400}
        placeholder={getString('search')}
        onChange={text => updateQueryParams({ searchTerm: text || undefined, page: DEFAULT_PAGE_INDEX })}
        defaultValue={searchTerm ?? ''}
      />
      <Page.Body
        className={classNames(pageBodyClassName)}
        loading={loading}
        error={error?.error?.message}
        retryOnError={() => refetch()}
        noData={{
          when: () => !resolvedData?.items?.length,
          icon: 'document',
          messageTitle: hasFilter
            ? getString('noResultsFound')
            : getString('versionDetails.artifactFiles.noFilesTitle'),
          button: hasFilter ? (
            <Button text={getString('clearFilters')} variation={ButtonVariation.LINK} onClick={handleClearFilters} />
          ) : undefined
        }}>
        {resolvedData ? (
          <ArtifactFileListV3Table
            data={resolvedData}
            setSortBy={sortArr => updateQueryParams({ sort: sortArr, page: DEFAULT_PAGE_INDEX })}
            sortBy={sort || []}
            onPageChange={pageNumber => updateQueryParams({ page: pageNumber })}
            onPageSizeChange={newSize => updateQueryParams({ size: newSize, page: DEFAULT_PAGE_INDEX })}
            repositoryIdentifier={repositoryIdentifier}
          />
        ) : null}
      </Page.Body>
    </Layout.Vertical>
  )
}
