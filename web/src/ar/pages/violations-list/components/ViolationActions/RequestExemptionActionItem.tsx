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

import React, { useState } from 'react'

import { useParentComponents } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'
import { PermissionIdentifier, ResourceType } from '@ar/common/permissionTypes'
import type { ViolationActionsProps } from './types'
import CreateExemptionFormModal from '../CreateExemptionForm/CreateExemptionFormModal'

function RequestExemptionActionItem(props: ViolationActionsProps) {
  const [open, setOpen] = useState(false)
  const { RbacMenuItem } = useParentComponents()
  const { getString } = useStrings()
  return (
    <>
      <RbacMenuItem
        icon="exclude-row"
        text={getString('actions.requestExemption')}
        onClick={() => {
          setOpen(true)
        }}
        permission={{
          resource: {
            resourceType: ResourceType.ARTIFACT_REGISTRY,
            resourceIdentifier: props.data.id
          },
          permission: PermissionIdentifier.DOWNLOAD_ARTIFACT
        }}
      />
      {open && (
        <CreateExemptionFormModal
          data={props.data}
          onClose={() => {
            setOpen(false)
            props.onClose?.()
          }}
        />
      )}
    </>
  )
}

export default RequestExemptionActionItem
