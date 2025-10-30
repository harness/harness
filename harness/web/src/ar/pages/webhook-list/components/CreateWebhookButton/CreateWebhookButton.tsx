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
import { useHistory, useParams } from 'react-router-dom'
import type { Webhook } from '@harnessio/react-har-service-client'
import { ButtonVariation, useToggleOpen } from '@harnessio/uicore'

import { useStrings } from '@ar/frameworks/strings'
import { useParentComponents, useRoutes } from '@ar/hooks'
import type { RepositoryDetailsTabPathParams } from '@ar/routes/types'
import { PermissionIdentifier, ResourceType } from '@ar/common/permissionTypes'

import CreateWebhookModal from './CreateWebhookModal'

export default function CreateWebhookButton() {
  const { isOpen, close, open } = useToggleOpen()
  const { getString } = useStrings()
  const { RbacButton } = useParentComponents()
  const { repositoryIdentifier } = useParams<RepositoryDetailsTabPathParams>()

  const routes = useRoutes()
  const history = useHistory()

  const handleAfterCreateWebhook = (data: Webhook) => {
    history.push(
      routes.toARRepositoryWebhookDetails({
        repositoryIdentifier,
        webhookIdentifier: data.identifier
      })
    )
  }
  return (
    <>
      <RbacButton
        permission={{
          resource: {
            resourceType: ResourceType.ARTIFACT_REGISTRY
          },
          permission: PermissionIdentifier.EDIT_ARTIFACT_REGISTRY
        }}
        variation={ButtonVariation.PRIMARY}
        icon="plus"
        iconProps={{ size: 10 }}
        onClick={open}>
        {getString('webhookList.newWebhook')}
      </RbacButton>
      <CreateWebhookModal isOpen={isOpen} onClose={close} onSubmit={handleAfterCreateWebhook} />
    </>
  )
}
