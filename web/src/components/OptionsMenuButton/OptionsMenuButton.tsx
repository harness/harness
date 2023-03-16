import React, { ReactElement } from 'react'
import cx from 'classnames'
import { omit } from 'lodash-es'
import { Classes, IconName, IMenuItemProps, Menu } from '@blueprintjs/core'
import { Button, ButtonProps, Icon } from '@harness/uicore'
import type { PopoverProps } from '@harness/uicore/dist/components/Popover/Popover'
import css from './OptionsMenuButton.module.scss'

export const MenuDivider = '-' as const

type OptionsMenuItem = React.ComponentProps<typeof Menu.Item> & {
  isDanger?: boolean
  text?: string
  hasIcon?: boolean
  iconName?: string
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
                      <Icon size={12} className={css.icon} name={(item as OptionsMenuItem).iconName as IconName} />
                    ) : null
                  }
                  key={(item as React.ComponentProps<typeof Menu.Item>)?.text as string}
                  className={cx(Classes.POPOVER_DISMISS, {
                    [css.danger]: (item as OptionsMenuItem).isDanger,
                    [css.isDark]: isDark
                  })}
                  {...omit(item as IMenuItemProps & React.AnchorHTMLAttributes<HTMLAnchorElement>, 'isDanger')}
                />
              )
          )}
        </Menu>
      }
      {...props}
    />
  )
}
