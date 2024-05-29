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
import {
  Breadcrumbs,
  Text,
  Button,
  ButtonVariation,
  ExpandingSearchInput,
  Layout,
  Page,
  Container
} from '@harnessio/uicore'
import { FontVariation } from '@harnessio/design-system'
import { useHistory } from 'react-router-dom'
import { ListGitspaces } from 'cde/components/ListGitspaces/ListGitspaces'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { OpenapiGetGitspaceResponse, useListGitspaces } from 'services/cde'
import { getErrorMessage } from 'utils/Utils'
import { useAppContext } from 'AppContext'
import { useStrings } from 'framework/strings'
import Gitspace from '../../icons/Gitspace.svg?url'
import noSpace from '../../images/no-gitspace.svg?url'
import css from './GitspacesListing.module.scss'

const GitspacesListing = () => {
  const space = useGetSpaceParam()
  const history = useHistory()
  const { getString } = useStrings()
  const { routes } = useAppContext()
  const { data, loading, error, refetch } = useListGitspaces({
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
        title=""
        breadcrumbs={
          <Layout.Horizontal spacing="small" flex={{ alignItems: 'center' }}>
            <img src={Gitspace} height={20} width={20} style={{ marginRight: '5px' }} />
            <Breadcrumbs
              links={[
                { url: routes.toCDEGitspaces({ space }), label: getString('cde.cloudDeveloperExperience') },
                { url: routes.toCDEGitspaces({ space }), label: getString('cde.createGitspace') }
              ]}
            />
          </Layout.Horizontal>
        }
      />
      <Container className={css.main}>
        <Layout.Vertical spacing={'large'}>
          <Layout.Horizontal flex={{ justifyContent: 'space-between', alignItems: 'center' }}>
            <Text font={{ variation: FontVariation.H3 }}>{getString('cde.manageGitspaces')}</Text>
            <Button onClick={() => history.push(routes.toCDEGitspaces({ space }))} variation={ButtonVariation.PRIMARY}>
              {getString('cde.newGitspace')}
            </Button>
          </Layout.Horizontal>
          <Page.Body
            loading={loading}
            error={
              <Layout.Vertical spacing={'large'}>
                <Text font={{ variation: FontVariation.FORM_MESSAGE_DANGER }}>{getErrorMessage(error)}</Text>
                <Button onClick={() => refetch()} variation={ButtonVariation.PRIMARY} text={'Retry'} />
              </Layout.Vertical>
            }
            noData={{
              when: () => data?.length === 0,
              image: noSpace,
              message: getString('cde.noGitspaces'),
              button: (
                <Button
                  onClick={() => history.push(routes.toCDEGitspaces({ space }))}
                  variation={ButtonVariation.PRIMARY}
                  text={getString('cde.newGitspace')}
                />
              )
            }}>
            <ExpandingSearchInput width={'50%'} alwaysExpanded />
            <ListGitspaces data={data as OpenapiGetGitspaceResponse[]} />
          </Page.Body>
        </Layout.Vertical>
      </Container>
    </>
  )
}

export default GitspacesListing
