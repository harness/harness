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

import { queryClient } from '@ar/utils/queryClient'
import { useStrings } from '@ar/frameworks/strings'
import { useParentComponents, useRoutes } from '@ar/hooks'
import { PermissionIdentifier, ResourceType } from '@ar/common/permissionTypes'

import { PageType } from '@ar/common/types'
import type { RepositoryActionsProps } from './types'
import useSoftDeleteRepositoryModal from '../../hooks/useSoftDeleteRepositoryModal/useSoftDeleteRepositoryModal'
import useRestoreDeleteRepositoryModal from '../../hooks/useSoftDeleteRepositoryModal/useRestoreDeleteRepositoryModal'

export default function SoftDeleteRepositoryMenuItem({ data, onClose, pageType }: RepositoryActionsProps): JSX.Element {
  const { isDeleted } = data
  const { getString } = useStrings()
  const { RbacMenuItem } = useParentComponents()
  const history = useHistory()
  const routes = useRoutes()

  const handleAfterDeleteRepository = (): void => {
    queryClient.invalidateQueries(['ListRegistries'])
    queryClient.invalidateQueries(['GetRegistry'])
    onClose?.()
    if (pageType === PageType.Table) {
      history.push(routes.toARRepositories())
    }
  }

  const { triggerDelete } = useSoftDeleteRepositoryModal({
    repoKey: data.identifier,
    uuid: data.uuid,
    onSuccess: handleAfterDeleteRepository
  })

  const { triggerRestore } = useRestoreDeleteRepositoryModal({
    onSuccess: handleAfterDeleteRepository,
    uuid: data.uuid
  })

  return (
    <>
      {isDeleted && (
        <RbacMenuItem
          icon="repeat"
          text={getString('actions.restore')}
          onClick={triggerRestore}
          permission={{
            resource: {
              resourceType: ResourceType.ARTIFACT_REGISTRY,
              resourceIdentifier: data.identifier
            },
            permission: PermissionIdentifier.DELETE_ARTIFACT_REGISTRY
          }}
        />
      )}
      {!isDeleted && (
        <RbacMenuItem
          icon="archive"
          text={getString('actions.softDelete')}
          onClick={triggerDelete}
          permission={{
            resource: {
              resourceType: ResourceType.ARTIFACT_REGISTRY,
              resourceIdentifier: data.identifier
            },
            permission: PermissionIdentifier.DELETE_ARTIFACT_REGISTRY
          }}
        />
      )}
    </>
  )
}
