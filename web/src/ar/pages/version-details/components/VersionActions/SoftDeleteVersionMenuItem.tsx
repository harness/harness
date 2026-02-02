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

import { useStrings } from '@ar/frameworks/strings'
import { useParentComponents } from '@ar/hooks'
import { queryClient } from '@ar/utils/queryClient'
import { PermissionIdentifier, ResourceType } from '@ar/common/permissionTypes'

import type { VersionActionProps } from './types'
import useSoftDeleteVersionModal from '../../hooks/useSoftDeleteVersionModal'
import useRestoreDeletedVersionModal from '../../hooks/useRestoreDeletedVersionModal'

export default function SoftDeleteVersionMenuItem(props: VersionActionProps): JSX.Element {
  const { artifactKey, readonly, onClose, versionKey, data } = props
  const { getString } = useStrings()
  const { RbacMenuItem } = useParentComponents()

  const handleAfterDeleteVersion = () => {
    queryClient.invalidateQueries(['ListArtifacts'])
    queryClient.invalidateQueries(['GetArtifactVersionSummary'])
    onClose?.()
  }

  const { triggerSoftDelete } = useSoftDeleteVersionModal({
    versionKey,
    uuid: data.uuid,
    onSuccess: handleAfterDeleteVersion
  })

  const { triggerRestore } = useRestoreDeletedVersionModal({
    onSuccess: handleAfterDeleteVersion,
    uuid: data.uuid
  })

  const isDeleted = !!data.deletedAt

  return (
    <>
      {isDeleted && (
        <RbacMenuItem
          icon="repeat"
          text={getString('versionList.actions.restoreVersion')}
          onClick={triggerRestore}
          disabled={readonly}
          permission={{
            resource: {
              resourceType: ResourceType.ARTIFACT_REGISTRY,
              resourceIdentifier: artifactKey
            },
            permission: PermissionIdentifier.DELETE_ARTIFACT
          }}
        />
      )}
      {!isDeleted && (
        <RbacMenuItem
          icon="archive"
          text={getString('versionList.actions.archiveVersion')}
          onClick={triggerSoftDelete}
          disabled={readonly}
          permission={{
            resource: {
              resourceType: ResourceType.ARTIFACT_REGISTRY,
              resourceIdentifier: artifactKey
            },
            permission: PermissionIdentifier.DELETE_ARTIFACT
          }}
        />
      )}
    </>
  )
}
