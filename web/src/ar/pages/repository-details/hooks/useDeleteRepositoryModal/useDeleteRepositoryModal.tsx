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
import { useDeleteRegistryMutation } from '@harnessio/react-har-service-client'

import { useStrings } from '@ar/frameworks/strings'
import { useGetSpaceRef, useParentHooks } from '@ar/hooks'
import DeleteModalContent from '@ar/components/Form/DeleteModalContent'

interface useDeleteRepositoryModalProps {
  repoKey: string
  onSuccess: () => void
}
export default function useDeleteRepositoryModal(props: useDeleteRepositoryModalProps) {
  const { repoKey, onSuccess } = props
  const { getString } = useStrings()
  const { showSuccess, showError, clear } = useToaster()
  const { useConfirmationDialog } = useParentHooks()
  const spaceRef = useGetSpaceRef(repoKey)

  const { mutateAsync: deleteRepository } = useDeleteRegistryMutation()

  const handleDeleteRepository = async (): Promise<void> => {
    try {
      const response = await deleteRepository({
        registry_ref: spaceRef
      })
      if (response.content.status === 'SUCCESS') {
        clear()
        showSuccess(getString('repositoryDetails.repositoryForm.repositoryDeleted'))
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
    titleText: getString('repositoryList.deleteModal.title'),
    contentText: (
      <DeleteModalContent
        entity="repository"
        value={repoKey}
        onSubmit={handleDeleteRepository}
        onClose={handleCloseDialog}
        content={getString('repositoryList.deleteModal.contentText')}
        placeholder={getString('repositoryList.deleteModal.inputPlaceholder')}
        inputLabel={getString('repositoryList.deleteModal.inputLabel')}
      />
    ),
    customButtons: <></>,
    intent: Intent.DANGER,
    onCloseDialog: handleCloseDialog
  })

  return { triggerDelete: openDialog }
}
