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

import React, { useState } from 'react'
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
import { getErrorMessage } from 'utils/Utils'
import noSpace from 'cde-gitness/assests/no-gitspace.svg?url'
import { ResourceListingPagination } from 'components/ResourceListingPagination/ResourceListingPagination'
import { useUpdateQueryParams } from 'hooks/useUpdateQueryParams'
import { useQueryParams } from 'hooks/useQueryParams'
import { usePageIndex } from 'hooks/usePageIndex'
import { ListGitspaces } from 'cde-gitness/components/GitspaceListing/ListGitspaces'
import CDEHomePage from 'cde-gitness/components/CDEHomePage/CDEHomePage'
import UsageMetrics from 'cde-gitness/components/UsageMetrics/UsageMetrics'

import StatusDropdown from 'cde-gitness/components/StatusDropdown/StatusDropdown'
import GitspaceOwnerDropdown from 'cde-gitness/components/GitspaceOwnerDropdown/GitspaceOwnerDropdown'
import { GitspaceOwnerType } from 'cde-gitness/constants'

import { useLisitngApi } from '../../hooks/useLisitngApi'
import zeroDayCss from 'cde-gitness/components/CDEHomePage/CDEHomePage.module.scss'
import css from './GitspacesListing.module.scss'

interface pageCDEBrowser {
  page?: string
  gitspace_states?: string
  gitspace_owner?: string
}

const GitspaceListing = () => {
  const space = useGetSpaceParam()
  const { replaceQueryParams } = useUpdateQueryParams()
  const history = useHistory()
  const { getString } = useStrings()
  const { routes, standalone } = useAppContext()
  const pageBrowser = useQueryParams<pageCDEBrowser>()
  const filterInit = {
    gitspace_states: pageBrowser.gitspace_states?.split(',') ?? [],
    gitspace_owner: pageBrowser.gitspace_owner ?? GitspaceOwnerType.SELF
  }
  const pageInit = pageBrowser.page ? parseInt(pageBrowser.page) : 1
  const [page, setPage] = usePageIndex(pageInit)
  const [filter, setFilter] = useState(filterInit)
  const [hasFilter, setHasFilter] = useState(!!(pageBrowser.gitspace_states || pageBrowser.gitspace_owner))

  const { data = '', loading = false, error = undefined, refetch, response } = useLisitngApi({ page, filter })

  const handleFilterChange = (key: string, value: any) => {
    const payload: any = { ...filter }
    payload[key] = value
    setFilter(payload)
    const queryParams: any = {}
    Object.keys(payload).forEach((entity: string) => {
      const val = payload[entity]
      if (val && typeof val === 'string') {
        queryParams[entity] = val
      } else if (Array.isArray(val) && val?.length) {
        queryParams[entity] = val?.toString()
      }
    })
    if (queryParams.gitspace_states?.length) {
      setHasFilter(true)
    }
    replaceQueryParams(queryParams)
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
                  <ListGitspaces data={(data as Unknown) || []} hasFilter={hasFilter} refreshList={refetch} />
                  <ResourceListingPagination response={response} page={page} setPage={setPage} />
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
