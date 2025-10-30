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
import { useParams } from 'react-router-dom'
import { getErrorInfoFromErrorObject, useToaster } from '@harnessio/uicore'
import { deleteWebhook, type Webhook } from '@harnessio/react-har-service-client'

import { useStrings } from '@ar/frameworks/strings'
import { queryClient } from '@ar/utils/queryClient'
import { useGetSpaceRef, useParentComponents, useParentHooks } from '@ar/hooks'
import type { RepositoryDetailsTabPathParams } from '@ar/routes/types'
import { PermissionIdentifier, ResourceType } from '@ar/common/permissionTypes'

interface DeleteWebhookActionProps {
  data: Webhook
  onClick?: () => void
}

export default function DeleteWebhookAction(props: DeleteWebhookActionProps) {
  const { data, onClick } = props
  const { RbacMenuItem } = useParentComponents()
  const { getString } = useStrings()
  const registryRef = useGetSpaceRef()
  const { showError, showSuccess, clear } = useToaster()
  const { useConfirmationDialog } = useParentHooks()
  const params = useParams<RepositoryDetailsTabPathParams>()

  const handleDeleteWebhook = async (isConfirmed: boolean) => {
    if (!isConfirmed) {
      onClick?.()
      return
    }
    try {
      await deleteWebhook({
        registry_ref: registryRef,
        webhook_identifier: data.identifier
      })
      clear()
      showSuccess(getString('webhookList.webhookDeleted'))
      queryClient.invalidateQueries(['ListWebhooks'])
    } catch (e) {
      clear()
      showError(getErrorInfoFromErrorObject(e as Error))
    } finally {
      onClick?.()
    }
  }

  const { openDialog } = useConfirmationDialog({
    titleText: getString('webhookList.deleteModal.title'),
    contentText: getString('webhookList.deleteModal.message'),
    confirmButtonText: getString('delete'),
    cancelButtonText: getString('cancel'),
    intent: Intent.DANGER,
    onCloseDialog: handleDeleteWebhook
  })

  return (
    <>
      <RbacMenuItem
        icon="code-delete"
        onClick={() => {
          openDialog()
        }}
        text={getString('actions.delete')}
        disabled={data.internal}
        permission={{
          resource: {
            resourceType: ResourceType.ARTIFACT_REGISTRY,
            resourceIdentifier: params.repositoryIdentifier
          },
          permission: PermissionIdentifier.DELETE_ARTIFACT_REGISTRY
        }}
      />
    </>
  )
}
