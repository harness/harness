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

import React, { useState } from 'react'
import { useHistory, useParams } from 'react-router-dom'
import type { Webhook } from '@harnessio/react-har-service-client'

import { useStrings } from '@ar/frameworks/strings'
import { useParentComponents, useRoutes } from '@ar/hooks'
import type { RepositoryDetailsTabPathParams } from '@ar/routes/types'
import ActionButton from '@ar/components/ActionButton/ActionButton'
import { WebhookDetailsTab } from '@ar/pages/webhook-details/constants'
import { PermissionIdentifier, ResourceType } from '@ar/common/permissionTypes'

import DeleteWebhookAction from './DeleteAction'

interface WebhookActionsProps {
  data: Webhook
  readonly?: boolean
}

export default function WebhookActions(props: WebhookActionsProps) {
  const { data } = props
  const [open, setOpen] = useState(false)
  const { getString } = useStrings()
  const { RbacMenuItem } = useParentComponents()
  const routes = useRoutes()
  const history = useHistory()
  const params = useParams<RepositoryDetailsTabPathParams>()

  const handleEdit = (tab: WebhookDetailsTab) => {
    history.push(
      routes.toARRepositoryWebhookDetailsTab({
        ...params,
        webhookIdentifier: data.identifier,
        tab
      })
    )
  }
  return (
    <ActionButton isOpen={open} setOpen={setOpen}>
      <RbacMenuItem
        icon="edit"
        text={getString('actions.edit')}
        onClick={() => {
          setOpen(false)
          handleEdit(WebhookDetailsTab.Configuration)
        }}
        permission={{
          resource: {
            resourceType: ResourceType.ARTIFACT_REGISTRY,
            resourceIdentifier: params.repositoryIdentifier
          },
          permission: PermissionIdentifier.VIEW_ARTIFACT_REGISTRY
        }}
      />
      <RbacMenuItem
        icon="execution"
        text={getString('actions.executions')}
        onClick={() => {
          setOpen(false)
          handleEdit(WebhookDetailsTab.Executions)
        }}
        permission={{
          resource: {
            resourceType: ResourceType.ARTIFACT_REGISTRY,
            resourceIdentifier: params.repositoryIdentifier
          },
          permission: PermissionIdentifier.VIEW_ARTIFACT_REGISTRY
        }}
      />
      <DeleteWebhookAction
        data={data}
        onClick={() => {
          setOpen(false)
        }}
      />
    </ActionButton>
  )
}
