import React from 'react'
import { Button, ButtonProps } from '@harness/uicore'
import { useAppContext } from 'AppContext'
import type { Unknown } from 'utils/Utils'

interface PermissionButtonProps extends ButtonProps {
  permission?: Unknown
}

export const PermissionsButton: React.FC<PermissionButtonProps> = (props: PermissionButtonProps) => {
  const {
    components: { RbacButton }
  } = useAppContext()
  const { permission, ...buttonProps } = props

  return RbacButton ? <RbacButton permission={permission} {...props} /> : <Button {...buttonProps} />
}
