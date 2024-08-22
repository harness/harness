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
import { defaultTo } from 'lodash-es'
import { Expander } from '@blueprintjs/core'
import {
  ExpandingSearchInput,
  HarnessDocTooltip,
  Page,
  type ExpandingSearchInputHandle,
  Button,
  ButtonVariation
} from '@harnessio/uicore'
import { useGetAllArtifactsQuery } from '@harnessio/react-har-service-client'

import { useStrings } from '@ar/frameworks/strings'
import { DEFAULT_PAGE_INDEX, PreferenceScope } from '@ar/constants'
import { useGetSpaceRef, useParentComponents, useParentHooks } from '@ar/hooks'
import PackageTypeSelector from '@ar/components/PackageTypeSelector/PackageTypeSelector'
import RepositorySelector from './components/RepositorySelector/RepositorySelector'
import ArtifactListTable from './components/ArtifactListTable/ArtifactListTable'
import LabelsSelector from './components/LabelsSelector/LabelsSelector'
import { useArtifactListQueryParamOptions, type ArtifactListPageQueryParams } from './utils'
import css from './ArtifactListPage.module.scss'

interface ArtifactListPageProps {
  withHeader?: boolean
  parentRepoKey?: string
  pageBodyClassName?: string
}

function ArtifactListPage({ withHeader = true, parentRepoKey, pageBodyClassName }: ArtifactListPageProps): JSX.Element {
  const { getString } = useStrings()
  const { NGBreadcrumbs } = useParentComponents()
  const { useQueryParams, useUpdateQueryParams, usePreferenceStore } = useParentHooks()
  const searchRef = useRef({} as ExpandingSearchInputHandle)
  const { updateQueryParams } = useUpdateQueryParams<Partial<ArtifactListPageQueryParams>>()
  const queryParams = useQueryParams<ArtifactListPageQueryParams>(useArtifactListQueryParamOptions())
  const { searchTerm, isDeployedArtifacts, packageTypes, repositoryKey, page, size, labels } = queryParams
  const shouldRenderRepositorySelectFilter = !parentRepoKey
  const shouldRenderPackageTypeSelectFilter = !parentRepoKey
  const spaceRef = useGetSpaceRef('')

  const { preference: sortingPreference, setPreference: setSortingPreference } = usePreferenceStore<string | undefined>(
    PreferenceScope.USER,
    'ArtifactRepositorySortingPreference'
  )
  const sort = useMemo(
    () => (sortingPreference ? JSON.parse(sortingPreference) : queryParams.sort),
    [queryParams.sort, sortingPreference]
  )

  const [sortField, sortOrder] = sort || []

  const {
    data,
    refetch,
    isLoading: loading,
    error
  } = useGetAllArtifactsQuery({
    space_ref: spaceRef,
    queryParams: {
      page,
      size,
      label: labels,
      package_type: packageTypes,
      search_term: searchTerm,
      sort_field: sortField,
      sort_order: sortOrder,
      reg_identifier: defaultTo(parentRepoKey, repositoryKey)
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
      packageTypes: [],
      isDeployedArtifacts: false
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

  const hasFilter = !!searchTerm || packageTypes.length || isDeployedArtifacts
  const responseData = data?.content?.data

  return (
    <>
      {withHeader && (
        <Page.Header
          title={
            <div className="ng-tooltip-native">
              <h2 data-tooltip-id="artifactsPageHeading">{getString('artifactList.pageHeading')}</h2>
              <HarnessDocTooltip tooltipId="artifactsPageHeading" useStandAlone={true} />
            </div>
          }
          breadcrumbs={<NGBreadcrumbs links={[]} />}
        />
      )}
      <Page.SubHeader className={css.subHeader}>
        <div className={css.subHeaderItems}>
          {shouldRenderRepositorySelectFilter && (
            <RepositorySelector
              value={repositoryKey}
              onChange={val => {
                updateQueryParams({ repositoryKey: val, page: DEFAULT_PAGE_INDEX })
              }}
            />
          )}
          {shouldRenderPackageTypeSelectFilter && (
            <PackageTypeSelector
              value={packageTypes}
              onChange={val => {
                updateQueryParams({ packageTypes: val, page: DEFAULT_PAGE_INDEX })
              }}
            />
          )}
          <LabelsSelector
            value={labels}
            onChange={val => {
              updateQueryParams({ labels: val, page: DEFAULT_PAGE_INDEX })
            }}
          />
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
        className={classNames(css.pageBody, pageBodyClassName)}
        loading={loading}
        error={error?.message}
        retryOnError={() => refetch()}
        noData={{
          when: () => !responseData?.artifacts?.length,
          // image: getEmptyStateIllustration(hasFilter, module),
          icon: 'container',
          messageTitle: hasFilter ? getString('noResultsFound') : getString('artifactList.table.noArtifactsTitle'),
          button: hasFilter ? (
            <Button text={getString('clearFilters')} variation={ButtonVariation.LINK} onClick={handleClearAllFilters} />
          ) : undefined
        }}>
        {responseData && (
          <ArtifactListTable
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

export default ArtifactListPage
