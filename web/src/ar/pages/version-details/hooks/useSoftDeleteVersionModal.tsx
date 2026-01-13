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
import { getErrorInfoFromErrorObject, useToaster } from '@harnessio/uicore'
import { useDeleteVersionMutation } from '@harnessio/react-har-service-client'

import { useAppStore, useConfirmationDialog } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'
import DeleteModalContent from '@ar/components/Form/DeleteModalContent'

interface useSoftDeleteVersionModalProps {
  versionKey: string
  uuid: string
  onSuccess: () => void
}
export default function useSoftDeleteVersionModal(props: useSoftDeleteVersionModalProps) {
  const { onSuccess, versionKey, uuid } = props
  const { getString } = useStrings()
  const { showSuccess, showError, clear } = useToaster()
  const { scope } = useAppStore()
  const { accountId, projectIdentifier, orgIdentifier } = scope

  const { mutateAsync: softDeleteVersion } = useDeleteVersionMutation()

  const handleSoftDeleteVersion = async (): Promise<void> => {
    try {
      const response = await softDeleteVersion({
        uuid,
        queryParams: {
          account_identifier: accountId as string,
          org_identifier: orgIdentifier,
          project_identifier: projectIdentifier
        }
      })
      if (response.content.status === 'SUCCESS') {
        clear()
        showSuccess(getString('versionDetails.versionArchived'))
        onSuccess()
        closeDialog()
      }
    } catch (e: any) {
      showError(getErrorInfoFromErrorObject(e, true))
    }
  }

  const handleCloseDialog = () => {
    closeDialog()
  }

  const { openDialog, closeDialog } = useConfirmationDialog({
    titleText: getString('versionDetails.softDeleteModal.title'),
    contentText: (
      <DeleteModalContent
        entity="version"
        value={versionKey}
        onSubmit={handleSoftDeleteVersion}
        onClose={handleCloseDialog}
        content={getString('versionDetails.softDeleteModal.contentText')}
        placeholder={getString('versionDetails.softDeleteModal.inputPlaceholder')}
        inputLabel={getString('versionDetails.softDeleteModal.inputLabel')}
        deleteBtnText={getString('actions.softDelete')}
      />
    ),
    customButtons: <></>,
    intent: Intent.DANGER,
    onCloseDialog: handleCloseDialog
  })

  return { triggerSoftDelete: openDialog }
}
