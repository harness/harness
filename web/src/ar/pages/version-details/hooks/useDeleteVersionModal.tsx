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
import { useDeleteArtifactVersionMutation } from '@harnessio/react-har-service-client'

import { useGetSpaceRef, useParentHooks } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'
import { encodeRef } from '@ar/hooks/useGetSpaceRef'

interface useDeleteVersionModalProps {
  repoKey: string
  artifactKey: string
  versionKey: string
  onSuccess: () => void
}
export default function useDeleteVersionModal(props: useDeleteVersionModalProps) {
  const { repoKey, onSuccess, artifactKey, versionKey } = props
  const { getString } = useStrings()
  const { showSuccess, showError, clear } = useToaster()
  const { useConfirmationDialog } = useParentHooks()
  const spaceRef = useGetSpaceRef(repoKey)

  const { mutateAsync: deleteVersion } = useDeleteArtifactVersionMutation()

  const handleDeleteVersion = async (isConfirmed: boolean): Promise<void> => {
    if (isConfirmed) {
      try {
        const response = await deleteVersion({
          registry_ref: spaceRef,
          artifact: encodeRef(artifactKey),
          version: versionKey
        })
        if (response.content.status === 'SUCCESS') {
          clear()
          showSuccess(getString('versionDetails.versionDeleted'))
          onSuccess()
        }
      } catch (e: any) {
        showError(getErrorInfoFromErrorObject(e, true))
      }
    }
  }

  const { openDialog } = useConfirmationDialog({
    titleText: getString('versionDetails.deleteVersionModal.title'),
    contentText: getString('versionDetails.deleteVersionModal.contentText'),
    confirmButtonText: getString('delete'),
    cancelButtonText: getString('cancel'),
    intent: Intent.DANGER,
    onCloseDialog: handleDeleteVersion
  })

  return { triggerDelete: openDialog }
}
