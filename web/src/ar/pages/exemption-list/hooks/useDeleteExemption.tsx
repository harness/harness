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
import { useDeleteFirewallExceptionV3Mutation } from '@harnessio/react-har-service-client'

import { useStrings } from '@ar/frameworks/strings'
import { useAppStore, useParentHooks } from '@ar/hooks'
import DeleteModalContent from '@ar/components/Form/DeleteModalContent'

interface useDeleteExemptionModalProps {
  exemptionId: string
  packageName: string
  onSuccess: () => void
}

export default function useDeleteExemptionModal(props: useDeleteExemptionModalProps) {
  const { exemptionId, packageName, onSuccess } = props
  const { getString } = useStrings()
  const { showSuccess, showError, clear } = useToaster()
  const { useConfirmationDialog } = useParentHooks()
  const { scope } = useAppStore()

  const { mutateAsync: deleteExemption } = useDeleteFirewallExceptionV3Mutation()

  const handleDeleteExemption = async (): Promise<void> => {
    return deleteExemption({
      id: exemptionId,
      queryParams: {
        account_identifier: scope.accountId || ''
      }
    })
      .then(() => {
        clear()
        showSuccess(getString('exemptionList.table.toasters.exemptionDeleted'))
        onSuccess()
      })
      .catch(error => {
        showError(
          error?.message || error?.error?.message || getString('exemptionList.table.toasters.exemptionDeleteError')
        )
      })
  }

  const handleCloseDialog = () => {
    closeDialog()
  }

  const { openDialog, closeDialog } = useConfirmationDialog({
    titleText: getString('exemptionList.deleteModal.title'),
    contentText: (
      <DeleteModalContent
        entity="exemption"
        value={packageName}
        onSubmit={handleDeleteExemption}
        onClose={handleCloseDialog}
        content={getString('exemptionList.deleteModal.contentText')}
        placeholder={getString('exemptionList.deleteModal.inputPlaceholder')}
        inputLabel={getString('exemptionList.deleteModal.inputLabel')}
        forceDelete
      />
    ),
    customButtons: <></>,
    intent: Intent.DANGER,
    onCloseDialog: handleCloseDialog
  })

  return { triggerDelete: openDialog }
}
