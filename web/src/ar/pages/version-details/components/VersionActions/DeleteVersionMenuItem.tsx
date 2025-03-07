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

import type { VersionActionProps } from './types'
import useDeleteVersionModal from '../../hooks/useDeleteVersionModal'

export default function DeleteVersionMenuItem(props: VersionActionProps): JSX.Element {
  const { artifactKey, repoKey, readonly, onClose, versionKey } = props
  const { getString } = useStrings()
  const { RbacMenuItem } = useParentComponents()
  const history = useHistory()
  const routes = useRoutes()

  const handleAfterDeleteRepository = (): void => {
    onClose?.()
    queryClient.invalidateQueries(['GetAllArtifactVersions'])
    history.push(
      routes.toARArtifactDetails({
        repositoryIdentifier: repoKey,
        artifactIdentifier: artifactKey
      })
    )
  }

  const { triggerDelete } = useDeleteVersionModal({
    artifactKey,
    repoKey,
    versionKey,
    onSuccess: handleAfterDeleteRepository
  })

  const handleDeleteService = (): void => {
    triggerDelete()
  }

  return (
    <RbacMenuItem
      icon="code-delete"
      text={getString('versionList.actions.deleteVersion')}
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
