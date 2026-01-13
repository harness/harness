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

import { useStrings } from '@ar/frameworks/strings'
import { queryClient } from '@ar/utils/queryClient'
import { useParentComponents, useRoutes } from '@ar/hooks'
import { PermissionIdentifier, ResourceType } from '@ar/common/permissionTypes'

import type { ArtifactActionProps } from './types'
import useDeleteArtifactModal from '../../hooks/useDeleteArtifactModal/useDeleteArtifactModal'

export default function DeleteArtifactMenuItem(props: ArtifactActionProps): JSX.Element {
  const { artifactKey, repoKey, readonly, onClose, data } = props
  const { getString } = useStrings()
  const { RbacMenuItem } = useParentComponents()
  const history = useHistory()
  const routes = useRoutes()

  const handleAfterDeleteRepository = (): void => {
    onClose?.()
    queryClient.invalidateQueries(['GetAllArtifactsByRegistry'])
    queryClient.invalidateQueries(['ListPackages'])
    history.push(
      routes.toARRepositoryDetails({
        repositoryIdentifier: repoKey
      })
    )
  }

  const { triggerDelete } = useDeleteArtifactModal({
    artifactKey,
    repoKey,
    onSuccess: handleAfterDeleteRepository,
    artifactType: data?.artifactType
  })

  const handleDeleteService = (): void => {
    triggerDelete()
  }

  return (
    <RbacMenuItem
      icon="code-delete"
      text={getString('artifactList.table.actions.deleteArtifact')}
      onClick={handleDeleteService}
      disabled={readonly}
      permission={{
        resource: {
          resourceType: ResourceType.ARTIFACT_REGISTRY,
          resourceIdentifier: artifactKey
        },
        permission: PermissionIdentifier.DELETE_ARTIFACT
      }}
    />
  )
}
