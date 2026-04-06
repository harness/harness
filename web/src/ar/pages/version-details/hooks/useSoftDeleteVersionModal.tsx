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
import { useDeleteVersionV3Mutation } from '@harnessio/react-har-service-client'

import { useAppStore, useConfirmationDialog } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'
import DeleteModalContent, { DeleteFormValues } from '@ar/components/Form/DeleteModalContent'

interface useSoftDeleteVersionModalProps {
  versionKey: string
  uuid: string
  artifactKey: string
  onSuccess: (isForceDeleted: boolean) => void
}
export default function useSoftDeleteVersionModal(props: useSoftDeleteVersionModalProps) {
  const { onSuccess, versionKey, uuid, artifactKey } = props
  const { getString } = useStrings()
  const { showSuccess, showError, clear } = useToaster()
  const { scope } = useAppStore()
  const { accountId } = scope
  const { mutateAsync: softDeleteVersion } = useDeleteVersionV3Mutation()

  const handleSoftDeleteVersion = async (values: DeleteFormValues): Promise<void> => {
    try {
      await softDeleteVersion({
        id: uuid,
        queryParams: {
          account_identifier: accountId as string,
          force: values.force
        }
      })
      clear()
      showSuccess(getString('versionDetails.versionArchived'))
      onSuccess(values.force)
      closeDialog()
    } catch (e: any) {
      showError(getErrorInfoFromErrorObject(e?.error || e, true))
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
        value={`${artifactKey}@${versionKey}`}
        onSubmit={handleSoftDeleteVersion}
        onClose={handleCloseDialog}
        content={getString('versionDetails.softDeleteModal.contentText')}
        placeholder={`${artifactKey}@${versionKey}`}
        inputLabel={getString('versionDetails.softDeleteModal.inputLabel')}
        forceLabel={getString('versionDetails.softDeleteModal.forceLabel')}
        forceSubText={getString('versionDetails.softDeleteModal.forceSubText')}
      />
    ),
    customButtons: <></>,
    intent: Intent.DANGER,
    onCloseDialog: handleCloseDialog
  })

  return { triggerSoftDelete: openDialog }
}
