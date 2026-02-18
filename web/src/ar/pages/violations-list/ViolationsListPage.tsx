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
import classNames from 'classnames'
import { flushSync } from 'react-dom'
import { Expander } from '@blueprintjs/core'
import { Color } from '@harnessio/design-system'
import { useGetArtifactScansV3Query } from '@harnessio/react-har-service-client'
import {
  Button,
  ButtonVariation,
  ExpandingSearchInput,
  ExpandingSearchInputHandle,
  Layout,
  Page
} from '@harnessio/uicore'

import { DEFAULT_PAGE_INDEX } from '@ar/constants'
import { useStrings } from '@ar/frameworks/strings'
import { EntityScope } from '@ar/common/types'
import { useAppStore, useParentHooks } from '@ar/hooks'
import useGetPageScope from '@ar/hooks/useGetPageScope'
import PackageTypeSelector from '@ar/components/PackageTypeSelector/PackageTypeSelector'

import ViolationsListTable from './ViolationsListTable'
import TableCard from './components/TableCard/TableCard'
import RepositorySelector from '../artifact-list/components/RepositorySelector/RepositorySelector'
import { useViolationsListQueryParamOptions, type ViolationsListPageQueryParams } from './utils'
import ScopeSelector from '../repository-list/components/RepositoryScopeSelector/RepositoryScopeSelector'

import css from './ViolationsListPage.module.scss'

export default function ViolationsListPage() {
  const { getString } = useStrings()
  const { scope } = useAppStore()
  const { accountId, orgIdentifier, projectIdentifier } = scope
  const { useQueryParams, useUpdateQueryParams } = useParentHooks()
  const { updateQueryParams, replaceQueryParams } = useUpdateQueryParams<Partial<ViolationsListPageQueryParams>>()
  const queryParams = useQueryParams<ViolationsListPageQueryParams>(useViolationsListQueryParamOptions())

  const pageScope = useGetPageScope()
  const searchRef = useRef({} as ExpandingSearchInputHandle)
  const { searchTerm, repositoryIds, packageTypes, page, size, sort, status, scope: scopeParam } = queryParams

  const {
    data,
    isFetching: loading,
    error,
    refetch
  } = useGetArtifactScansV3Query({
    queryParams: {
      account_identifier: accountId || '',
      org_identifier: orgIdentifier,
      project_identifier: projectIdentifier,
      registry_ids: repositoryIds,
      package_types: packageTypes,
      page,
      size,
      search_term: searchTerm,
      scan_status: status,
      scope: scopeParam
    },
    stringifyQueryParamsOptions: {
      arrayFormat: 'repeat'
    }
  })

  const handleClearAllFilters = (): void => {
    flushSync(searchRef.current.clear)
    replaceQueryParams({})
  }

  const hasFilter = !!searchTerm || repositoryIds?.length || packageTypes?.length
  const responseData = data?.content
  return (
    <>
      <Page.SubHeader className={css.subHeader}>
        <div className={css.subHeaderItems}>
          <Expander />
          {pageScope !== EntityScope.PROJECT && (
            <ScopeSelector
              scope={pageScope}
              value={scopeParam}
              onChange={val => {
                updateQueryParams({
                  scope: val,
                  page: DEFAULT_PAGE_INDEX
                })
              }}
            />
          )}
          <RepositorySelector
            value={repositoryIds}
            valueKey="uuid"
            onChange={val => {
              updateQueryParams({ repositoryIds: val, page: DEFAULT_PAGE_INDEX })
            }}
          />
          <PackageTypeSelector
            value={packageTypes}
            onChange={val => {
              updateQueryParams({ packageTypes: val, page: DEFAULT_PAGE_INDEX })
            }}
          />
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
      {responseData && (
        <Layout.Horizontal className={classNames(css.cardsContainer)} spacing="large">
          <TableCard
            title={getString('violationsList.cards.totalViolations')}
            value={responseData.meta.totalCount?.toLocaleString() || '0'}
            subText={getString('violationsList.cards.dependencies')}
            onClick={() => updateQueryParams({ status: undefined, page: DEFAULT_PAGE_INDEX })}
            active={!status}
          />
          <TableCard
            title={getString('violationsList.cards.blockedViolations')}
            titleIcon="warning-sign"
            iconProps={{ size: 12, color: Color.RED_600 }}
            value={responseData.meta.blockedCount?.toLocaleString() || '0'}
            subText={getString('violationsList.cards.dependencies')}
            onClick={() => updateQueryParams({ status: 'BLOCKED', page: DEFAULT_PAGE_INDEX })}
            active={status === 'BLOCKED'}
          />
          <TableCard
            title={getString('violationsList.cards.warningViolations')}
            titleIcon="warning-icon"
            iconProps={{ size: 12, color: Color.ORANGE_700 }}
            value={responseData.meta.warnCount?.toLocaleString() || '0'}
            subText={getString('violationsList.cards.dependencies')}
            onClick={() => updateQueryParams({ status: 'WARN', page: DEFAULT_PAGE_INDEX })}
            active={status === 'WARN'}
          />
        </Layout.Horizontal>
      )}
      <Page.Body
        className={classNames(css.pageBody)}
        loading={loading}
        error={error?.error?.message}
        retryOnError={() => refetch()}
        noData={{
          when: () => !responseData?.data?.length,
          icon: 'ccm-policy-shield-checked',
          messageTitle: hasFilter ? getString('noResultsFound') : getString('violationsList.noViolationsFound'),
          button: hasFilter ? (
            <Button text={getString('clearFilters')} variation={ButtonVariation.LINK} onClick={handleClearAllFilters} />
          ) : undefined
        }}>
        {responseData && (
          <Layout.Vertical spacing="large">
            <ViolationsListTable
              data={responseData}
              gotoPage={pageNumber => updateQueryParams({ page: pageNumber })}
              setSortBy={sortBy => updateQueryParams({ sort: sortBy, page: DEFAULT_PAGE_INDEX })}
              sortBy={sort}
            />
          </Layout.Vertical>
        )}
      </Page.Body>
    </>
  )
}
