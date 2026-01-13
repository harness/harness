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
import { ArtifactType, useDeleteArtifactMutation } from '@harnessio/react-har-service-client'

import { useGetSpaceRef, useParentHooks } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'
import { encodeRef } from '@ar/hooks/useGetSpaceRef'
import DeleteModalContent from '@ar/components/Form/DeleteModalContent'

interface useDeleteArtifactModalProps {
  repoKey: string
  artifactKey: string
  artifactType?: ArtifactType
  onSuccess: () => void
}
export default function useDeleteArtifactModal(props: useDeleteArtifactModalProps) {
  const { repoKey, onSuccess, artifactKey, artifactType } = props
  const { getString } = useStrings()
  const { showSuccess, showError, clear } = useToaster()
  const { useConfirmationDialog } = useParentHooks()
  const spaceRef = useGetSpaceRef(repoKey)

  const { mutateAsync: deleteArtifact } = useDeleteArtifactMutation()

  const handleDeleteArtifact = async (): Promise<void> => {
    try {
      const response = await deleteArtifact({
        registry_ref: spaceRef,
        artifact: encodeRef(artifactKey),
        queryParams: {
          artifact_type: artifactType
        }
      })
      if (response.content.status === 'SUCCESS') {
        clear()
        showSuccess(getString('artifactDetails.artifactDeleted'))
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
    titleText: getString('artifactDetails.deleteModal.title'),
    contentText: (
      <DeleteModalContent
        entity="package"
        value={artifactKey}
        onSubmit={handleDeleteArtifact}
        onClose={handleCloseDialog}
        content={getString('artifactDetails.deleteModal.contentText')}
        placeholder={getString('artifactDetails.deleteModal.inputPlaceholder')}
        inputLabel={getString('artifactDetails.deleteModal.inputLabel')}
      />
    ),
    customButtons: <></>,
    intent: Intent.DANGER,
    onCloseDialog: handleCloseDialog
  })

  return { triggerDelete: openDialog }
}
