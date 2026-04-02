/*
 * Copyright 2024 Harness, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import React from 'react'
import { Intent } from '@blueprintjs/core'
import { useToaster } from '@harnessio/uicore'
import { useUpdateStatusFirewallExceptionV3Mutation } from '@harnessio/react-har-service-client'

import { useAppStore, useConfirmationDialog } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'
import StatusChangeModal, {
  type StatusChangeFormValues
} from '@ar/pages/exemption-list/components/StatusChangeModal/StatusChangeModal'

interface useRejectExemptionProps {
  exemptionId: string
  onSuccess: () => void
}
export default function useRejectExemption(props: useRejectExemptionProps) {
  const { exemptionId, onSuccess } = props
  const { getString } = useStrings()
  const { scope } = useAppStore()
  const { showSuccess, showError, clear } = useToaster()

  const { mutateAsync: updateStatus } = useUpdateStatusFirewallExceptionV3Mutation()

  const handleReject = async (values: StatusChangeFormValues): Promise<void> => {
    return updateStatus({
      id: exemptionId,
      queryParams: {
        account_identifier: scope.accountId || ''
      },
      body: {
        status: 'REJECTED',
        notes: values.value
      }
    })
      .then(() => {
        clear()
        showSuccess(getString('exemptionDetails.toasters.exemptionRejected'))
        onSuccess()
        closeDialog()
      })
      .catch((e: any) => {
        showError(e.message || e.error?.message || getString('exemptionDetails.toasters.failedToUpdateStatus'))
      })
  }

  const handleCloseDialog = () => {
    closeDialog()
  }

  const { openDialog, closeDialog } = useConfirmationDialog({
    titleText: getString('exemptionDetails.rejectModal.title'),
    contentText: (
      <StatusChangeModal
        content={getString('exemptionDetails.rejectModal.content')}
        inputLabel={getString('exemptionDetails.rejectModal.inputLabel')}
        placeholder={getString('exemptionDetails.rejectModal.placeholder')}
        submitButtonText={getString('exemptionDetails.rejectModal.submitButton')}
        submitButtonIntent={Intent.DANGER}
        cancelButtonText={getString('exemptionDetails.rejectModal.cancelButton')}
        onSubmit={handleReject}
        onClose={handleCloseDialog}
      />
    ),
    customButtons: <></>,
    intent: Intent.DANGER,
    onCloseDialog: handleCloseDialog
  })

  return { triggerReject: openDialog }
}
