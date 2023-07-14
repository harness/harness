import React from 'react'
import { Color, Icon, IconName, Layout, Text } from '@harness/uicore'
import cx from 'classnames'
import { Link, useRouteMatch, matchPath } from 'react-router-dom'
import css from './layout.module.scss'

interface NavEntryProps {
  href: string
  icon: IconName
  text?: string
  external?: boolean
  className?: string
  height?: string
  iconSize?: number
  isSelected?: boolean
  iconColor?: Color
}

export const NavEntry: React.FC<NavEntryProps> = ({
  href,
  icon,
  text,
  external,
  className,
  height,
  iconSize = 30,
  isSelected,
  iconColor = Color.WHITE
}) => {
  const routeMatch = useRouteMatch()
  const match = isSelected || matchPath(href, routeMatch)?.isExact
  const linkProps = external ? { to: href, target: '_blank' } : { to: href }

  return (
    <li
      className={cx({ [css.active]: !!match }, className)}
      style={{ '--nav-custom-item-height': height } as React.CSSProperties}>
      <Link {...linkProps}>
        <Layout.Vertical spacing="xsmall">
          <Icon name={icon} size={iconSize} color={iconColor} className={css.icon} />
          {text && <Text className={css.text}>{text}</Text>}
        </Layout.Vertical>
      </Link>
    </li>
  )
}
