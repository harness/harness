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
import classNames from 'classnames'
import { Expander } from '@blueprintjs/core'
import { HarnessDocTooltip, Page, Button, ButtonVariation } from '@harnessio/uicore'
import {
  GetAllHarnessArtifactsQueryQueryParams,
  useGetAllHarnessArtifactsQuery
} from '@harnessio/react-har-service-client'

import { useStrings } from '@ar/frameworks/strings'
import { DEFAULT_PAGE_INDEX, PreferenceScope } from '@ar/constants'
import { ButtonTab, ButtonTabs } from '@ar/components/ButtonTabs/ButtonTabs'
import { useGetSpaceRef, useParentComponents, useParentHooks } from '@ar/hooks'
import PackageTypeSelector from '@ar/components/PackageTypeSelector/PackageTypeSelector'

import { ArtifactListVersionFilter } from './constants'
import LabelsSelector from './components/LabelsSelector/LabelsSelector'
import ArtifactListTable from './components/ArtifactListTable/ArtifactListTable'
import RepositorySelector from './components/RepositorySelector/RepositorySelector'
import ArtifactSearchInput from './components/ArtifactSearchInput/ArtifactSearchInput'
import { useArtifactListQueryParamOptions, type ArtifactListPageQueryParams } from './utils'

import css from './ArtifactListPage.module.scss'

function ArtifactListPage(): JSX.Element {
  const { getString } = useStrings()
  const { NGBreadcrumbs } = useParentComponents()
  const { useQueryParams, useUpdateQueryParams, usePreferenceStore } = useParentHooks()
  const { updateQueryParams } = useUpdateQueryParams<Partial<ArtifactListPageQueryParams>>()
  const queryParams = useQueryParams<ArtifactListPageQueryParams>(useArtifactListQueryParamOptions())
  const { searchTerm, isDeployedArtifacts, repositoryKey, page, size, latestVersion, packageTypes, labels } =
    queryParams
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
  } = useGetAllHarnessArtifactsQuery({
    space_ref: spaceRef,
    queryParams: {
      page,
      size,
      search_term: searchTerm,
      sort_field: sortField,
      sort_order: sortOrder,
      reg_identifier: repositoryKey ? [repositoryKey] : undefined,
      latest_version: latestVersion,
      deployed_artifact: isDeployedArtifacts,
      package_type: packageTypes,
      label: labels
    } as GetAllHarnessArtifactsQueryQueryParams,
    stringifyQueryParamsOptions: {
      arrayFormat: 'repeat'
    }
  })

  const handleClearAllFilters = (): void => {
    updateQueryParams({
      page: 0,
      searchTerm: '',
      isDeployedArtifacts: false,
      latestVersion: false
    })
  }

  const hasFilter = !!searchTerm || isDeployedArtifacts || latestVersion
  const responseData = data?.content?.data

  return (
    <>
      <Page.Header
        title={
          <div className="ng-tooltip-native">
            <h2 data-tooltip-id="artifactsPageHeading">{getString('artifactList.pageHeading')}</h2>
            <HarnessDocTooltip tooltipId="artifactsPageHeading" useStandAlone={true} />
          </div>
        }
        breadcrumbs={<NGBreadcrumbs links={[]} />}
      />
      <Page.SubHeader className={css.subHeader}>
        <div className={css.subHeaderItems}>
          <ArtifactSearchInput
            searchTerm={searchTerm || ''}
            onChange={text => {
              updateQueryParams({ searchTerm: text || undefined, page: DEFAULT_PAGE_INDEX })
            }}
            placeholder={getString('search')}
          />
          <RepositorySelector
            value={repositoryKey}
            onChange={val => {
              updateQueryParams({ repositoryKey: val, page: DEFAULT_PAGE_INDEX })
            }}
          />
          <PackageTypeSelector
            value={packageTypes}
            onChange={val => {
              updateQueryParams({ packageTypes: val, page: DEFAULT_PAGE_INDEX })
            }}
          />
          <LabelsSelector
            value={labels}
            onChange={val => {
              updateQueryParams({ labels: val, page: DEFAULT_PAGE_INDEX })
            }}
          />
          <Expander />
          <ButtonTabs
            className={css.filterTabContainer}
            small
            bold
            selectedTabId={
              latestVersion ? ArtifactListVersionFilter.LATEST_VERSION : ArtifactListVersionFilter.ALL_VERSION
            }
            onChange={newTab => {
              updateQueryParams({
                latestVersion: newTab === ArtifactListVersionFilter.LATEST_VERSION,
                page: DEFAULT_PAGE_INDEX
              })
            }}>
            <ButtonTab
              id={ArtifactListVersionFilter.LATEST_VERSION}
              icon="layers"
              iconProps={{ size: 12 }}
              panel={<></>}
              title={getString('artifactList.table.latestVersions')}
            />
            <ButtonTab
              id={ArtifactListVersionFilter.ALL_VERSION}
              icon="document"
              iconProps={{ size: 12 }}
              panel={<></>}
              title={getString('artifactList.table.allVersions')}
            />
          </ButtonTabs>
        </div>
      </Page.SubHeader>
      <Page.Body
        className={classNames(css.pageBody)}
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
            sortBy={sort}
          />
        )}
      </Page.Body>
    </>
  )
}

export default ArtifactListPage
