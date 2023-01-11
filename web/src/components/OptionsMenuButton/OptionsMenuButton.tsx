import React, { ReactElement } from 'react'
import cx from 'classnames'
import { Classes, IMenuItemProps, Menu } from '@blueprintjs/core'
import { Button, ButtonProps } from '@harness/uicore'
import type { PopoverProps } from '@harness/uicore/dist/components/Popover/Popover'
import css from './OptionsMenuButton.module.scss'

export const MenuDivider = '-' as const

type OptionsMenuItem = React.ComponentProps<typeof Menu.Item> & {
  isDanger?: boolean
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
                  key={(item as React.ComponentProps<typeof Menu.Item>)?.text as string}
                  className={cx(Classes.POPOVER_DISMISS, { [css.danger]: (item as OptionsMenuItem).isDanger })}
                  {...(item as IMenuItemProps & React.AnchorHTMLAttributes<HTMLAnchorElement>)}
                />
              )
          )}
        </Menu>
      }
      {...props}
    />
  )
}
