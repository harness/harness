import React, { useState } from 'react'
import { Button } from '@harnessio/uicore'
import copy from 'clipboard-copy'
import { useStrings } from 'framework/strings'
import codeCopy from 'cde-gitness/assests/codeCopy.svg?url'

const CopyButton = ({ value, className }: { value?: string; className?: string }) => {
  const { getString } = useStrings()
  const [openTooltip, setOpenTooltip] = useState(false)
  const showCopySuccess = (): void => {
    setOpenTooltip(true)
    setTimeout(() => {
      setOpenTooltip(false)
    }, 1000)
  }
  return (
    <Button
      minimal
      className={className}
      icon={<img src={codeCopy} height={16} width={16} />}
      onClick={event => {
        event.preventDefault()
        event.stopPropagation()
        copy(value ?? '')
        showCopySuccess()
      }}
      withoutCurrentColor
      tooltip={getString('cde.copied')}
      tooltipProps={{ isOpen: openTooltip, isDark: true }}
    />
  )
}

export default CopyButton
