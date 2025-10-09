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
import { Button, ButtonVariation, Container, Layout, Page, Text } from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'
import { useHistory } from 'react-router-dom'
import gitspace from 'cde-gitness/assests/gitspace.svg?url'
import homepageGraphics from 'cde-gitness/assests/homepageGraphics.svg?url'
import { useStrings } from 'framework/strings'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { useAppContext } from 'AppContext'
import css from './CDEHomePage.module.scss'

const CDEHomePage = () => {
  const space = useGetSpaceParam()
  const history = useHistory()
  const { getString } = useStrings()
  const { routes } = useAppContext()

  return (
    <Page.Body className={css.main}>
      <Container height="30%">
        <img src={gitspace} height={70} width={70} />
        <Layout.Vertical>
          <Layout.Horizontal spacing="large">
            <Text tag="h1" color={Color.BLACK} font={{ variation: FontVariation.H1 }}>
              {getString('cde.homePage.start')}
            </Text>
            <Text tag="h1" color={Color.PRIMARY_8} font={{ variation: FontVariation.H1 }}>{`<Coding>_`}</Text>
          </Layout.Horizontal>
          <Text color={Color.BLACK} font={{ variation: FontVariation.H1 }}>
            {getString('cde.homePage.noSetupRequired')}
          </Text>
        </Layout.Vertical>
        <Container margin={{ top: 'large' }}>
          <Text>{getString('cde.homePage.noteOne')}</Text>
          <Text>{getString('cde.homePage.noteTwo')}</Text>
        </Container>
      </Container>
      <Container margin={{ top: 'xxxlarge' }} width="70%">
        <Layout.Horizontal margin={{ bottom: 'large' }}>
          <Button
            onClick={() => history.push(routes.toCDEGitspacesCreate({ space }))}
            variation={ButtonVariation.PRIMARY}
            rightIcon="chevron-right">
            {getString('cde.homePage.getStartedNow')}
          </Button>
          <Button
            onClick={e => {
              e.preventDefault()
              window.open('https://developer.harness.io/docs/cloud-development-environments/', '_blank')
            }}
            variation={ButtonVariation.LINK}
            rightIcon="launch">
            {getString('cde.homePage.learnMoreAboutGitspaces')}
          </Button>
        </Layout.Horizontal>
        <img src={homepageGraphics} style={{ width: '60vw' }} />
      </Container>
    </Page.Body>
  )
}

export default CDEHomePage
