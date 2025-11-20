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
import classNames from 'classnames'
import { Button, ButtonVariation, Layout, Text } from '@harnessio/uicore'
import { PopoverInteractionKind, PopoverPosition } from '@blueprintjs/core'

import { useStrings } from '@ar/frameworks/strings'

import PopoverContent from './PopoverContent'
import type { PropertySpec } from '../PropertiesForm/types'

import css from './MetadataFilterSelector.module.scss'

interface MetadataFilterSelectorProps {
  value: PropertySpec[]
  onSubmit: (data: PropertySpec[]) => void
}

function MetadataFilterSelector({ value, onSubmit }: MetadataFilterSelectorProps) {
  const [isOpen, setOpen] = React.useState(false)
  const { getString } = useStrings()

  const handleSubmit = (data: PropertySpec[]) => {
    setOpen(false)
    onSubmit(data)
  }

  const handleClose = () => {
    setOpen(false)
  }

  const isActive = value.length > 0
  return (
    <Button
      className={classNames(css.actionBtn, {
        [css.active]: isActive
      })}
      rightIcon="main-chevron-down"
      text={
        <Layout.Horizontal spacing="small">
          <Text>{getString('metadata')}</Text>
          {isActive && <Text className={css.counter}>{String(value.length).padStart(2, '0')}</Text>}
        </Layout.Horizontal>
      }
      variation={ButtonVariation.TERTIARY}
      tooltip={<PopoverContent value={value} onSubmit={handleSubmit} onClose={handleClose} />}
      tooltipProps={{
        interactionKind: PopoverInteractionKind.CLICK,
        position: PopoverPosition.BOTTOM_LEFT,
        minimal: true,
        onInteraction: nextOpenState => {
          setOpen(nextOpenState)
        },
        isOpen: isOpen
      }}
    />
  )
}

export default MetadataFilterSelector
