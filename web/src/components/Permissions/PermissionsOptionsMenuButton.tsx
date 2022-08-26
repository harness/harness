import React, { AnchorHTMLAttributes, ReactElement } from 'react'
import type { IMenuItemProps } from '@blueprintjs/core'
import { OptionsMenuButton, OptionsMenuButtonProps } from 'components/OptionsMenuButton/OptionsMenuButton'
import { useAppContext } from 'AppContext'
import type { Unknown } from 'utils/Utils'

type Item = ((IMenuItemProps | PermissionsMenuItemProps) & AnchorHTMLAttributes<HTMLAnchorElement>) | '-'

interface PermissionsMenuItemProps extends IMenuItemProps {
  permission?: Unknown
}

export interface PermissionOptionsMenuButtonProps extends OptionsMenuButtonProps {
  items: Item[]
}

export const PermissionsOptionsMenuButton = (props: PermissionOptionsMenuButtonProps): ReactElement => {
  const {
    components: { RbacOptionsMenuButton }
  } = useAppContext()

  return RbacOptionsMenuButton ? <RbacOptionsMenuButton {...props} /> : <OptionsMenuButton {...props} />
}
