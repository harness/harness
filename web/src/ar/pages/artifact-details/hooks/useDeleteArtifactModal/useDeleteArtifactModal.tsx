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

import { Intent } from '@blueprintjs/core'
import { getErrorInfoFromErrorObject, useToaster } from '@harnessio/uicore'
import { useDeleteArtifactMutation } from '@harnessio/react-har-service-client'

import { useGetSpaceRef, useParentHooks } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'
import { encodeRef } from '@ar/hooks/useGetSpaceRef'

interface useDeleteArtifactModalProps {
  repoKey: string
  artifactKey: string
  onSuccess: () => void
}
export default function useDeleteArtifactModal(props: useDeleteArtifactModalProps) {
  const { repoKey, onSuccess, artifactKey } = props
  const { getString } = useStrings()
  const { showSuccess, showError, clear } = useToaster()
  const { useConfirmationDialog } = useParentHooks()
  const spaceRef = useGetSpaceRef(repoKey)

  const { mutateAsync: deleteArtifact } = useDeleteArtifactMutation()

  const handleDeleteArtifact = async (isConfirmed: boolean): Promise<void> => {
    if (isConfirmed) {
      try {
        const response = await deleteArtifact({
          registry_ref: spaceRef,
          artifact: encodeRef(artifactKey)
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
  }

  const { openDialog } = useConfirmationDialog({
    titleText: getString('artifactDetails.deleteArtifactModal.title'),
    contentText: getString('artifactDetails.deleteArtifactModal.contentText'),
    confirmButtonText: getString('delete'),
    cancelButtonText: getString('cancel'),
    intent: Intent.DANGER,
    onCloseDialog: handleDeleteArtifact
  })

  return { triggerDelete: openDialog }
}
