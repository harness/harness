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
import { useDeletePackageMutation } from '@harnessio/react-har-service-client'

import { useAppStore, useParentHooks } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'
import DeleteModalContent from '@ar/components/Form/DeleteModalContent'

interface useSoftDeleteArtifactModalProps {
  artifactKey: string
  onSuccess: () => void
  uuid: string
}
export default function useSoftDeleteArtifactModal(props: useSoftDeleteArtifactModalProps) {
  const { onSuccess, artifactKey, uuid } = props
  const { getString } = useStrings()
  const { showSuccess, showError, clear } = useToaster()
  const { useConfirmationDialog } = useParentHooks()
  const { scope } = useAppStore()
  const { accountId, orgIdentifier, projectIdentifier } = scope

  const { mutateAsync: softDeleteArtifact } = useDeletePackageMutation()

  const handleSoftDeleteArtifact = async (): Promise<void> => {
    try {
      const response = await softDeleteArtifact({
        queryParams: {
          account_identifier: accountId as string,
          org_identifier: orgIdentifier,
          project_identifier: projectIdentifier,
          force: false
        },
        uuid
      })
      if (response.content.status === 'SUCCESS') {
        clear()
        showSuccess(getString('artifactDetails.packageArchived'))
        onSuccess()
      }
    } catch (e: any) {
      showError(getErrorInfoFromErrorObject(e, true))
    }
  }

  const handleCloseDialog = () => {
    closeDialog()
  }

  const { openDialog, closeDialog } = useConfirmationDialog({
    titleText: getString('artifactDetails.softDeleteModal.title'),
    contentText: (
      <DeleteModalContent
        entity="package"
        value={artifactKey}
        onSubmit={handleSoftDeleteArtifact}
        onClose={handleCloseDialog}
        content={getString('artifactDetails.softDeleteModal.contentText')}
        placeholder={getString('artifactDetails.softDeleteModal.inputPlaceholder')}
        inputLabel={getString('artifactDetails.softDeleteModal.inputLabel')}
        deleteBtnText={getString('actions.softDelete')}
      />
    ),
    customButtons: <></>,
    intent: Intent.DANGER,
    onCloseDialog: handleCloseDialog
  })

  return { triggerSoftDelete: openDialog }
}
