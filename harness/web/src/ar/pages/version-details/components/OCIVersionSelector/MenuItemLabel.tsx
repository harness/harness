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
import { Position } from '@blueprintjs/core'
import { Text } from '@harnessio/uicore'

import css from './OCIVersionSelector.module.scss'

export default function MenuItemLabel(props: PropsWithChildren<unknown>) {
  return (
    <Text lineClamp={1} tooltipProps={{ position: Position.RIGHT }} className={css.menuItemText}>
      {props.children}
    </Text>
  )
}
