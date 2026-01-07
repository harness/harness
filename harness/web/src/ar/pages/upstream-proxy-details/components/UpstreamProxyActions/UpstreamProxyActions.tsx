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
import { Menu, Position } from '@blueprintjs/core'
import { Button, ButtonVariation } from '@harnessio/uicore'

import DeleteUpstreamProxy from './DeleteUpstreamProxy'
import type { UpstreamProxyActionProps } from './type'

import css from './UpstreamProxyActions.module.scss'

export default function UpstreamProxyActions({ data, readonly, pageType }: UpstreamProxyActionProps): JSX.Element {
  const [menuOpen, setMenuOpen] = useState(false)
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
          <DeleteUpstreamProxy data={data} readonly={readonly} pageType={pageType} />
        </Menu>
      }
      tooltipProps={{
        interactionKind: 'click',
        onInteraction: nextOpenState => {
          setMenuOpen(nextOpenState)
        },
        isOpen: menuOpen,
        position: Position.BOTTOM
      }}
      onClick={e => {
        e.stopPropagation()
        setMenuOpen(true)
      }}
    />
  )
}
