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
import { PermissionIdentifier, ResourceType } from '@ar/common/permissionTypes'
import useCloneVersionModal from '../../hooks/useCloneVersionModal'

import type { VersionActionProps } from './types'

export type CloneVersionMenuItemProps = VersionActionProps

export default function CloneVersionMenuItem(props: CloneVersionMenuItemProps): JSX.Element {
  const { artifactKey, repoKey, versionKey, repoType, data, readonly, onClose } = props
  const { getString } = useStrings()
  const { RbacMenuItem } = useParentComponents()

  const handleAfterClone = () => {
    onClose?.()
  }

  const { openCloneVersionModal } = useCloneVersionModal({
    artifactKey,
    repoKey,
    versionKey,
    versionUuid: data?.uuid,
    registryType: repoType,
    packageType: data?.packageType,
    onSuccess: handleAfterClone
  })

  return (
    <RbacMenuItem
      icon="duplicate"
      text={getString('versionList.actions.copyVersion')}
      onClick={openCloneVersionModal}
      disabled={readonly}
      permission={{
        resource: {
          resourceType: ResourceType.ARTIFACT_REGISTRY,
          resourceIdentifier: repoKey
        },
        permission: PermissionIdentifier.UPLOAD_ARTIFACT
      }}
    />
  )
}
