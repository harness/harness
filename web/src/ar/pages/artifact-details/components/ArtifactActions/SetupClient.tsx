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
import { defaultTo } from 'lodash-es'

import { useParentComponents } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'
import type { RepositoryPackageType } from '@ar/common/types'
import { PermissionIdentifier, ResourceType } from '@ar/common/permissionTypes'
import { useSetupClientModal } from '@ar/pages/repository-details/hooks/useSetupClientModal/useSetupClientModal'

import type { ArtifactActionProps } from './types'

export default function SetupClientMenuItem({ data, repoKey }: ArtifactActionProps): JSX.Element {
  const { getString } = useStrings()
  const { RbacMenuItem } = useParentComponents()
  const artifactKey = data.imageName || ''

  const [showSetupClientModal] = useSetupClientModal({
    repoKey,
    packageType: data.packageType as RepositoryPackageType,
    artifactKey
  })
  return (
    <>
      <RbacMenuItem
        icon="setup-client"
        text={getString('actions.setupClient')}
        onClick={showSetupClientModal}
        permission={{
          resource: {
            resourceType: ResourceType.ARTIFACT_REGISTRY,
            resourceIdentifier: defaultTo(repoKey, '')
          },
          permission: PermissionIdentifier.VIEW_ARTIFACT_REGISTRY
        }}
      />
    </>
  )
}
