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

import React, { useMemo, useState } from 'react'
import {
  Button,
  Page,
  ButtonVariation,
  Breadcrumbs,
  HarnessDocTooltip,
  Container,
  Layout,
  Text
} from '@harnessio/uicore'
import { FontVariation } from '@harnessio/design-system'
import { useHistory } from 'react-router-dom'
import { useAppContext } from 'AppContext'
import { useStrings } from 'framework/strings'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { getErrorMessage, LIST_FETCHING_LIMIT } from 'utils/Utils'
import noSpace from 'cde-gitness/assests/no-gitspace.svg?url'
import { useUpdateQueryParams } from 'hooks/useUpdateQueryParams'
import { useQueryParams } from 'hooks/useQueryParams'
import { ListGitspaces } from 'cde-gitness/components/GitspaceListing/ListGitspaces'
import CDEHomePage from 'cde-gitness/components/CDEHomePage/CDEHomePage'
import UsageMetrics from 'cde-gitness/components/UsageMetrics/UsageMetrics'

import StatusDropdown from 'cde-gitness/components/StatusDropdown/StatusDropdown'
import GitspaceOwnerDropdown from 'cde-gitness/components/GitspaceOwnerDropdown/GitspaceOwnerDropdown'
import { GitspaceOwnerType, GitspaceStatus } from 'cde-gitness/constants'

import SortByDropdown from 'cde-gitness/components/SortByDropdown/SortByDropdown'
import type { EnumGitspaceSort } from 'services/cde'
import { useLisitngApi } from '../../hooks/useLisitngApi'
import zeroDayCss from 'cde-gitness/components/CDEHomePage/CDEHomePage.module.scss'
import css from './GitspacesListing.module.scss'

interface pageCDEBrowser {
  page?: string
  limit?: string
  gitspace_states?: string
  gitspace_owner?: GitspaceOwnerType
  sort?: EnumGitspaceSort
  order?: 'asc' | 'desc'
}

interface filterProps {
  gitspace_states: GitspaceStatus[]
  gitspace_owner: GitspaceOwnerType
}

interface sortProps {
  sort: EnumGitspaceSort
  order: 'asc' | 'desc'
}

interface pageConfigProps {
  page: number
  limit: number
}

