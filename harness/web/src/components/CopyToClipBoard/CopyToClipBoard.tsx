/*
 * Copyright 2021 Harness Inc. All rights reserved.
 * Use of this source code is governed by the PolyForm Shield 1.0.0 license
 * that can be found in the licenses directory at the root of this repository, also available at
 * https://polyformproject.org/wp-content/uploads/2020/06/PolyForm-Shield-1.0.0.txt.
 */

import React, { useState } from 'react'
import { Position, Popover } from '@blueprintjs/core'
import { Icon } from '@harnessio/icons'
import { Container, useToaster } from '@harnessio/uicore'

import css from './CopyToClipBoard.module.scss'

interface CopyToClipboardProps {
  content?: string
  text?: string // Added for backward compatibility
  showFeedback?: boolean
  iconSize?: number
  hidePopover?: boolean
  className?: string
}

const CopyToClipboard: React.FC<CopyToClipboardProps> = props => {
  const { showSuccess, showError } = useToaster()
  const { hidePopover = false, className } = props
  const [isOpen, setIsOpen] = useState(false)
  // Use either content or text prop
  const textToClipboard = props.content || props.text || ''

  const getPopoverContent = (): JSX.Element => {
    return (
      <Container className={css.popoverContent}>
        <span className={css.tooltipLabel}>Copied!</span>
      </Container>
    )
  }

  const handleCopy = async (event: React.MouseEvent<HTMLHeadingElement, globalThis.MouseEvent>) => {
    event.preventDefault()
    event.stopPropagation()

    try {
      await navigator.clipboard.writeText(textToClipboard)

      // Show the tooltip
      setIsOpen(true)

      // Hide the tooltip after 1.5 seconds
      setTimeout(() => {
        setIsOpen(false)
      }, 1500)

      // Show success notification if needed
      if (props.showFeedback) {
        showSuccess('Copied to clipboard')
      }
    } catch (error) {
      showError('Failed to copy:' + error)
    }
  }

  return (
    <>
      <Popover minimal position={Position.TOP_RIGHT} isOpen={!hidePopover && isOpen} content={getPopoverContent()}>
        <Container className={className}>
          <Icon name="copy-alt" size={props.iconSize ?? 20} onClick={handleCopy} className={css.copyIcon} />
        </Container>
      </Popover>
    </>
  )
}

export default CopyToClipboard
