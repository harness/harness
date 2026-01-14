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

import { useParentComponents } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'
import { queryClient } from '@ar/utils/queryClient'
import { ResourceType } from '@ar/common/permissionTypes'
import { PermissionIdentifier } from '@ar/common/permissionTypes'

import type { VersionActionProps } from './types'
import useRemoveQuarantineVersion from '../../hooks/useRemoveQuarantineVersion'

function RemoveQurantineMenuItem(props: VersionActionProps) {
  const { artifactKey, readonly, onClose, versionKey, repoKey } = props
  const { getString } = useStrings()
  const { RbacMenuItem } = useParentComponents()

  const handleAfterQuarantineVersion = () => {
    queryClient.invalidateQueries(['GetAllArtifactVersions'])
    queryClient.invalidateQueries(['GetAllHarnessArtifacts'])
    queryClient.invalidateQueries(['GetArtifactVersionSummary'])
    onClose?.()
  }

  const { triggerRemoveQuarantine } = useRemoveQuarantineVersion({
    artifactKey,
    repoKey,
    versionKey,
    onSuccess: handleAfterQuarantineVersion
  })

  const handleRemoveFromQuarantine = (): void => {
    triggerRemoveQuarantine()
  }

  return (
    <RbacMenuItem
      icon="error-tracking"
      text={getString('versionList.actions.removeQuarantine')}
      onClick={handleRemoveFromQuarantine}
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

export default RemoveQurantineMenuItem
