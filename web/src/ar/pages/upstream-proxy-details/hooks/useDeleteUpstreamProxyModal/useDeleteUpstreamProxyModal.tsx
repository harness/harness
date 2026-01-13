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
import { getErrorInfoFromErrorObject, useToaster } from '@harnessio/uicore'
import { Intent } from '@harnessio/design-system'
import { useDeleteRegistryMutation } from '@harnessio/react-har-service-client'

import { useGetSpaceRef, useParentHooks } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'
import DeleteModalContent from '@ar/components/Form/DeleteModalContent'

interface useDeleteUpstreamProxyModalProps {
  repoKey: string
  onSuccess: () => void
}

export default function useDeleteUpstreamProxyModal(props: useDeleteUpstreamProxyModalProps) {
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
    titleText: getString('upstreamProxyDetails.actions.delete.title'),
    contentText: (
      <DeleteModalContent
        entity="repository"
        value={repoKey}
        onSubmit={handleDeleteRepository}
        onClose={handleCloseDialog}
        content={getString('upstreamProxyDetails.actions.delete.contentText')}
        placeholder={getString('upstreamProxyDetails.actions.delete.inputPlaceholder')}
        inputLabel={getString('upstreamProxyDetails.actions.delete.inputLabel')}
      />
    ),
    customButtons: <></>,
    intent: Intent.DANGER,
    onCloseDialog: handleCloseDialog
  })

  return { triggerDelete: openDialog }
}
