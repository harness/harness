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

import React, { useCallback, useMemo, useRef } from 'react'
import classNames from 'classnames'
import { flushSync } from 'react-dom'
import { Expander } from '@blueprintjs/core'
import { ExpandingSearchInput, Page, type ExpandingSearchInputHandle, Button, ButtonVariation } from '@harnessio/uicore'
import type { ArtifactType } from '@harnessio/react-har-service-client'

import { Parent } from '@ar/common/types'
import { useStrings } from '@ar/frameworks/strings'
import { DEFAULT_PAGE_INDEX, PreferenceScope } from '@ar/constants'
import { useAppStore, useFeatureFlags, useParentHooks } from '@ar/hooks'
import MetadataFilterSelector from '@ar/components/MetadataFilterSelector/MetadataFilterSelector'
import useMetadatadataFilterFromQuery from '@ar/components/MetadataFilterSelector/useMetadataFilterFromQuery'

import useLocalGetAllArtifactsByRegistryQuery from './hooks/useLocalGetAllArtifactsByRegistryQuery'
import RegistryArtifactListTable from './components/RegistryArtifactListTable/RegistryArtifactListTable'
import {
  useRegistryArtifactListQueryParamOptions,
  type RegistryArtifactListPageQueryParams
} from './components/RegistryArtifactListTable/utils'

import css from './ArtifactListPage.module.scss'

interface RegistryArtifactListPageProps {
  pageBodyClassName?: string
  artifactType?: ArtifactType
}

function RegistryArtifactListPage({ pageBodyClassName, artifactType }: RegistryArtifactListPageProps): JSX.Element {
  const { getString } = useStrings()
  const { parent } = useAppStore()
  const { HAR_CUSTOM_METADATA_ENABLED } = useFeatureFlags()
  const { useQueryParams, useUpdateQueryParams, usePreferenceStore } = useParentHooks()
  const searchRef = useRef({} as ExpandingSearchInputHandle)
  const { updateQueryParams, replaceQueryParams } = useUpdateQueryParams<Partial<RegistryArtifactListPageQueryParams>>()
  const queryParams = useQueryParams<RegistryArtifactListPageQueryParams>(useRegistryArtifactListQueryParamOptions())
  const { searchTerm, isDeployedArtifacts, packageTypes, page, size, labels } = queryParams

  const { getValue, updateValue } = useMetadatadataFilterFromQuery()
  const metadataFilter = getValue()

  const { preference: sortingPreference, setPreference: setSortingPreference } = usePreferenceStore<string | undefined>(
    PreferenceScope.USER,
    'RegistryImageListSortingPreference'
  )
  const sort = useMemo(
    () => (sortingPreference ? JSON.parse(sortingPreference) : queryParams.sort),
    [queryParams.sort, sortingPreference]
  )

  const [sortField, sortOrder] = sort || []

  const {
    data,
    refetch,
    isFetching: loading,
    error
  } = useLocalGetAllArtifactsByRegistryQuery({
    page,
    size,
    search_term: searchTerm,
    sort_field: sortField,
    sort_order: sortOrder,
    artifact_type: artifactType
  })

  const handleClearAllFilters = (): void => {
    flushSync(searchRef.current.clear)
    replaceQueryParams({
      page: undefined,
      searchTerm: undefined,
      packageTypes: undefined,
      isDeployedArtifacts: undefined
    })
  }

  const handleClickLabel = useCallback(
    (val: string) => {
      if (labels.includes(val)) return
      updateQueryParams({
        labels: [...labels, val],
        page: DEFAULT_PAGE_INDEX
      })
    },
    [labels]
  )

  const hasFilter = !!searchTerm || packageTypes?.length || isDeployedArtifacts || metadataFilter.length
  const responseData = data?.content?.data

  return (
    <>
      <Page.SubHeader className={css.subHeader}>
        <div className={css.subHeaderItems}>
          {HAR_CUSTOM_METADATA_ENABLED && parent === Parent.Enterprise && (
            <MetadataFilterSelector
              value={metadataFilter}
              onSubmit={val => {
                updateValue(val)
                updateQueryParams({
                  page: DEFAULT_PAGE_INDEX
                })
              }}
            />
          )}
          <Expander />
          <ExpandingSearchInput
            alwaysExpanded
            width={200}
            placeholder={getString('search')}
            onChange={text => {
              updateQueryParams({ searchTerm: text || undefined, page: DEFAULT_PAGE_INDEX })
            }}
            defaultValue={searchTerm}
            ref={searchRef}
          />
        </div>
      </Page.SubHeader>
      <Page.Body
        className={classNames(css.pageBody, pageBodyClassName)}
        loading={loading}
        error={error?.message}
        retryOnError={() => refetch()}
        noData={{
          when: () => !responseData?.packages?.length,
          // image: getEmptyStateIllustration(hasFilter, module),
          icon: 'store-artifact-bundle',
          messageTitle: hasFilter ? getString('noResultsFound') : getString('artifactList.table.noArtifactsTitle'),
          button: hasFilter ? (
            <Button text={getString('clearFilters')} variation={ButtonVariation.LINK} onClick={handleClearAllFilters} />
          ) : undefined
        }}>
        {responseData && (
          <RegistryArtifactListTable
            data={responseData}
            gotoPage={pageNumber => updateQueryParams({ page: pageNumber })}
            onPageSizeChange={newSize => updateQueryParams({ size: newSize, page: DEFAULT_PAGE_INDEX })}
            refetchList={() => {
              refetch()
            }}
            setSortBy={sortArray => {
              setSortingPreference(JSON.stringify(sortArray))
              updateQueryParams({ sort: sortArray })
            }}
            onClickLabel={handleClickLabel}
            sortBy={sort}
          />
        )}
      </Page.Body>
    </>
  )
}

export default RegistryArtifactListPage
