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
import { Breadcrumbs, Card, Container, Heading, Layout, Page, Text } from '@harnessio/uicore'
import { Color } from '@harnessio/design-system'
import { useStrings } from 'framework/strings'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { useAppContext } from 'AppContext'
import { GitnessCreateGitspace } from './GitnessCreateGitspace'
import { CDECreateGitspace } from './CDECreateGitspace'
import css from './GitspaceCreate.module.scss'

const GitspaceCreate = () => {
  const { getString } = useStrings()
  const space = useGetSpaceParam()
  const { standalone, routes } = useAppContext()

  return (
    <>
      <Page.Header
        title={getString('cde.createGitspace')}
        breadcrumbs={
          <Breadcrumbs
            links={[
              { url: routes.toCDEGitspaces({ space }), label: getString('cde.gitspaces') },
              { url: routes.toCDEGitspacesCreate({ space }), label: getString('cde.createGitspace') }
            ]}
          />
        }
      />
      <Page.Body className={css.main}>
        <Container className={css.titleContainer}>
          <Layout.Vertical spacing="small" margin={{ bottom: 'medium' }}>
            <Heading font={{ weight: 'bold' }} color={Color.BLACK} level={2}>
              {getString('cde.createGitspace')}
            </Heading>
            <Text font={{ size: 'medium' }}>{getString('cde.create.subtext')}</Text>
          </Layout.Vertical>
        </Container>
        <Card className={css.cardMain}>
          <Container className={css.subContainers}>
            {standalone ? <GitnessCreateGitspace /> : <CDECreateGitspace />}
          </Container>
        </Card>
      </Page.Body>
    </>
  )
}

export default GitspaceCreate
