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
import { useDeleteRegistryV3Mutation } from '@harnessio/react-har-service-client'

import { useAppStore, useParentHooks } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'
import DeleteModalContent, { DeleteFormValues } from '@ar/components/Form/DeleteModalContent'

interface useSoftDeleteRepositoryModalProps {
  repoKey: string
  onSuccess: (isForceDeleted: boolean) => void
  uuid: string
}
export default function useSoftDeleteRepositoryModal(props: useSoftDeleteRepositoryModalProps) {
  const { repoKey, onSuccess, uuid } = props
  const { getString } = useStrings()
  const { showSuccess, showError, clear } = useToaster()
  const { useConfirmationDialog } = useParentHooks()
  const { scope } = useAppStore()
  const { accountId } = scope

  const { mutateAsync: deleteRepository } = useDeleteRegistryV3Mutation()

  const handleDeleteRepository = async (values: DeleteFormValues): Promise<void> => {
    try {
      await deleteRepository({
        queryParams: {
          account_identifier: accountId as string,
          force: values.force
        },
        id: uuid
      })
      clear()
      showSuccess(getString('repositoryDetails.repositoryForm.repositorySoftDeleted'))
      onSuccess(values.force)
    } catch (e: any) {
      showError(getErrorInfoFromErrorObject(e?.error || e, true))
    }
  }

  const handleCloseDialog = () => {
    closeDialog()
  }

  const { openDialog, closeDialog } = useConfirmationDialog({
    titleText: getString('repositoryList.softDeleteModal.title'),
    contentText: (
      <DeleteModalContent
        entity="repository"
        value={repoKey}
        onSubmit={handleDeleteRepository}
        onClose={handleCloseDialog}
        content={getString('repositoryList.softDeleteModal.contentText')}
        placeholder={repoKey}
        inputLabel={getString('repositoryList.softDeleteModal.inputLabel')}
        forceLabel={getString('repositoryList.softDeleteModal.forceLabel')}
        forceSubText={getString('repositoryList.softDeleteModal.forceSubText')}
      />
    ),
    customButtons: <></>,
    intent: Intent.DANGER,
    onCloseDialog: handleCloseDialog
  })

  return { triggerDelete: openDialog }
}
