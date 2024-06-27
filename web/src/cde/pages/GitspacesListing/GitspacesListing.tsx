/*
 * Copyright 2023 Harness, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import React, { useEffect } from 'react'
import { Breadcrumbs, Text, Button, ButtonVariation, Layout, Page, Container } from '@harnessio/uicore'
import { FontVariation } from '@harnessio/design-system'
import { useHistory } from 'react-router-dom'
import { ListGitspaces } from 'cde/components/ListGitspaces/ListGitspaces'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { OpenapiGetGitspaceResponse, useListGitspaces } from 'services/cde'
import { LIST_FETCHING_LIMIT, PageBrowserProps, getErrorMessage } from 'utils/Utils'
import { useAppContext } from 'AppContext'
import { useStrings } from 'framework/strings'
import { usePageIndex } from 'hooks/usePageIndex'
import { useQueryParams } from 'hooks/useQueryParams'
import { ResourceListingPagination } from 'components/ResourceListingPagination/ResourceListingPagination'
import Gitspace from '../../icons/Gitspace.svg?url'
import noSpace from '../../images/no-gitspace.svg?url'
import css from './GitspacesListing.module.scss'

const GitspacesListing = () => {
  const space = useGetSpaceParam()
  const history = useHistory()
  const { getString } = useStrings()
  const { routes } = useAppContext()

  const pageBrowser = useQueryParams<PageBrowserProps>()
  const pageInit = pageBrowser.page ? parseInt(pageBrowser.page) : 1
  const [page, setPage] = usePageIndex(pageInit)

  const {
    data = '',
    loading = false,
    error = undefined,
    refetch,
    response
  } = useListGitspaces({
    queryParams: { page, limit: LIST_FETCHING_LIMIT },
    accountIdentifier: space?.split('/')[0],
    orgIdentifier: space?.split('/')[1],
    projectIdentifier: space?.split('/')[2]
  })

  useEffect(() => {
    if (!data && !loading) {
      history.push(routes.toCDEGitspacesCreate({ space }))
    }
  }, [data, loading])

  return (
    <>
      <Page.Header
        title={getString('cde.manageGitspaces')}
        breadcrumbs={
          <Layout.Horizontal spacing="small" flex={{ alignItems: 'center' }}>
            <img src={Gitspace} height={20} width={20} style={{ marginRight: '5px' }} />
            <Breadcrumbs
              links={[
                { url: routes.toCDEGitspaces({ space }), label: getString('cde.cloudDeveloperExperience') },
                { url: routes.toCDEGitspaces({ space }), label: getString('cde.gitspaces') }
              ]}
            />
          </Layout.Horizontal>
        }
      />
      <Page.SubHeader>
        <Button
          onClick={() => history.push(routes.toCDEGitspacesCreate({ space }))}
          variation={ButtonVariation.PRIMARY}>
          {getString('cde.newGitspace')}
        </Button>
      </Page.SubHeader>
      <Container className={css.main}>
        <Layout.Vertical spacing={'large'}>
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
              when: () => data?.length === 0,
              image: noSpace,
              message: getString('cde.noGitspaces')
            }}>
            <ListGitspaces data={data as OpenapiGetGitspaceResponse[]} refreshList={refetch} />
            <ResourceListingPagination response={response} page={page} setPage={setPage} />
          </Page.Body>
        </Layout.Vertical>
      </Container>
    </>
  )
}

export default GitspacesListing
