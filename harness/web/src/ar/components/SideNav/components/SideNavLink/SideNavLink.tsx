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
import cx from 'classnames'
import { NavLink, NavLinkProps } from 'react-router-dom'
import { PopoverInteractionKind, PopoverPosition, Classes } from '@blueprintjs/core'
import { Icon, IconName, IconProps } from '@harnessio/icons'
import { Text, Popover } from '@harnessio/uicore'
import { FontVariation, Color } from '@harnessio/design-system'
import css from './SideNavLink.module.scss'

export interface SideNavLinkProps extends NavLinkProps {
  icon?: IconName
  iconProps?: Partial<IconProps>
  label: string
  isRenderedInAccordion?: boolean
  hidden?: boolean
  __type?: string
  className?: string
  disableHighlightOnActive?: boolean
}

export const SideNavLink: React.FC<SideNavLinkProps> = (props): JSX.Element => {
  const { icon, iconProps, label, className, to, disableHighlightOnActive } = props
  const renderIcon = (): JSX.Element | undefined => {
    return icon ? <Icon name={icon} size={20} margin={{ right: 'small' }} {...iconProps} /> : undefined
  }

  const renderText = (): JSX.Element => {
    return (
      <Text font={{ variation: FontVariation.BODY }} color={Color.GREY_300}>
        {label}
      </Text>
    )
  }

  const renderLink = (): JSX.Element => {
    return (
      <NavLink
        data-name="nav-link"
        className={cx(css.link, className)}
        activeClassName={cx({ [css.selected]: !!to && !disableHighlightOnActive })}
        to={to}>
        {renderIcon()}
        {renderText()}
      </NavLink>
    )
  }

  return (
    <Popover
      interactionKind={PopoverInteractionKind.HOVER}
      position={PopoverPosition.RIGHT}
      boundary="viewport"
      popoverClassName={Classes.DARK}
      content={
        <Text color={Color.WHITE} padding="small">
          {label}
        </Text>
      }
      portalClassName={css.popover}>
      {renderLink()}
    </Popover>
  )
}

SideNavLink.defaultProps = {
  __type: 'SIDENAV_LINK'
}

export default SideNavLink
