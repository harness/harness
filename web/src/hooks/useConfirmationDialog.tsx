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
    className
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
    },
    [hideModal, onCloseDialog]
  )

  return {
    openDialog: () => showModal(),
    closeDialog: () => hideModal()
  }
}
