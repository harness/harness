import React, { useMemo } from 'react'
import { Container, Icon, Layout } from '@harness/uicore'
import { Render } from 'react-jsx-match'
import { Link, useRouteMatch } from 'react-router-dom'
import { useStrings } from 'framework/strings'
import { routes } from 'RouteDefinitions'
import { NavEntry } from './NavEntry'
import css from './layout.module.scss'

interface AppLayoutProps {
  type?: 'with-nav' | 'with-menu'
  menu?: React.ReactNode
}

const AppLayout: React.FC<AppLayoutProps> = ({ type, children, menu }) => {
  const { getString } = useStrings()
  const routeMatch = useRouteMatch()
  const isSpace = useMemo(
    () => routeMatch.path.startsWith('/:space') || routeMatch.path.endsWith('/:space'),
    [routeMatch]
  )

  if (!type) {
    return <>{children}</>
  }

  return (
    <Container className={css.main}>
      <Layout.Horizontal className={css.layout}>
        <nav className={css.nav}>
          <ol>
            {<NavCodeLogo />}

            <NavEntry href={routes.toCODESpaces()} icon="grid" text={getString('spaces')} isSelected={isSpace} />

            <li className={css.spacer}></li>

            <NavEntry href="//docs.harness.io" external icon="nav-help" text={getString('help')} />

            <NavEntry href={routes.toCODEGlobalSettings()} icon="code-settings" height="56px" />
          </ol>
        </nav>

        <Render when={type === 'with-menu'}>
          <Container className={css.menu}>{menu}</Container>
        </Render>

        <Container className={css.content}>{children}</Container>
      </Layout.Horizontal>
    </Container>
  )
}

const NavCodeLogo: React.FC = () => (
  <li>
    <Link to="/">
      <Icon name="code" size={34} />
    </Link>
  </li>
)

export const LayoutWithSideNav: React.FC = ({ children }) => <AppLayout type="with-nav">{children}</AppLayout>

export const LayoutWithSideMenu: React.FC<Required<Pick<AppLayoutProps, 'menu'>>> = ({ children, menu }) => (
  <AppLayout type="with-menu" menu={menu}>
    {children}
  </AppLayout>
)
