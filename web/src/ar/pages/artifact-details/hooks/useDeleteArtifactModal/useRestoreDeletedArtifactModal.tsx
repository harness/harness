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
import { useRestorePackageMutation } from '@harnessio/react-har-service-client'

import { useAppStore, useParentHooks } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'

interface useRestoreDeletedArtifactModalProps {
  onSuccess: () => void
  uuid: string
}
export default function useRestoreDeletedArtifactModal(props: useRestoreDeletedArtifactModalProps) {
  const { onSuccess, uuid } = props
  const { getString } = useStrings()
  const { showSuccess, showError, clear } = useToaster()
  const { useConfirmationDialog } = useParentHooks()
  const { scope } = useAppStore()
  const { accountId, orgIdentifier, projectIdentifier } = scope

  const { mutateAsync: restoreDeletedArtifact } = useRestorePackageMutation()

  const handleRestoreDeletedArtifact = async (isConfirmed: boolean): Promise<void> => {
    if (isConfirmed) {
      try {
        const response = await restoreDeletedArtifact({
          queryParams: {
            account_identifier: accountId as string,
            org_identifier: orgIdentifier,
            project_identifier: projectIdentifier
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
  }

  const { openDialog } = useConfirmationDialog({
    titleText: getString('artifactDetails.restoreModal.title'),
    contentText: getString('artifactDetails.restoreModal.contentText'),
    confirmButtonText: getString('restore'),
    cancelButtonText: getString('cancel'),
    intent: Intent.DANGER,
    onCloseDialog: handleRestoreDeletedArtifact
  })

  return { triggerRestore: openDialog }
}
