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

import React, { PropsWithChildren } from 'react'
import { Button, ButtonVariation } from '@harnessio/uicore'
import { Menu, Position } from '@blueprintjs/core'

import css from './ActionButton.module.scss'

interface ActionButtonProps {
  isOpen: boolean
  setOpen: (val: boolean) => void
}

export default function ActionButton({ children, isOpen, setOpen }: PropsWithChildren<ActionButtonProps>): JSX.Element {
  return (
    <Button
      variation={ButtonVariation.ICON}
      icon="Options"
      tooltip={
        <Menu
          className={css.optionsMenu}
          onClick={e => {
            e.stopPropagation()
          }}>
          {children}
        </Menu>
      }
      tooltipProps={{
        interactionKind: 'click',
        onInteraction: nextOpenState => {
          setOpen(nextOpenState)
        },
        isOpen: isOpen,
        position: Position.BOTTOM
      }}
      onClick={e => {
        e.stopPropagation()
        setOpen(!isOpen)
      }}
    />
  )
}
