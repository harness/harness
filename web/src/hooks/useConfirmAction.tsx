/*
 * Copyright 2022 Harness Inc. All rights reserved.
 * Use of this source code is governed by the PolyForm Shield 1.0.0 license
 * that can be found in the licenses directory at the root of this repository, also available at
 * https://polyformproject.org/wp-content/uploads/2020/06/PolyForm-Shield-1.0.0.txt.
 */

import { Intent } from '@blueprintjs/core'
import { useConfirmationDialog } from '@harness/uicore'
import { useStrings } from 'framework/strings'

export interface UseConfirmActionDialogProps {
  message: React.ReactElement
  intent?: Intent
  title?: string
  confirmText?: string
  cancelText?: string
  action: () => void
}

export const useConfirmAction = (props: UseConfirmActionDialogProps) => {
  const { title, message, confirmText, cancelText, intent, action } = props
  const { getString } = useStrings()
  const { openDialog } = useConfirmationDialog({
    intent,
    titleText: title || getString('confirmation'),
    contentText: message,
    confirmButtonText: confirmText || getString('confirm'),
    cancelButtonText: cancelText || getString('cancel'),
    buttonIntent: intent || Intent.DANGER,
    onCloseDialog: async (isConfirmed: boolean) => {
      if (isConfirmed) {
        action()
      }
    }
  })

  return openDialog
}
