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
import { useAddOciArtifactTagsMutation } from '@harnessio/react-har-service-client'

import { useAppStore, useConfirmationDialog } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'
import { queryClient } from '@ar/utils/queryClient'

import { AddTagModalContent } from '../components/VersionActions/AddTagModalContent'

interface UseAddTagModalProps {
  repoKey: string
  artifactKey: string
  versionKey: string
  onSuccess?: () => void
}

export default function useAddTagModal(props: UseAddTagModalProps) {
  const { repoKey, artifactKey, versionKey, onSuccess } = props
  const { getString } = useStrings()
  const { showSuccess, showError, clear } = useToaster()
  const { scope } = useAppStore()

  const { mutateAsync: addOciArtifactTags, isLoading: loading } = useAddOciArtifactTagsMutation()

  const handleAddTag = async (tagNames: string[]): Promise<void> => {
    try {
      await addOciArtifactTags({
        queryParams: {
          account_identifier: typeof scope?.accountId === 'string' ? scope.accountId : '',
          org_identifier: typeof scope?.orgId === 'string' ? scope.orgId : undefined,
          project_identifier: typeof scope?.projectId === 'string' ? scope.projectId : undefined,
          registry_identifier: repoKey
        },
        body: {
          package: artifactKey,
          version: versionKey,
          tags: tagNames
        }
      })
      clear()
      showSuccess(getString('versionList.messages.addTagSuccess'))
      onSuccess?.()
      closeDialog()
      queryClient.invalidateQueries(['GetAllHarnessArtifacts'])
      queryClient.invalidateQueries(['ListVersions'])
    } catch (e: any) {
      showError(getErrorInfoFromErrorObject(e, true))
    }
  }

  const handleCloseDialog = () => {
    closeDialog()
  }

  const { openDialog, closeDialog } = useConfirmationDialog({
    titleText: getString('versionList.actions.addTag'),
    contentText: (
      <AddTagModalContent
        onSubmit={handleAddTag}
        onClose={handleCloseDialog}
        label={getString('versionList.table.columns.tags')}
        placeholder={getString('versionList.actions.addTagPlaceholder')}
        addButtonText={getString('add')}
        disabled={loading}
        instructions={getString('versionList.actions.addTagDescription')}
      />
    ),
    customButtons: <></>,
    onCloseDialog: handleCloseDialog
  })

  return { openAddTagModal: openDialog }
}
