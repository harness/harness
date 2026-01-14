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

import type { LabelActionProps } from './type'
import EditLabelActionItem from './EditLabelActionItem'
import DeleteLabelActionItem from './DeleteLabelActionItem'

function LabelActions(props: LabelActionProps) {
  const [open, setOpen] = useState(false)

  const handleClose = (reload: boolean): void => {
    setOpen(false)
    props.onClose?.(reload)
  }

  return (
    <ActionButton isOpen={open} setOpen={setOpen}>
      <EditLabelActionItem {...props} onClose={handleClose} />
      <DeleteLabelActionItem {...props} onClose={handleClose} />
    </ActionButton>
  )
}

export default LabelActions
