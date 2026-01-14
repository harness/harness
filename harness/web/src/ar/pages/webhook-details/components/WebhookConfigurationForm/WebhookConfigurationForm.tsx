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

import React, { useContext } from 'react'
import type { FormikProps } from 'formik'
import { useParams } from 'react-router-dom'
import { updateWebhook, WebhookRequest } from '@harnessio/react-har-service-client'
import { Container, getErrorInfoFromErrorObject, useToaster } from '@harnessio/uicore'

import { useAppStore, useGetSpaceRef, useParentHooks } from '@ar/hooks'
import { queryClient } from '@ar/utils/queryClient'
import { useStrings } from '@ar/frameworks/strings'
import WebhookForm from '@ar/pages/webhook-list/components/Forms/WebhookForm'
import type { RepositoryWebhookDetailsTabPathParams } from '@ar/routes/types'
import { PermissionIdentifier, ResourceType } from '@ar/common/permissionTypes'

import { WebhookDetailsContext } from '../../context/WebhookDetailsContext'
import css from './WebhookConfigurationForm.module.scss'

interface WebhookConfigurationFormProps {
  formRef: React.RefObject<FormikProps<WebhookRequest>>
}

export default function WebhookConfigurationForm(props: WebhookConfigurationFormProps): JSX.Element {
  const { formRef } = props
  const { showError, showSuccess, clear } = useToaster()
  const { getString } = useStrings()
  const { usePermission } = useParentHooks()
  const { scope } = useAppStore()
  const { accountId, orgIdentifier, projectIdentifier } = scope

  const { data, setDirty, setUpdating, isInternalWebhook } = useContext(WebhookDetailsContext)
  const registryRef = useGetSpaceRef()
  const { webhookIdentifier, repositoryIdentifier } = useParams<RepositoryWebhookDetailsTabPathParams>()

  const [isEditPermission] = usePermission(
    {
      resourceScope: {
        accountIdentifier: accountId,
        orgIdentifier,
        projectIdentifier
      },
      resource: {
        resourceType: ResourceType.ARTIFACT_REGISTRY,
        resourceIdentifier: repositoryIdentifier
      },
      permissions: [PermissionIdentifier.EDIT_ARTIFACT_REGISTRY]
    },
    [accountId, projectIdentifier, orgIdentifier, repositoryIdentifier]
  )

  const handleUpdateWebhook = async (values: WebhookRequest) => {
    try {
      setUpdating?.(true)
      await updateWebhook({
        registry_ref: registryRef,
        webhook_identifier: webhookIdentifier,
        body: values
      })
      showSuccess(getString('webhookList.webhookUpdated'))
      queryClient.invalidateQueries(['GetWebhook'])
    } catch (e) {
      clear()
      showError(getErrorInfoFromErrorObject(e as Error))
    } finally {
      setUpdating?.(false)
    }
  }

  if (!data) return <></>
  return (
    <Container className={css.formContainer} padding="xlarge">
      <WebhookForm
        data={data}
        ref={formRef}
        readonly={!isEditPermission || isInternalWebhook}
        isEdit
        onSubmit={handleUpdateWebhook}
        setDirty={setDirty}
      />
    </Container>
  )
}
