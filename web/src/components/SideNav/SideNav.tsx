import React from 'react'
import cx from 'classnames'
import { NavLink as Link, NavLinkProps } from 'react-router-dom'
import { Container, Text, Layout, IconName } from '@harness/uicore'
import { useAPIToken } from 'hooks/useAPIToken'
import { useStrings } from 'framework/strings'
import routes from 'RouteDefinitions'
import css from './SideNav.module.scss'

interface SidebarLinkProps extends NavLinkProps {
  label: string
  icon?: IconName
  className?: string
}

const SidebarLink: React.FC<SidebarLinkProps> = ({ label, icon, className, ...others }) => (
  <Link className={cx(css.link, className)} activeClassName={css.selected} {...others}>
    <Text icon={icon} className={css.text}>
      {label}
    </Text>
  </Link>
)

export const SideNav: React.FC = ({ children }) => {
  const { getString } = useStrings()
  const [, setToken] = useAPIToken()

  return (
    <Container height="inherit" className={css.root}>
      <Layout.Vertical spacing="small" padding={{ top: 'xxxlarge' }} className={css.sideNav}>
        <SidebarLink exact icon="pipeline" label={getString('pipelines')} to={routes.toPipelines()} />
        <SidebarLink exact icon="advanced" label={getString('account')} to={routes.toAccount()} />
        <SidebarLink onClick={() => setToken('')} icon="log-out" label={getString('logout')} to={routes.toLogin()} />
      </Layout.Vertical>
      {children}
    </Container>
  )
}
