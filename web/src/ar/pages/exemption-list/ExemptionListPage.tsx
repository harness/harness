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
import {
  ListFirewallExceptionsV3QueryQueryParams,
  useListFirewallExceptionsV3Query
} from '@harnessio/react-har-service-client'
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
import TableCard from '@ar/pages/violations-list/components/TableCard/TableCard'
import PackageTypeSelector from '@ar/components/PackageTypeSelector/PackageTypeSelector'

import ExemptionListTable from './ExemptionListTable'
import RepositorySelector from '../artifact-list/components/RepositorySelector/RepositorySelector'
import { useExemptionListQueryParamOptions, type ExemptionListPageQueryParams } from './utils'
import ScopeSelector from '../repository-list/components/RepositoryScopeSelector/RepositoryScopeSelector'

import css from './ExemptionListPage.module.scss'

export default function ExemptionListPage() {
  const { getString } = useStrings()
  const { scope } = useAppStore()
  const { accountId, orgIdentifier, projectIdentifier } = scope
  const { useQueryParams, useUpdateQueryParams } = useParentHooks()
  const { updateQueryParams, replaceQueryParams } = useUpdateQueryParams<Partial<ExemptionListPageQueryParams>>()
  const queryParams = useQueryParams<ExemptionListPageQueryParams>(useExemptionListQueryParamOptions())

  const pageScope = useGetPageScope()
  const searchRef = useRef({} as ExpandingSearchInputHandle)
  const { repositoryIds, packageTypes, page, size, sort, status, scope: scopeParam, searchTerm } = queryParams

  const {
    data,
    isFetching: loading,
    error,
    refetch
  } = useListFirewallExceptionsV3Query({
    queryParams: {
      account_identifier: accountId || '',
      org_identifier: orgIdentifier,
      project_identifier: projectIdentifier,
      status: status,
      package_types: packageTypes.length ? packageTypes : undefined,
      registry_ids: repositoryIds.length ? repositoryIds : undefined,
      search_term: searchTerm,
      page,
      size,
      sort: sort.join(':')
    } as ListFirewallExceptionsV3QueryQueryParams,
    stringifyQueryParamsOptions: {
      arrayFormat: 'repeat'
    }
  })

  const handleClearAllFilters = (): void => {
    flushSync(searchRef.current.clear)
    replaceQueryParams({})
  }

  const hasFilter = repositoryIds?.length || packageTypes?.length || status || searchTerm
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
      <Layout.Horizontal className={classNames(css.cardsContainer)} spacing="large">
        <TableCard
          title={getString('exemptionList.cards.totalExemptions')}
          value={responseData?.meta?.totalCount?.toLocaleString() || '0'}
          onClick={() => updateQueryParams({ status: undefined, page: DEFAULT_PAGE_INDEX })}
          active={!status}
        />
        <TableCard
          title={getString('exemptionList.cards.approvedExemptions')}
          titleIcon="tick-circle"
          iconProps={{ size: 14, color: Color.GREEN_600 }}
          value={responseData?.meta?.approvedCount?.toLocaleString() || '0'}
          onClick={() => updateQueryParams({ status: 'APPROVED', page: DEFAULT_PAGE_INDEX })}
          active={status === 'APPROVED'}
        />
        <TableCard
          title={getString('exemptionList.cards.rejectedExemptions')}
          titleIcon="circle-cross"
          iconProps={{ size: 14, color: Color.RED_600 }}
          value={responseData?.meta?.rejectedCount?.toLocaleString() || '0'}
          onClick={() => updateQueryParams({ status: 'REJECTED', page: DEFAULT_PAGE_INDEX })}
          active={status === 'REJECTED'}
        />
        <TableCard
          title={getString('exemptionList.cards.pendingExemptions')}
          titleIcon="status-pending"
          iconProps={{ size: 14, color: Color.ORANGE_700 }}
          value={responseData?.meta?.pendingCount?.toLocaleString() || '0'}
          onClick={() => updateQueryParams({ status: 'PENDING', page: DEFAULT_PAGE_INDEX })}
          active={status === 'PENDING'}
        />
        <TableCard
          title={getString('exemptionList.cards.expiredExemptions')}
          titleIcon="expired"
          iconProps={{ size: 14 }}
          value={responseData?.meta?.expiredCount?.toLocaleString() || '0'}
          onClick={() => updateQueryParams({ status: 'EXPIRED', page: DEFAULT_PAGE_INDEX })}
          active={status === 'EXPIRED'}
        />
      </Layout.Horizontal>
      <Page.Body
        className={classNames(css.pageBody)}
        loading={loading}
        error={error?.error?.message}
        retryOnError={() => refetch()}
        noData={{
          when: () => !responseData?.items?.length,
          icon: 'ccm-policy-shield-checked',
          messageTitle: hasFilter ? getString('noResultsFound') : getString('exemptionList.noExemptionsFound'),
          button: hasFilter ? (
            <Button text={getString('clearFilters')} variation={ButtonVariation.LINK} onClick={handleClearAllFilters} />
          ) : undefined
        }}>
        {responseData && (
          <Layout.Vertical spacing="large">
            <ExemptionListTable
              data={responseData}
              gotoPage={pageNumber => updateQueryParams({ page: pageNumber })}
              onPageSizeChange={pageSize => updateQueryParams({ size: pageSize, page: DEFAULT_PAGE_INDEX })}
              setSortBy={sortBy => updateQueryParams({ sort: sortBy, page: DEFAULT_PAGE_INDEX })}
              sortBy={sort}
            />
          </Layout.Vertical>
        )}
      </Page.Body>
    </>
  )
}
