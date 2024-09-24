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
import { defaultTo } from 'lodash-es'

import { useStrings } from '@ar/frameworks/strings'
import { useParentComponents, useRoutes } from '@ar/hooks'
import { PermissionIdentifier, ResourceType } from '@ar/common/permissionTypes'

import { queryClient } from '@ar/utils/queryClient'
import useDeleteRepositoryModal from '@ar/pages/repository-details/hooks/useDeleteRepositoryModal/useDeleteRepositoryModal'
import type { RepositoryActionsProps } from './types'

export default function DeleteRepositoryMenuItem({ data }: RepositoryActionsProps): JSX.Element {
  const { getString } = useStrings()
  const { RbacMenuItem } = useParentComponents()
  const history = useHistory()
  const routes = useRoutes()

  const handleAfterDeleteRepository = (): void => {
    queryClient.invalidateQueries(['GetAllRegistries'])
    history.push(routes.toARRepositories())
  }

  const { triggerDelete } = useDeleteRepositoryModal({
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
          resourceIdentifier: defaultTo(data?.identifier, '')
        },
        permission: PermissionIdentifier.DELETE_ARTIFACT_REGISTRY
      }}
    />
  )
}
