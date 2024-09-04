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
import { Expander } from '@blueprintjs/core'
import { Button, ButtonVariation, ExpandingSearchInput, ExpandingSearchInputHandle, Page } from '@harnessio/uicore'
import { PackageType, useGetAllArtifactVersionsQuery } from '@harnessio/react-har-service-client'

import { useStrings } from '@ar/frameworks/strings'
import { encodeRef } from '@ar/hooks/useGetSpaceRef'
import { useParentHooks, useDecodedParams, useGetSpaceRef } from '@ar/hooks'
import type { RepositoryPackageType } from '@ar/common/types'
import type { ArtifactDetailsPathParams } from '@ar/routes/types'
import { DEFAULT_PAGE_INDEX, PreferenceScope } from '@ar/constants'
import VersionListTableWidget from '@ar/frameworks/Version/VersionListTableWidget'

import { VersionListPageQueryParams, useVersionListQueryParamOptions } from './utils'

import css from './VersionListPage.module.scss'

interface VersionListPageProps {
  packageType: PackageType
}

function VersionListPage(props: VersionListPageProps): JSX.Element {
  const { packageType } = props
  const pathParams = useDecodedParams<ArtifactDetailsPathParams>()
  const { useQueryParams, useUpdateQueryParams, usePreferenceStore } = useParentHooks()
  const searchRef = useRef({} as ExpandingSearchInputHandle)
  const { updateQueryParams } = useUpdateQueryParams<Partial<VersionListPageQueryParams>>()
  const queryParams = useQueryParams<VersionListPageQueryParams>(useVersionListQueryParamOptions())
  const { searchTerm, isDeployedArtifacts, page, size } = queryParams
  const { getString } = useStrings()
  const spaceRef = useGetSpaceRef()

  const { preference: sortingPreference, setPreference: setSortingPreference } = usePreferenceStore<string | undefined>(
    PreferenceScope.USER,
    'ArtifactVersionsSortingPreference'
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
  } = useGetAllArtifactVersionsQuery({
    registry_ref: spaceRef,
    artifact: encodeRef(pathParams.artifactIdentifier),
    queryParams: {
      page,
      size,
      sort_field: sortField,
      sort_order: sortOrder,
      search_term: searchTerm
    },
    stringifyQueryParamsOptions: {
      arrayFormat: 'repeat'
    }
  })

  const handleClearAllFilters = (): void => {
    flushSync(searchRef.current.clear)
    updateQueryParams({
      page: 0,
      searchTerm: '',
      isDeployedArtifacts: false
    })
  }

  const hasFilter = !!searchTerm || isDeployedArtifacts

  const responseData = data?.content.data

  return (
    <>
      <Page.SubHeader className={css.subHeader}>
        <div className={css.subHeaderItems}>
          {/* TODO: removed till BE support this filter */}
          {/* <TableFilterCheckbox
            value={isDeployedArtifacts}
            label={getString('artifactList.deployedArtifacts')}
            disabled={false}
            onChange={val => {
              updateQueryParams({ isDeployedArtifacts: val, page: DEFAULT_PAGE_INDEX })
            }}
          /> */}
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
        className={css.pageBody}
        loading={loading}
        error={error?.message}
        retryOnError={() => refetch()}
        noData={{
          when: () => !responseData?.artifactVersions?.length,
          // image: getEmptyStateIllustration(hasFilter, module),
          messageTitle: hasFilter ? getString('noResultsFound') : getString('versionList.table.noVersionsTitle'),
          button: hasFilter ? (
            <Button text={getString('clearFilters')} variation={ButtonVariation.LINK} onClick={handleClearAllFilters} />
          ) : undefined
        }}>
        {responseData && (
          <VersionListTableWidget
            packageType={packageType as RepositoryPackageType}
            data={responseData}
            gotoPage={pageNumber => updateQueryParams({ page: pageNumber })}
            onPageSizeChange={newSize => updateQueryParams({ size: newSize, page: 0 })}
            setSortBy={sortArray => {
              setSortingPreference(JSON.stringify(sortArray))
              updateQueryParams({ sort: sortArray })
            }}
            sortBy={sort}
          />
        )}
      </Page.Body>
    </>
  )
}

export default VersionListPage
