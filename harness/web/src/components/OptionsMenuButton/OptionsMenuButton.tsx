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

import React, { ReactElement } from 'react'
import cx from 'classnames'
import { omit } from 'lodash-es'
import { Classes, IconName, IMenuItemProps, Menu } from '@blueprintjs/core'
import { Button, ButtonProps } from '@harnessio/uicore'
import { Icon } from '@harnessio/icons'
import type { PopoverProps } from '@harnessio/uicore/dist/components/Popover/Popover'
import css from './OptionsMenuButton.module.scss'

export const MenuDivider = '-' as const

type OptionsMenuItem = React.ComponentProps<typeof Menu.Item> & {
  isDanger?: boolean
  text?: string
  hasIcon?: boolean
  iconName?: string
  iconSize?: number
}

export interface OptionsMenuButtonProps extends ButtonProps {
  items: Array<OptionsMenuItem | '-'>
  isDark?: boolean
  icon?: ButtonProps['icon']
  width?: string
}

export const OptionsMenuButton = ({
  items,
  icon = 'code-more',
  isDark = false,
  width = 'unset',
  ...props
}: OptionsMenuButtonProps): ReactElement => {
  return (
    <Button
      minimal
      icon={icon}
      tooltipProps={{ isDark, interactionKind: 'click', hasBackdrop: true } as PopoverProps}
      tooltip={
        <Menu style={{ minWidth: width }}>
          {items.map(
            (item, index) =>
              ((item as string) === MenuDivider && <Menu.Divider key={index} />) || (
                <Menu.Item
                  icon={
                    (item as OptionsMenuItem).hasIcon ? (
                      <Icon
                        size={(item as OptionsMenuItem).iconSize || 12}
                        className={css.icon}
                        name={(item as OptionsMenuItem).iconName as IconName}
                      />
                    ) : null
                  }
                  key={(item as React.ComponentProps<typeof Menu.Item>)?.text as string}
                  className={cx(
                    Classes.POPOVER_DISMISS,
                    {
                      [css.danger]: (item as OptionsMenuItem).isDanger,
                      [css.isDark]: isDark
                    },
                    (item as OptionsMenuItem).className
                  )}
                  {...omit(
                    item as IMenuItemProps & React.AnchorHTMLAttributes<HTMLAnchorElement>,
                    'isDanger',
                    'hasIcon',
                    'iconName',
                    'iconSize'
                  )}
                />
              )
          )}
        </Menu>
      }
      {...props}
    />
  )
}
