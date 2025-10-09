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

import type { Column } from 'react-table'
import type { StringsMap } from '@ar/strings/types'
import { VERSION_LIST_TABLE_CELL_CONFIG } from './constants'
import type { VersionListColumnEnum } from './types'
import type { CommonVersionListTableProps } from './VersionListTable'

export const getVersionListTableCellConfigs = (
  columnConfigs: CommonVersionListTableProps['columnConfigs'],
  getServerSortProps: (id: string) => void,
  getString: (key: keyof StringsMap) => string
): Column[] => {
  return Object.keys(columnConfigs).map(key => {
    const columnConfig = VERSION_LIST_TABLE_CELL_CONFIG[key as VersionListColumnEnum]
    return {
      ...columnConfig,
      Header: columnConfig.Header ? getString(columnConfig.Header) : '',
      serverSortProps: getServerSortProps(columnConfig.accessor),
      ...columnConfigs[key as VersionListColumnEnum]
    }
  }, {})
}
