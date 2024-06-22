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

import React from 'react'
import { useParams } from 'react-router-dom'
import { Breadcrumbs, Container, Heading, Layout, Page, Text } from '@harnessio/uicore'
import { Color } from '@harnessio/design-system'
import { CreateGitspace } from 'cde/components/CreateGitspace/CreateGitspace'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { useAppContext } from 'AppContext'
import { useStrings } from 'framework/strings'
import Gitspace from '../../icons/Gitspace.svg?url'
import css from './Gitspaces.module.scss'

const CreateGitspaceTitle = () => {
  const { getString } = useStrings()
  return (
    <Container className={css.gitspaceTitle}>
      <img src={Gitspace} width={42} height={42}></img>
      <Layout.Vertical spacing="small">
        <Heading color={Color.BLACK} level={1}>
          {getString('cde.introText1')}
        </Heading>
        <Text>{getString('cde.introText2')}</Text>
        <Text>{getString('cde.introText3')}</Text>
      </Layout.Vertical>
    </Container>
  )
}

const Gitspaces = () => {
  const space = useGetSpaceParam()
  const { getString } = useStrings()
  const { routes } = useAppContext()
  const { gitspaceId = '' } = useParams<{ gitspaceId?: string }>()
  const createEditLabel = gitspaceId ? getString('cde.editGitspace') : getString('cde.createGitspace')
  return (
    <>
      <Page.Header
        title={createEditLabel}
        breadcrumbs={
          <Layout.Horizontal spacing="small" flex={{ alignItems: 'center' }}>
            <img src={Gitspace} height={20} width={20} style={{ marginRight: '5px' }} />
            <Breadcrumbs
              links={[
                { url: routes.toCDEGitspaces({ space }), label: getString('cde.cloudDeveloperExperience') },
                {
                  url: gitspaceId ? routes.toCDEGitspacesEdit({ space, gitspaceId }) : routes.toCDEGitspaces({ space }),
                  label: createEditLabel
                }
              ]}
            />
          </Layout.Horizontal>
        }
      />
      <Page.Body>
        <Layout.Vertical className={css.main} spacing="medium">
          <CreateGitspaceTitle />
          <CreateGitspace />
        </Layout.Vertical>
      </Page.Body>
    </>
  )
}

export default Gitspaces
