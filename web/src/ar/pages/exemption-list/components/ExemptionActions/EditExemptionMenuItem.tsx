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
import { queryClient } from '@ar/utils/queryClient'
import { useStrings } from '@ar/frameworks/strings'
import { PermissionIdentifier, ResourceType } from '@ar/common/permissionTypes'
import EditExemptionFormModal from '@ar/pages/violations-list/components/EditExemptionForm/EditExemptionFormModal'

import type { ExemptionActionsProps } from './types'

function EditExemptionMenuItem(props: ExemptionActionsProps) {
  const [open, setOpen] = useState(false)
  const { RbacMenuItem } = useParentComponents()
  const { getString } = useStrings()
  return (
    <>
      <RbacMenuItem
        icon="edit"
        text={getString('actions.editExemption')}
        onClick={() => {
          setOpen(true)
        }}
        permission={{
          resource: {
            resourceType: ResourceType.ARTIFACT_FIREWALL_EXCEPTIONS,
            resourceIdentifier: props.data.exceptionId
          },
          permission: PermissionIdentifier.ARTIFACT_FIREWALL_EXCEPTIONS_CREATE
        }}
      />
      {open && (
        <EditExemptionFormModal
          data={props.data}
          onClose={() => {
            setOpen(false)
            props.onClose?.()
            queryClient.invalidateQueries(['ListFirewallExceptionsV3'])
          }}
        />
      )}
    </>
  )
}

export default EditExemptionMenuItem
