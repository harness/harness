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
import type { FormikProps } from 'formik'
import type { IDialogProps } from '@blueprintjs/core'
import { createWebhook, Webhook, WebhookRequest } from '@harnessio/react-har-service-client'
import { Button, ButtonVariation, Layout, ModalDialog, useToaster } from '@harnessio/uicore'

import { useGetSpaceRef } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'

import { getErrorMessage } from 'utils/Utils'
import WebhookForm from '../Forms/WebhookForm'

interface CreateWebhookModalProps extends IDialogProps {
  onSubmit: (res: Webhook) => void
}

export default function CreateWebhookModal(props: CreateWebhookModalProps) {
  const { onSubmit, isOpen, onClose, title } = props
  const [isLoading, setLoading] = useState(false)
  const { getString } = useStrings()

  const registryRef = useGetSpaceRef()

  const { showError, clear, showSuccess } = useToaster()

  const formRef = useRef<FormikProps<WebhookRequest>>(null)

  const handleSubmit = () => {
    formRef.current?.submitForm()
  }

  const handleCreateWebhook = async (formData: WebhookRequest) => {
    try {
      setLoading(true)
      const response = await createWebhook({
        registry_ref: registryRef,
        body: formData
      })
      showSuccess(getString('webhookList.webhookCreated'))
      onSubmit(response.content.data)
    } catch (e) {
      clear()
      showError(getErrorMessage(e))
    } finally {
      setLoading(false)
    }
  }

  return (
    <ModalDialog
      title={title ?? getString('webhookList.newWebhook')}
      onClose={onClose}
      isOpen={isOpen}
      showOverlay={isLoading}
      footer={
        <Layout.Horizontal spacing="small">
          <Button variation={ButtonVariation.PRIMARY} onClick={handleSubmit}>
            {getString('add')}
          </Button>
          <Button variation={ButtonVariation.SECONDARY} onClick={() => onClose?.()}>
            {getString('cancel')}
          </Button>
        </Layout.Horizontal>
      }>
      <WebhookForm onSubmit={handleCreateWebhook} ref={formRef} />
    </ModalDialog>
  )
}
