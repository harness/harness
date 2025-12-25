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
import { useHistory } from 'react-router-dom'

import { useParentComponents, useRoutes } from '@ar/hooks'
import { queryClient } from '@ar/utils/queryClient'
import { useStrings } from '@ar/frameworks/strings'
import { PermissionIdentifier, ResourceType } from '@ar/common/permissionTypes'
import useDeleteUpstreamProxyModal from '@ar/pages/upstream-proxy-details/hooks/useDeleteUpstreamProxyModal/useDeleteUpstreamProxyModal'
import type { UpstreamProxyActionProps } from './type'

export default function DeleteUpstreamProxy({ data }: UpstreamProxyActionProps): JSX.Element {
  const { getString } = useStrings()
  const { RbacMenuItem } = useParentComponents()
  const history = useHistory()
  const routes = useRoutes()

  const handleAfterDeleteRepository = (): void => {
    queryClient.invalidateQueries(['GetAllRegistries'])
    history.push(routes.toARRepositories())
  }

  const { triggerDelete } = useDeleteUpstreamProxyModal({
    repoKey: data.identifier,
    onSuccess: handleAfterDeleteRepository
  })

  const handleDeleteService = (): void => {
    triggerDelete()
  }

  return (
    <RbacMenuItem
      icon="code-delete"
      text={getString('actions.delete')}
      onClick={handleDeleteService}
      permission={{
        resource: {
          resourceType: ResourceType.ARTIFACT_REGISTRY,
          resourceIdentifier: data.identifier
        },
        permission: PermissionIdentifier.DELETE_ARTIFACT_REGISTRY
      }}
    />
  )
}
