import React from 'react'
import { Avatar, Container, FlexExpander, Layout } from '@harness/uicore'
import { Render } from 'react-jsx-match'
import { routes } from 'RouteDefinitions'
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
  const { currentUser } = useAppContext()
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
            <Container className={css.settings}>
              <NavMenuItem icon="user-groups" label={getString('userManagement.text')} to={routes.toCODEUsers()} />
            </Container>
          </Render>

          <Render when={currentUser?.uid}>
            <Container className={css.profile}>
              <NavMenuItem
                label={currentUser?.display_name || currentUser?.email}
                to={routes.toCODEUserProfile()}
                textProps={{ tag: 'span' }}>
                <Avatar name={currentUser?.display_name || currentUser?.email} size="small" hoverCard={false} />
              </NavMenuItem>
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
