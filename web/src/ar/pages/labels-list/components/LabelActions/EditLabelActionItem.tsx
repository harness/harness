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

import type { LabelActionProps } from './type'

export default function EditLabelActionItem(props: LabelActionProps): JSX.Element {
  const { data, readonly } = props
  const { getString } = useStrings()
  const { RbacMenuItem } = useParentComponents()

  const handleEditService = (): void => {
    // TODO: implement edit label flow
  }

  return (
    <RbacMenuItem
      icon="code-edit"
      text={getString('labelsList.table.actions.edit')}
      onClick={handleEditService}
      disabled={readonly}
      permission={{
        resource: {
          resourceType: ResourceType.ARTIFACT_REGISTRY,
          resourceIdentifier: data.key
        },
        permission: PermissionIdentifier.EDIT_ARTIFACT_REGISTRY
      }}
    />
  )
}
