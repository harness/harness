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
import { PermissionIdentifier, ResourceType } from '@ar/common/permissionTypes'

import { queryClient } from '@ar/utils/queryClient'
import type { ExemptionActionsProps } from './types'
import useDeleteExemptionModal from '../../hooks/useDeleteExemption'

export default function DeleteExemptionMenuItem({ data, onClose }: ExemptionActionsProps): JSX.Element {
  const { getString } = useStrings()
  const { RbacMenuItem } = useParentComponents()

  const handleAfterDeleteExemption = (): void => {
    queryClient.invalidateQueries(['ListFirewallExceptionsV3'])
    onClose?.()
  }

  const { triggerDelete } = useDeleteExemptionModal({
    exemptionId: data.exceptionId,
    packageName: data.packageName,
    onSuccess: handleAfterDeleteExemption
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
          resourceIdentifier: data.registryName
        },
        permission: PermissionIdentifier.DELETE_ARTIFACT_REGISTRY
      }}
    />
  )
}
