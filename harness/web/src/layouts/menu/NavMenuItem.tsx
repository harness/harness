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
  customIcon?: React.ReactNode
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
  customIcon,
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
    {customIcon && <span className={css.customIcon}>{customIcon}</span>}
    <Text icon={customIcon ? undefined : icon} rightIcon={rightIcon} className={css.text} {...textProps}>
      {label}
    </Text>
  </Link>
)
