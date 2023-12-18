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

import React from 'react'
import { Intent } from '@blueprintjs/core'
import { ButtonProps, ConfirmationDialog } from '@harnessio/uicore'
import { useModalHook } from 'hooks/useModalHook'

export interface UseConfirmationDialogProps {
  titleText: React.ReactNode
  contentText: React.ReactNode
  cancelButtonText?: React.ReactNode
  intent?: Intent
  buttonIntent?: ButtonProps['intent']
  confirmButtonText?: React.ReactNode
  onCloseDialog?: (isConfirmed: boolean) => void
  customButtons?: React.ReactNode
  showCloseButton?: boolean
  canOutsideClickClose?: boolean
  canEscapeKeyClose?: boolean
  children?: JSX.Element
  className?: string
  persistDialog?: boolean
}

export interface UseConfirmationDialogReturn {
  openDialog: () => void
  closeDialog: () => void
}

export const useConfirmationDialog = (props: UseConfirmationDialogProps): UseConfirmationDialogReturn => {
  const {
    titleText,
    contentText,
    cancelButtonText,
    intent = Intent.NONE,
    buttonIntent = Intent.PRIMARY,
    confirmButtonText,
    onCloseDialog,
    customButtons,
    showCloseButton,
    canOutsideClickClose,
    canEscapeKeyClose,
    children,
    className,
    persistDialog
  } = props

  const [showModal, hideModal] = useModalHook(() => {
    return (
      <ConfirmationDialog
        isOpen
        titleText={titleText}
        contentText={contentText}
        confirmButtonText={confirmButtonText}
        className={className}
        onClose={onClose}
        cancelButtonText={cancelButtonText}
        intent={intent}
        buttonIntent={buttonIntent}
        customButtons={customButtons}
        showCloseButton={showCloseButton}
        canOutsideClickClose={canOutsideClickClose}
        canEscapeKeyClose={canEscapeKeyClose}>
        {children}
      </ConfirmationDialog>
    )
  }, [props])

  const onClose = React.useCallback(
    (isConfirmed: boolean): void => {
      onCloseDialog?.(isConfirmed)
      hideModal()
      if (persistDialog) showModal()
      if (!isConfirmed) hideModal()
    },
    [hideModal, onCloseDialog, persistDialog]
  )

  return {
    openDialog: () => showModal(),
    closeDialog: () => hideModal()
  }
}
