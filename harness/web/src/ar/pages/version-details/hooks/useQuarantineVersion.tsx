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

import React, { useRef, useState } from 'react'
import { Intent } from '@blueprintjs/core'
import { useQuarantineFilePathMutation } from '@harnessio/react-har-service-client'
import { getErrorInfoFromErrorObject, Layout, Text, useToaster } from '@harnessio/uicore'

import type { FormikRef } from '@ar/common/types'
import { useStrings } from '@ar/frameworks/strings'
import { useConfirmationDialog, useGetSpaceRef } from '@ar/hooks'
import type { QuarantineVersionFormData } from '../components/QuarantineForm/type'
import QuarantineVersionForm from '../components/QuarantineForm/QuarantineForm'

interface useQuarantineVersionProps {
  repoKey: string
  artifactKey: string
  versionKey: string
  onSuccess: () => void
}

function useQuarantineVersion(props: useQuarantineVersionProps) {
  const { repoKey, artifactKey, versionKey, onSuccess } = props
  const [submitting, setSubmitting] = useState(false)
  const formikRef = useRef<FormikRef<QuarantineVersionFormData>>(null)
  const { getString } = useStrings()
  const { showSuccess, showError, clear } = useToaster()
  const spaceRef = useGetSpaceRef(repoKey)

  const { mutateAsync: quarantineVersion } = useQuarantineFilePathMutation()

  const handleSubmitQuarantineVersion = async (values: QuarantineVersionFormData): Promise<void> => {
    setSubmitting(true)
    try {
      const response = await quarantineVersion({
        body: {
          artifact: artifactKey,
          version: versionKey,
          reason: values.reason
        },
        registry_ref: spaceRef
      })
      if (response.content.status === 'SUCCESS') {
        clear()
        showSuccess(getString('versionDetails.quarantineVersionModal.versionQuarantined'))
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
      formikRef.current?.submitForm()
    } else {
      closeDialog()
    }
  }

  const { openDialog, closeDialog } = useConfirmationDialog({
    titleText: getString('versionDetails.quarantineVersionModal.title'),
    contentText: (
      <Layout.Vertical spacing="medium">
        <Text>{getString('versionDetails.quarantineVersionModal.contentText')}</Text>
        <QuarantineVersionForm onSubmit={handleSubmitQuarantineVersion} ref={formikRef} />
      </Layout.Vertical>
    ),
    confirmButtonText: getString('versionDetails.quarantineVersionModal.confirmButtonText'),
    cancelButtonText: getString('versionDetails.quarantineVersionModal.cancelButtonText'),
    intent: Intent.WARNING,
    onCloseDialog: handleModalButtonClick,
    buttonDisabled: submitting
  })

  return { triggerQuarantine: openDialog }
}

export default useQuarantineVersion
