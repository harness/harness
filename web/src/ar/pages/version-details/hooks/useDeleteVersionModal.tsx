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
import { ArtifactType, useDeleteArtifactVersionMutation } from '@harnessio/react-har-service-client'

import { useConfirmationDialog, useGetSpaceRef } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'
import { encodeRef } from '@ar/hooks/useGetSpaceRef'
import DeleteModalContent from '@ar/components/Form/DeleteModalContent'

interface useDeleteVersionModalProps {
  repoKey: string
  artifactKey: string
  versionKey: string
  artifactType?: ArtifactType
  onSuccess: () => void
}
export default function useDeleteVersionModal(props: useDeleteVersionModalProps) {
  const { repoKey, onSuccess, artifactKey, versionKey, artifactType } = props
  const { getString } = useStrings()
  const { showSuccess, showError, clear } = useToaster()
  const spaceRef = useGetSpaceRef(repoKey)

  const { mutateAsync: deleteVersion } = useDeleteArtifactVersionMutation()

  const handleDeleteVersion = async (): Promise<void> => {
    try {
      const response = await deleteVersion({
        registry_ref: spaceRef,
        artifact: encodeRef(artifactKey),
        version: versionKey,
        queryParams: {
          artifact_type: artifactType
        }
      })
      if (response.content.status === 'SUCCESS') {
        clear()
        showSuccess(getString('versionDetails.versionDeleted'))
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
    titleText: getString('versionDetails.deleteVersionModal.title'),
    contentText: (
      <DeleteModalContent
        entity="version"
        value={versionKey}
        onSubmit={handleDeleteVersion}
        onClose={handleCloseDialog}
        content={getString('versionDetails.deleteVersionModal.contentText')}
        placeholder={getString('versionDetails.deleteVersionModal.inputPlaceholder')}
        inputLabel={getString('versionDetails.deleteVersionModal.inputLabel')}
      />
    ),
    customButtons: <></>,
    intent: Intent.DANGER,
    onCloseDialog: handleCloseDialog
  })

  return { triggerDelete: openDialog }
}
