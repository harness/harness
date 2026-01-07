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

import { useState } from 'react'
import { Intent } from '@blueprintjs/core'
import { useDeleteQuarantineFilePathMutation } from '@harnessio/react-har-service-client'
import { getErrorInfoFromErrorObject, useToaster } from '@harnessio/uicore'

import { useStrings } from '@ar/frameworks/strings'
import { useConfirmationDialog, useGetSpaceRef } from '@ar/hooks'

interface useRemoveQuarantineVersionProps {
  repoKey: string
  artifactKey: string
  versionKey: string
  onSuccess: () => void
}

function useRemoveQuarantineVersion(props: useRemoveQuarantineVersionProps) {
  const { repoKey, artifactKey, versionKey, onSuccess } = props
  const [submitting, setSubmitting] = useState(false)
  const { getString } = useStrings()
  const { showSuccess, showError, clear } = useToaster()
  const spaceRef = useGetSpaceRef(repoKey)

  const { mutateAsync: removeQuarantineVersion } = useDeleteQuarantineFilePathMutation()

  const handleSubmitRemoveQuarantineVersion = async (): Promise<void> => {
    setSubmitting(true)
    try {
      const response = await removeQuarantineVersion({
        queryParams: {
          artifact: artifactKey,
          version: versionKey
        },
        registry_ref: spaceRef
      })
      if (response.content.status === 'SUCCESS') {
        clear()
        showSuccess(getString('versionDetails.removeFromQuarantineModal.versionRemovedFromQuarantine'))
        onSuccess()
        closeDialog()
      }
    } catch (e: any) {
      showError(getErrorInfoFromErrorObject(e, true))
    } finally {
      setSubmitting(false)
    }
  }

  const handleModalButtonClick = (isConfirmed: boolean) => {
    if (isConfirmed) {
      handleSubmitRemoveQuarantineVersion()
    } else {
      closeDialog()
    }
  }

  const { openDialog, closeDialog } = useConfirmationDialog({
    titleText: getString('versionDetails.removeFromQuarantineModal.title'),
    contentText: getString('versionDetails.removeFromQuarantineModal.contentText'),
    confirmButtonText: getString('versionDetails.removeFromQuarantineModal.confirmButtonText'),
    cancelButtonText: getString('versionDetails.removeFromQuarantineModal.cancelButtonText'),
    intent: Intent.WARNING,
    onCloseDialog: handleModalButtonClick,
    buttonDisabled: submitting
  })

  return { triggerRemoveQuarantine: openDialog }
}

export default useRemoveQuarantineVersion
