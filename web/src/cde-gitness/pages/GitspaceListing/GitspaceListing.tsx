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

import React from 'react'
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
import { PageBrowserProps, getErrorMessage } from 'utils/Utils'
import noSpace from 'cde-gitness/assests/no-gitspace.svg?url'
import { ResourceListingPagination } from 'components/ResourceListingPagination/ResourceListingPagination'
import { useQueryParams } from 'hooks/useQueryParams'
import { usePageIndex } from 'hooks/usePageIndex'
import { ListGitspaces } from 'cde-gitness/components/GitspaceListing/ListGitspaces'
import CDEHomePage from 'cde-gitness/components/CDEHomePage/CDEHomePage'
import { useLisitngApi } from '../../hooks/useLisitngApi'
import css from './GitspacesListing.module.scss'
import zeroDayCss from 'cde-gitness/components/CDEHomePage/CDEHomePage.module.scss'

const GitspaceListing = () => {
  const space = useGetSpaceParam()
  const history = useHistory()
  const { getString } = useStrings()
  const { routes } = useAppContext()
  const pageBrowser = useQueryParams<PageBrowserProps>()
  const pageInit = pageBrowser.page ? parseInt(pageBrowser.page) : 1
  const [page, setPage] = usePageIndex(pageInit)

  const { data = '', loading = false, error = undefined, refetch, response } = useLisitngApi({ page })

  return (
    <>
      {data && data?.length !== 0 && (
        <Page.Header
          title={
            <div className="ng-tooltip-native">
              <h2> {getString('cde.gitspaces')}</h2>
              <HarnessDocTooltip tooltipId="GitSpaceListPageHeading" useStandAlone={true} />
            </div>
          }
          breadcrumbs={
            <Breadcrumbs links={[{ url: routes.toCDEGitspaces({ space }), label: getString('cde.gitspaces') }]} />
          }
          toolbar={
            <Button
              onClick={() => history.push(routes.toCDEGitspacesCreate({ space }))}
              variation={ButtonVariation.PRIMARY}>
              {getString('cde.newGitspace')}
            </Button>
          }
        />
      )}
      <Container className={data?.length === 0 ? zeroDayCss.background : css.main}>
        <Layout.Vertical spacing={'large'}>
          {data && data?.length === 0 ? (
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
                when: () => data?.length === 0,
                image: noSpace,
                message: getString('cde.noGitspaces')
              }}>
              {data?.length && (
                <>
                  <ListGitspaces data={(data as Unknown) || []} refreshList={refetch} />
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
