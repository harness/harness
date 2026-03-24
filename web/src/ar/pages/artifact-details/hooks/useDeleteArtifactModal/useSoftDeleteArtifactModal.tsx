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
import { useDeletePackageV3Mutation } from '@harnessio/react-har-service-client'

import { useAppStore, useParentHooks } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'
import DeleteModalContent, { DeleteFormValues } from '@ar/components/Form/DeleteModalContent'

interface useSoftDeleteArtifactModalProps {
  artifactKey: string
  onSuccess: (isForceDeleted: boolean) => void
  uuid: string
}
export default function useSoftDeleteArtifactModal(props: useSoftDeleteArtifactModalProps) {
  const { onSuccess, artifactKey, uuid } = props
  const { getString } = useStrings()
  const { showSuccess, showError, clear } = useToaster()
  const { useConfirmationDialog } = useParentHooks()
  const { scope } = useAppStore()
  const { accountId } = scope

  const { mutateAsync: softDeleteArtifact } = useDeletePackageV3Mutation()

  const handleSoftDeleteArtifact = async (values: DeleteFormValues): Promise<void> => {
    try {
      await softDeleteArtifact({
        queryParams: {
          account_identifier: accountId as string,
          force: values.force
        },
        id: uuid
      })
      clear()
      showSuccess(getString('artifactDetails.packageArchived'))
      onSuccess(values.force)
    } catch (e: any) {
      showError(getErrorInfoFromErrorObject(e?.error || e, true))
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
        forceLabel={getString('artifactDetails.softDeleteModal.forceLabel')}
        forceSubText={getString('artifactDetails.softDeleteModal.forceSubText')}
      />
    ),
    customButtons: <></>,
    intent: Intent.DANGER,
    onCloseDialog: handleCloseDialog
  })

  return { triggerSoftDelete: openDialog }
}
