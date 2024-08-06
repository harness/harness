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
import { useHistory } from 'react-router-dom'
import { Avatar, Button, ButtonVariation, Container, FlexExpander, Layout } from '@harnessio/uicore'
import { Render } from 'react-jsx-match'
import { ProfileCircle } from 'iconoir-react'
import { useAppContext } from 'AppContext'
import { useStrings } from 'framework/strings'
import { useDocumentTitle } from 'hooks/useDocumentTitle'
import { NavMenuItem } from './menu/NavMenuItem'
import { GitnessLogo } from '../components/GitnessLogo/GitnessLogo'
import { DefaultMenu } from './menu/DefaultMenu'
import css from './layout.module.scss'

interface LayoutWithSideNavProps {
  title: string
  menu?: React.ReactNode
}

export const LayoutWithSideNav: React.FC<LayoutWithSideNavProps> = ({ title, children, menu = <DefaultMenu /> }) => {
  const { routes, currentUser, isCurrentSessionPublic } = useAppContext()
  const history = useHistory()
  const { getString } = useStrings()
  useDocumentTitle(title)

  return (
    <Container className={css.main}>
      <Layout.Horizontal className={css.layout}>
        <Container className={css.menu}>
          <Layout.Vertical spacing="small">
            <GitnessLogo />
            <Container>{menu}</Container>
          </Layout.Vertical>

          <FlexExpander />

          <Render when={currentUser?.admin}>
            <Container className={css.userManagement}>
              <NavMenuItem
                customIcon={<ProfileCircle />}
                label={getString('userManagement.text')}
                to={routes.toCODEUsers()}
              />
            </Container>
          </Render>

          <Render when={currentUser?.uid}>
            <Container className={css.navContainer}>
              <NavMenuItem
                label={currentUser?.display_name || currentUser?.email}
                to={routes.toCODEUserProfile()}
                textProps={{ tag: 'span' }}>
                <Avatar name={currentUser?.display_name || currentUser?.email} size="small" hoverCard={false} />
              </NavMenuItem>
            </Container>
          </Render>

          <Render when={isCurrentSessionPublic}>
            <Container className={css.navContainer}>
              <Button
                onClick={() => history.push(routes.toSignIn())}
                variation={ButtonVariation.PRIMARY}
                intent="primary"
                loading={false}
                disabled={false}
                width="100%">
                {getString('signIn')}
              </Button>
            </Container>
          </Render>
        </Container>

        <Container className={css.content}>{children}</Container>
      </Layout.Horizontal>
    </Container>
  )
}

export const LayoutWithoutSideNav: React.FC<{ title: string }> = ({ title, children }) => {
  useDocumentTitle(title)
  return <>{children}</>
}
