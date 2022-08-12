import React from 'react'
import { Button, ButtonProps } from '@harness/uicore'
import { useAppContext } from 'AppContext'

interface PermissionButtonProps extends ButtonProps {
  permission?: any
}

export const PermissionsButton: React.FC<PermissionButtonProps> = (props: PermissionButtonProps) => {
  const {
    components: { RbacButton }
  } = useAppContext()
  const { permission, ...buttonProps } = props

  return RbacButton ? <RbacButton permission={permission} {...props} /> : <Button {...buttonProps} />
}
