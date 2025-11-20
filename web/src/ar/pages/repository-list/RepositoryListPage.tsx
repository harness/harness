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
import {
  ExpandingSearchInput,
  HarnessDocTooltip,
  Page,
  Button,
  ButtonVariation,
  GridListToggle,
  Views
} from '@harnessio/uicore'
import type { ExpandingSearchInputHandle } from '@harnessio/uicore'

import { useStrings } from '@ar/frameworks/strings'
import { DEFAULT_PAGE_INDEX, PreferenceScope } from '@ar/constants'
import { RepositoryListViewTypeEnum } from '@ar/contexts/AppStoreContext'
import { EntityScope, Parent, RepositoryScopeType } from '@ar/common/types'
import useGetPageScope from '@ar/hooks/useGetPageScope'
import { useParentHooks, useAppStore, useGetRepositoryListViewType, useFeatureFlags } from '@ar/hooks'
import Breadcrumbs from '@ar/components/Breadcrumbs/Breadcrumbs'
import PackageTypeSelector from '@ar/components/PackageTypeSelector/PackageTypeSelector'
import MetadataFilterSelector from '@ar/components/MetadataFilterSelector/MetadataFilterSelector'
import useMetadatadataFilterFromQuery from '@ar/components/MetadataFilterSelector/useMetadataFilterFromQuery'

import { CreateRepository } from './components/CreateRepository/CreateRepository'
import { RepositoryListTable } from './components/RepositoryListTable'
import { useArtifactRepositoriesQueryParamOptions } from './utils'
import type { ArtifactRepositoryListPageQueryParams } from './utils'
import useLocalGetRegistriesQuery from './hooks/useLocalGetRegistriesQuery'
import RepositoryTypeSelector from './components/RepositoryTypeSelector/RepositoryTypeSelector'
import RepositoryScopeSelector from './components/RepositoryScopeSelector/RepositoryScopeSelector'

import css from './RepositoryListPage.module.scss'

function RepositoryListPage(): JSX.Element {
  const searchRef = useRef({} as ExpandingSearchInputHandle)
  const { getString } = useStrings()
  const { useQueryParams, useUpdateQueryParams, usePreferenceStore } = useParentHooks()
  const { updateQueryParams, replaceQueryParams } =
    useUpdateQueryParams<Partial<ArtifactRepositoryListPageQueryParams>>()
  const { setRepositoryListViewType, parent } = useAppStore()
  const { getValue, updateValue } = useMetadatadataFilterFromQuery()
  const metadataFilter = getValue()
  const repositoryListViewType = useGetRepositoryListViewType()
  const { HAR_CUSTOM_METADATA_ENABLED } = useFeatureFlags()

  const queryParamOptions = useArtifactRepositoriesQueryParamOptions()
  const queryParams = useQueryParams<ArtifactRepositoryListPageQueryParams>(queryParamOptions)
  const { searchTerm, page, size, repositoryTypes, configType, scope } = queryParams

  const pageScope = useGetPageScope()

  const { preference: sortingPreference, setPreference: setSortingPreference } = usePreferenceStore<string | undefined>(
    PreferenceScope.USER,
    'RegistryListSortingPreference'
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
  } = useLocalGetRegistriesQuery({
    page,
    size,
    sort_field: sortField,
    sort_order: sortOrder,
    package_type: repositoryTypes,
    search_term: searchTerm,
    type: configType,
    scope: scope
  })

  const handleClearFilters = (): void => {
    flushSync(searchRef.current.clear)
    replaceQueryParams({
      page: undefined,
      searchTerm: undefined,
      repositoryTypes: undefined,
      configType: undefined,
      scope: undefined
    })
  }

  const hasFilter = !!searchTerm || repositoryTypes?.length || configType?.length || metadataFilter.length

  const responseData = data?.content.data

  return (
    <>
      <Page.Header
        className={css.pageHeader}
        title={
          <div className="ng-tooltip-native">
            <h2 data-tooltip-id="artifactRepositoriesPageHeading">{getString('repositoryList.pageHeading')}</h2>
            <HarnessDocTooltip tooltipId="artifactRepositoriesPageHeading" useStandAlone={true} />
          </div>
        }
        breadcrumbs={<Breadcrumbs links={[]} />}
      />
      <Page.SubHeader className={css.subHeader}>
        <div className={css.subHeaderItems}>
          <CreateRepository />
          <RepositoryTypeSelector
            value={configType}
            onChange={val => {
              updateQueryParams({ configType: val, page: DEFAULT_PAGE_INDEX })
            }}
          />
          <PackageTypeSelector
            value={repositoryTypes}
            onChange={val => {
              updateQueryParams({ repositoryTypes: val, page: DEFAULT_PAGE_INDEX })
            }}
          />
          {parent === Parent.Enterprise && pageScope !== EntityScope.PROJECT && (
            <RepositoryScopeSelector
              scope={pageScope}
              value={scope}
              onChange={val => {
                updateQueryParams({
                  scope: val,
                  page: DEFAULT_PAGE_INDEX
                })
              }}
            />
          )}
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
          <GridListToggle
            initialSelectedView={repositoryListViewType === RepositoryListViewTypeEnum.LIST ? Views.LIST : Views.GRID}
            icons={{ left: 'SplitView' }}
            onViewToggle={newView => {
              if (newView === Views.LIST) return
              setRepositoryListViewType(RepositoryListViewTypeEnum.DIRECTORY)
            }}
          />
        </div>
      </Page.SubHeader>
      <Page.Body
        className={css.pageBody}
        loading={loading}
        error={error?.message}
        retryOnError={() => refetch()}
        noData={{
          when: () => !responseData?.registries?.length, // TODO: change to itemCount once BE fixes the issue with paginated response
          icon: 'registry',
          // image: getEmptyStateIllustration(hasFilter, module),
          messageTitle: hasFilter ? getString('noResultsFound') : getString('repositoryList.table.noRepositoriesTitle'),
          button: hasFilter ? (
            <Button text={getString('clearFilters')} variation={ButtonVariation.LINK} onClick={handleClearFilters} />
          ) : (
            <CreateRepository />
          )
        }}>
        {responseData && (
          <RepositoryListTable
            gotoPage={pageNumber => updateQueryParams({ page: pageNumber })}
            onPageSizeChange={newSize => updateQueryParams({ size: newSize, page: DEFAULT_PAGE_INDEX })}
            data={responseData}
            refetchList={refetch}
            setSortBy={sortArray => {
              setSortingPreference(JSON.stringify(sortArray))
              updateQueryParams({ sort: sortArray, page: DEFAULT_PAGE_INDEX })
            }}
            sortBy={sort}
            showScope={scope !== RepositoryScopeType.NONE}
          />
        )}
      </Page.Body>
    </>
  )
}

export default RepositoryListPage
