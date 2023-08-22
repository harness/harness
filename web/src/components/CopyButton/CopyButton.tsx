import React, { useEffect, useState } from 'react'
import { Button, Utils, ButtonProps, ButtonVariation } from '@harnessio/uicore'
import { Color } from '@harnessio/design-system'

interface CopyButtonProps extends ButtonProps {
  content: string
}

export function CopyButton({ content, icon, iconProps, ...props }: CopyButtonProps) {
  const [copied, setCopied] = useState(false)

  useEffect(() => {
    let timeoutId: number
    if (copied) {
      timeoutId = window.setTimeout(() => setCopied(false), 2500)
    }
    return () => {
      clearTimeout(timeoutId)
    }
  }, [copied])

  return (
    <Button
      variation={ButtonVariation.ICON}
      icon={copied ? 'tick' : icon || 'copy-alt'}
      iconProps={{ color: copied ? Color.GREEN_500 : undefined, ...iconProps }}
      onClick={() => {
        setCopied(true)
        Utils.copy(content)
      }}
      {...props}
    />
  )
}
