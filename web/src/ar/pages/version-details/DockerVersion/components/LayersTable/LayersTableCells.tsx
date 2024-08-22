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
import type { Cell, CellValue, ColumnInstance, Renderer, Row, TableInstance } from 'react-table'
import { FontVariation } from '@harnessio/design-system'
import { CopyToClipboard, Text } from '@harnessio/uicore'

import css from './LayersTable.module.scss'

type CellTypeWithActions<D extends Record<string, any>, V = any> = TableInstance<D> & {
  column: ColumnInstance<D>
  row: Row<D>
  cell: Cell<D, V>
  value: CellValue<V>
}

type CellType = Renderer<CellTypeWithActions<any>>

export const LayerIndexCell: CellType = ({ value }) => {
  return (
    <Text className={css.codeText} font={{ variation: FontVariation.BODY }}>
      {value}
    </Text>
  )
}

export const LayerCodeCell: CellType = ({ value }) => {
  return (
    <Text className={classNames(css.codeText, css.codeCell)} font={{ variation: FontVariation.BODY }}>
      {value}
    </Text>
  )
}

export const LayerSizeCell: CellType = ({ value }) => {
  return (
    <Text className={classNames(css.codeText, css.sizeCell)} font={{ variation: FontVariation.BODY }}>
      {value}
    </Text>
  )
}

export const LayerActionCell: CellType = ({ row }) => {
  const { original } = row
  const { code } = original
  return <CopyToClipboard content={code} showFeedback iconSize={18} />
}
