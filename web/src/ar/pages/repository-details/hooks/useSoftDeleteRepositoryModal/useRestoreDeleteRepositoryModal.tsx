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
import { useRestoreRegistryV3Mutation } from '@harnessio/react-har-service-client'

import { useAppStore, useParentHooks } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'

interface useRestoreDeleteRepositoryModalProps {
  onSuccess: (isForceDeleted: boolean) => void
  uuid: string
}
export default function useRestoreDeleteRepositoryModal(props: useRestoreDeleteRepositoryModalProps) {
  const { onSuccess, uuid } = props
  const { getString } = useStrings()
  const { showSuccess, showError, clear } = useToaster()
  const { useConfirmationDialog } = useParentHooks()
  const { scope } = useAppStore()
  const { accountId } = scope

  const { mutateAsync: restoreRepository } = useRestoreRegistryV3Mutation()

  const handleRestoreRepository = async (isConfirmed: boolean): Promise<void> => {
    if (isConfirmed) {
      try {
        await restoreRepository({
          queryParams: {
            account_identifier: accountId as string
          },
          id: uuid
        })
        clear()
        showSuccess(getString('repositoryDetails.repositoryForm.repositoryRestored'))
        onSuccess(false)
      } catch (e: any) {
        showError(getErrorInfoFromErrorObject(e?.error || e, true))
      }
    } else {
      closeDialog()
    }
  }

  const { openDialog, closeDialog } = useConfirmationDialog({
    titleText: getString('repositoryList.restoreModal.title'),
    contentText: getString('repositoryList.restoreModal.contentText'),
    confirmButtonText: getString('restore'),
    cancelButtonText: getString('cancel'),
    intent: Intent.DANGER,
    onCloseDialog: handleRestoreRepository
  })

  return { triggerRestore: openDialog }
}
