import React, { ReactElement } from 'react'
import { Classes, IMenuItemProps, Menu } from '@blueprintjs/core'
import { Button, ButtonProps } from '@harness/uicore'
import type { PopoverProps } from '@harness/uicore/dist/components/Popover/Popover'

export const MenuDivider = '-' as const

export interface OptionsMenuButtonProps extends ButtonProps {
  items: Array<React.ComponentProps<typeof Menu.Item> | '-'>
}

export const OptionsMenuButton = ({ items, ...props }: OptionsMenuButtonProps): ReactElement => {
  return (
    <Button
      minimal
      icon="Options"
      tooltipProps={{ isDark: true, interactionKind: 'click', hasBackdrop: true } as PopoverProps}
      tooltip={
        <Menu style={{ minWidth: 'unset' }}>
          {items.map(
            (item, index) =>
              ((item as string) === MenuDivider && <Menu.Divider key={index} />) || (
                <Menu.Item
                  key={(item as React.ComponentProps<typeof Menu.Item>)?.text as string}
                  className={Classes.POPOVER_DISMISS}
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
