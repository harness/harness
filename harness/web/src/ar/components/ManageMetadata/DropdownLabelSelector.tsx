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
import { PopoverPosition } from '@blueprintjs/core'
import { Button, ButtonSize, ButtonVariation } from '@harnessio/uicore'
import type { TypesLabel, TypesLabelValue } from '@harnessio/react-har-service-client'

import { useStrings } from '@ar/frameworks/strings'

import PopoverContent from './PopoverContent'

interface DropdownLabelSelectorProps {
  getItems: (searchTerm: string) => Promise<Array<TypesLabel>>
  getValues: (label: TypesLabel, searchTerm: string) => Promise<Array<TypesLabelValue>>
  onSelect: (label: TypesLabel, value?: TypesLabelValue) => void
  disabled?: boolean
}

function DropdownLabelSelector(props: DropdownLabelSelectorProps) {
  const { getItems, onSelect, getValues, disabled } = props
  const [isOpen, setIsOpen] = useState(false)
  const { getString } = useStrings()

  const handleSelect = (label: TypesLabel, value?: TypesLabelValue): void => {
    onSelect(label, value)
    setIsOpen(false)
  }

  return (
    <Button
      variation={ButtonVariation.TERTIARY}
      size={ButtonSize.SMALL}
      icon="plus"
      text={getString('labels.addLabel')}
      tooltipProps={{
        interactionKind: 'click',
        isOpen: disabled ? false : isOpen,
        usePortal: true,
        minimal: true,
        position: PopoverPosition.BOTTOM_RIGHT,
        onInteraction: nxtState => setIsOpen(nxtState)
      }}
      disabled={disabled}
      tooltip={<PopoverContent getLabels={getItems} getValues={getValues} onSelect={handleSelect} />}
    />
  )
}

export default DropdownLabelSelector