const GitspaceListing = () => {
  const space = useGetSpaceParam()
  const { updateQueryParams } = useUpdateQueryParams()

  const history = useHistory()
  const { getString } = useStrings()
  const { routes, standalone } = useAppContext()
  const pageBrowser = useQueryParams<pageCDEBrowser>()
  const statesString: any = pageBrowser.gitspace_states
  const filterInit: filterProps = {
    gitspace_states: statesString?.split(',')?.map((state: string) => state.trim() as GitspaceStatus) ?? [],
    gitspace_owner: pageBrowser.gitspace_owner ?? GitspaceOwnerType.SELF
  }
  const pageInit: pageConfigProps = {
    page: pageBrowser.page ? parseInt(pageBrowser.page) : 1,
    limit: pageBrowser.limit ? parseInt(pageBrowser.limit) : LIST_FETCHING_LIMIT
  }
  const [pageConfig, setPageConfig] = useState(pageInit)
  const [filter, setFilter] = useState(filterInit)

  const sortInit: sortProps = { sort: (pageBrowser.sort as EnumGitspaceSort) ?? '', order: 'desc' }
  const [sortConfig, setSortConfig] = useState(sortInit)
  const [hasFilter, setHasFilter] = useState(!!(pageBrowser.gitspace_states || pageBrowser.gitspace_owner))

  const {
    data = '',
    loading = false,
    error = undefined,
    refetch,
    response
  } = useLisitngApi({ page: pageConfig.page, limit: pageConfig.limit, filter, sortConfig })

  function useParsePaginationInfo(responseData: Nullable<Response>) {
    const totalData = useMemo(() => parseInt(responseData?.headers?.get('x-total') || '0'), [responseData])
    const totalPages = useMemo(() => parseInt(responseData?.headers?.get('x-total-pages') || '0'), [responseData])

    return { totalItems: totalData, totalPages }
  }
  const { totalItems, totalPages } = useParsePaginationInfo(response)

  const handleFilterChange = (key: string, value: any) => {
    const payload: any = { ...filter }
    payload[key] = value
    setFilter(payload)
    if (typeof value === 'string') {
      updateQueryParams({ [key]: value })
    } else if (Array.isArray(value)) {
      updateQueryParams({ [key]: value?.toString() })
    }
    if (payload.gitspace_states?.length || payload.gitspace_owner) {
      setHasFilter(true)
    }
  }

  const handleSort = (key: string, value: string) => {
    updateQueryParams({ [key]: value })
    setSortConfig({
      ...sortConfig,
      [key]: value
    })
  }

  const handlePagination = (key: string, value: number) => {
    const payload = {
      ...pageConfig,
      [key]: value
    }
    if (key === 'limit') {
      payload['page'] = 1
    }
    updateQueryParams({ page: payload?.page?.toString(), limit: payload?.limit?.toString() })
    setPageConfig(payload)
  }

  return (
    <>
      {((data && data?.length !== 0) || hasFilter) && (
        <>
          <Page.Header
            title={
              <div className="ng-tooltip-native">
                <h2> {getString('cde.manageGitspaces')}</h2>
                <HarnessDocTooltip tooltipId="GitSpaceListPageHeading" useStandAlone={true} />
              </div>
            }
            breadcrumbs={
              <Breadcrumbs links={[{ url: routes.toCDEGitspaces({ space }), label: getString('cde.gitspaces') }]} />
            }
            toolbar={
              standalone ? (
                <Button
                  onClick={() => history.push(routes.toCDEGitspacesCreate({ space }))}
                  variation={ButtonVariation.PRIMARY}>
                  {getString('cde.newGitspace')}
                </Button>
              ) : (
                <UsageMetrics />
              )
            }
          />
          {!standalone && (
            <Page.SubHeader>
              <Layout.Horizontal spacing="small" flex={{ alignItems: 'center', justifyContent: 'flex-start' }}>
                <Button
                  onClick={() => history.push(routes.toCDEGitspacesCreate({ space }))}
                  variation={ButtonVariation.PRIMARY}>
                  {getString('cde.newGitspace')}
                </Button>
                <StatusDropdown
                  value={filter.gitspace_states}
                  onChange={(val: any) => handleFilterChange('gitspace_states', val)}
                />
                <GitspaceOwnerDropdown
                  value={filter?.gitspace_owner}
                  onChange={(val: any) => handleFilterChange('gitspace_owner', val)}
                />
              </Layout.Horizontal>

              <Layout.Horizontal spacing="small" flex={{ alignItems: 'center', justifyContent: 'flex-start' }}>
                <SortByDropdown value={sortConfig?.sort} onChange={(val: any) => handleSort('sort', val)} />
                <Button
                  icon={sortConfig.order === 'asc' ? 'sort-asc' : 'sort-desc'}
                  className={css.sortOrder}
                  minimal
                  withoutBoxShadow
                  onClick={() => handleSort('order', sortConfig.order === 'asc' ? 'desc' : 'asc')}
                  disabled={!sortConfig.sort}
                />
              </Layout.Horizontal>
            </Page.SubHeader>
          )}
        </>
      )}
      <Container className={data?.length === 0 && !hasFilter ? zeroDayCss.background : css.main}>
        <Layout.Vertical spacing={'large'}>
          {data && data?.length === 0 && !hasFilter ? (
            <CDEHomePage />
          ) : (
            <Page.Body
              loading={loading}
              error={
                error ? (
                  <Layout.Vertical spacing={'large'}>
                    <Text font={{ variation: FontVariation.FORM_MESSAGE_DANGER }}>{getErrorMessage(error)}</Text>
                    <Button
                      onClick={() => refetch?.()}
                      variation={ButtonVariation.PRIMARY}
                      text={getString('cde.retry')}
                    />
                  </Layout.Vertical>
                ) : null
              }
              noData={{
                when: () => data?.length === 0 && !hasFilter,
                image: noSpace,
                message: getString('cde.noGitspaces')
              }}>
              {(data?.length || hasFilter) && (
                <>
                  <Text className={css.totalItems}>
                    {getString('cde.total')}: {totalItems}
                  </Text>
                  <ListGitspaces
                    data={(data as Unknown) || []}
                    hasFilter={hasFilter}
                    refreshList={refetch}
                    gotoPage={(pageNumber: number) => handlePagination('page', pageNumber + 1)}
                    onPageSizeChange={(newSize: number) => handlePagination('limit', newSize)}
                    pageConfig={{
                      page: pageConfig.page,
                      pageSize: pageConfig.limit,
                      totalItems,
                      totalPages
                    }}
                  />
                </>
              )}
            </Page.Body>
          )}
        </Layout.Vertical>
      </Container>
    </>
  )
}

export default GitspaceListing
