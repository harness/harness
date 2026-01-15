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
import { getErrorInfoFromErrorObject, useToaster } from '@harnessio/uicore'
import { useRestoreVersionMutation } from '@harnessio/react-har-service-client'

import { useAppStore, useConfirmationDialog } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'

interface useRestoreDeletedVersionModalProps {
  uuid: string
  onSuccess: () => void
}
export default function useRestoreDeletedVersionModal(props: useRestoreDeletedVersionModalProps) {
  const { onSuccess, uuid } = props
  const [submitting, setSubmitting] = useState(false)
  const { getString } = useStrings()
  const { showSuccess, showError, clear } = useToaster()
  const { scope } = useAppStore()
  const { accountId, projectIdentifier, orgIdentifier } = scope

  const { mutateAsync: restoreVersion } = useRestoreVersionMutation()

  const handleRestoreVersion = async (isConfirmed: boolean): Promise<void> => {
    if (isConfirmed) {
      setSubmitting(true)
      try {
        const response = await restoreVersion({
          queryParams: {
            account_identifier: accountId as string,
            org_identifier: orgIdentifier,
            project_identifier: projectIdentifier
          },
          uuid
        })
        if (response.content.status === 'SUCCESS') {
          clear()
          showSuccess(getString('versionDetails.versionRestored'))
          onSuccess()
          closeDialog()
        }
      } catch (e: any) {
        showError(getErrorInfoFromErrorObject(e, true))
      } finally {
        setSubmitting(false)
      }
    } else {
      closeDialog()
    }
  }

  const { openDialog, closeDialog } = useConfirmationDialog({
    titleText: getString('versionDetails.restoreModal.title'),
    contentText: getString('versionDetails.restoreModal.contentText'),
    confirmButtonText: getString('restore'),
    cancelButtonText: getString('cancel'),
    intent: Intent.DANGER,
    onCloseDialog: handleRestoreVersion,
    buttonDisabled: submitting
  })

  return { triggerRestore: openDialog }
}
