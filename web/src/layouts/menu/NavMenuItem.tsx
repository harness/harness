import React from 'react'
import cx from 'classnames'
import { TextProps, Text } from '@harnessio/uicore'
import type { IconName } from '@harnessio/icons'
import { NavLink as Link, NavLinkProps } from 'react-router-dom'
import css from './NavMenuItem.module.scss'

interface NavMenuItemProps extends NavLinkProps {
  label: string
  icon?: IconName
  className?: string
  textProps?: TextProps
  rightIcon?: IconName
  isSubLink?: boolean
  isSelected?: boolean
  isDeselected?: boolean
  isHighlighted?: boolean
}

export const NavMenuItem: React.FC<NavMenuItemProps> = ({
  label,
  icon,
  rightIcon,
  className,
  isSubLink,
  textProps,
  isSelected,
  isDeselected,
  isHighlighted,
  children,
  ...others
}) => (
  <Link
    className={cx(css.link, className, {
      [css.subLink]: isSubLink,
      [css.selected]: isSelected,
      [css.highlighted]: isHighlighted
    })}
    activeClassName={isDeselected ? '' : css.selected}
    {...others}>
    {children}
    <Text icon={icon} rightIcon={rightIcon} className={css.text} {...textProps}>
      {label}
    </Text>
  </Link>
)
