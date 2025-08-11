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
  Text,
  ExpandingSearchInput
} from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'
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
import { docLink, GitspaceOwnerType, GitspaceStatus, SortByType } from 'cde-gitness/constants'
import { useGetCDEAPIParams } from 'cde-gitness/hooks/useGetCDEAPIParams'
import SortByDropdown from 'cde-gitness/components/SortByDropdown/SortByDropdown'
import { useFindGitspaceSettings } from 'services/cde'
import type { EnumGitspaceSort } from 'services/cde'
import { useLisitngApi } from '../../hooks/useLisitngApi'
import GraduationHat from '../../../images/graduation-hat.svg?url'
import zeroDayCss from 'cde-gitness/components/CDEHomePage/CDEHomePage.module.scss'
import css from './GitspacesListing.module.scss'

interface pageCDEBrowser {
  page?: string
  limit?: string
  gitspace_states?: string
  gitspace_owner?: GitspaceOwnerType
  sort?: EnumGitspaceSort
  order?: 'asc' | 'desc'
  query?: string
}

interface filterProps {
  gitspace_states: GitspaceStatus[]
  gitspace_owner: GitspaceOwnerType
  query: string
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
  const { accountIdentifier = '' } = useGetCDEAPIParams()
  const {
    data: gitspaceSettings,
    loading: settingsLoading,
    error: settingsError
  } = useFindGitspaceSettings({
    accountIdentifier: accountIdentifier || '',
    lazy: !accountIdentifier
  })
  const filterInit: filterProps = {
    gitspace_states: statesString?.split(',')?.map((state: string) => state.trim() as GitspaceStatus) ?? [],
    gitspace_owner: pageBrowser.gitspace_owner ?? GitspaceOwnerType.SELF,
    query: pageBrowser.query ?? ''
  }
  const [filter, setFilter] = useState(filterInit)

  const pageInit: pageConfigProps = {
    page: pageBrowser.page ? parseInt(pageBrowser.page) : 1,
    limit: pageBrowser.limit ? parseInt(pageBrowser.limit) : LIST_FETCHING_LIMIT
  }
  const [pageConfig, setPageConfig] = useState(pageInit)
  const sortInit: sortProps = {
    sort: (pageBrowser.sort as EnumGitspaceSort) ?? SortByType.LAST_ACTIVATED,
    order: 'desc'
  }
  const [sortConfig, setSortConfig] = useState(sortInit)

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
    const gitspaceExists = useMemo(
      () => !!parseInt(responseData?.headers?.get('x-total-no-filter') || '0'),
      [responseData]
    )

    return { totalItems: totalData, totalPages, gitspaceExists }
  }
  const { totalItems, totalPages, gitspaceExists } = useParsePaginationInfo(response)

  const handleFilterChange = (key: string, value: any) => {
    const payload = { ...filter, [key]: value }
    setPageConfig(prevState => ({
      page: 1,
      limit: prevState.limit
    }))
    setFilter(payload)
    const params = typeof value === 'string' ? { [key]: value, page: 1 } : { [key]: value?.toString(), page: 1 }
    updateQueryParams(params)
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
      {((data && data?.length !== 0) || gitspaceExists) && (
        <>
          <Page.Header
            className={standalone ? '' : css.pageHeaderStyles}
            title={
              <Layout.Vertical spacing="none">
                <div className="ng-tooltip-native">
                  <h2> {getString('cde.manageGitspaces')}</h2>
                  <HarnessDocTooltip tooltipId="GitSpaceListPageHeading" useStandAlone={true} />
                </div>
                {!standalone ? (
                  <Text
                    font="small"
                    icon={<img src={GraduationHat} height={16} width={16} className={css.svgStyle} />}
                    color={Color.PRIMARY_7}
                    onClick={e => {
                      e.preventDefault()
                      e.stopPropagation()
                      window.open(docLink, '_blank')
                    }}
                    className={css.linkButton}>
                    {getString('cde.homePage.learnMoreAboutGitspaces')}
                  </Text>
                ) : (
                  <></>
                )}
              </Layout.Vertical>
            }
            breadcrumbs={
              <Breadcrumbs
                className={standalone ? '' : css.breadcrumbs}
                links={[{ url: routes.toCDEGitspaces({ space }), label: getString('cde.gitspaces') }]}
              />
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
                <ExpandingSearchInput
                  autoFocus={false}
                  alwaysExpanded
                  placeholder={getString('search')}
                  onChange={text => {
                    handleFilterChange('query', text)
                  }}
                  defaultValue={filter?.query ?? ''}
                />
              </Layout.Horizontal>
            </Page.SubHeader>
          )}
        </>
      )}
      <Container className={data?.length === 0 && !gitspaceExists ? zeroDayCss.background : css.main}>
        <Layout.Vertical spacing={'large'}>
          {data && data?.length === 0 && !gitspaceExists ? (
            <CDEHomePage />
          ) : (
            <Page.Body
              loading={loading || settingsLoading}
              error={
                error || settingsError ? (
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
                when: () => data?.length === 0 && !gitspaceExists,
                image: noSpace,
                message: getString('cde.noGitspaces')
              }}>
              {(data?.length || gitspaceExists) && (
                <>
                  <Text className={css.totalItems}>
                    {getString('cde.total')}: {totalItems}
                  </Text>
                  <ListGitspaces
                    data={(data as Unknown) || []}
                    hasFilter={gitspaceExists}
                    gitspaceSettings={gitspaceSettings}
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
