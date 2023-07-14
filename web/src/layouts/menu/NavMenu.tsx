import React from 'react'
import cx from 'classnames'
import { IconName, TextProps, Text } from '@harness/uicore'
import { NavLink as Link, NavLinkProps } from 'react-router-dom'
import css from './NavMenu.module.scss'

interface NavMenuProps extends NavLinkProps {
  label: string
  icon?: IconName
  className?: string
  textProps?: TextProps
  rightIcon?: IconName
  isSubLink?: boolean
  isSelected?: boolean
  isDeselected?: boolean
}

export const NavMenu: React.FC<NavMenuProps> = ({
  label,
  icon,
  rightIcon,
  className,
  isSubLink,
  textProps,
  isSelected,
  isDeselected,
  ...others
}) => (
  <Link
    className={cx(css.link, className, { [css.subLink]: isSubLink, [css.selected]: isSelected })}
    activeClassName={isDeselected ? '' : css.selected}
    {...others}>
    <Text icon={icon} rightIcon={rightIcon} className={css.text} {...textProps}>
      {label}
    </Text>
  </Link>
)
