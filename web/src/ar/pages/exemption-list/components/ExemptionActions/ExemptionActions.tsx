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

import ActionButton from '@ar/components/ActionButton/ActionButton'

import type { ExemptionActionsProps } from './types'
import EditExemptionMenuItem from './EditExemptionMenuItem'
import DeleteExemptionMenuItem from './DeleteExemptionMenuItem'

export default function ExemptionActions(props: ExemptionActionsProps): JSX.Element {
  const [open, setOpen] = useState(false)
  return (
    <ActionButton isOpen={open} setOpen={setOpen}>
      {!props.data.expirationAt && (
        <EditExemptionMenuItem
          {...props}
          onClose={() => {
            setOpen(false)
            props.onClose?.()
          }}
        />
      )}
      <DeleteExemptionMenuItem
        {...props}
        onClose={() => {
          setOpen(false)
          props.onClose?.()
        }}
      />
    </ActionButton>
  )
}
