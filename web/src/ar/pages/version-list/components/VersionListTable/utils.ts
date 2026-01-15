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
import { SoftDeleteFilterEnum } from '@ar/constants'
import { VERSION_LIST_TABLE_CELL_CONFIG } from './constants'
import { VersionListColumnEnum } from './types'
import type { CommonVersionListTableProps } from './VersionListTable'

const getColumnConfigBasedOnSoftDeleteFilter = (
  key: VersionListColumnEnum,
  softDeleteFilter?: SoftDeleteFilterEnum
): VersionListColumnEnum => {
  if (softDeleteFilter === SoftDeleteFilterEnum.ONLY) {
    switch (key) {
      case VersionListColumnEnum.LastModified:
        return VersionListColumnEnum.DeletedAt
      default:
        return key
    }
  }
  return key
}

export const getVersionListTableCellConfigs = (
  columnConfigs: CommonVersionListTableProps['columnConfigs'],
  getServerSortProps: (id: string) => void,
  getString: (key: keyof StringsMap) => string,
  softDeleteFilter?: SoftDeleteFilterEnum
): Column[] => {
  return Object.keys(columnConfigs).map(key => {
    const columnConfig =
      VERSION_LIST_TABLE_CELL_CONFIG[
        getColumnConfigBasedOnSoftDeleteFilter(key as VersionListColumnEnum, softDeleteFilter)
      ]
    return {
      ...columnConfig,
      Header: columnConfig.Header ? getString(columnConfig.Header) : '',
      serverSortProps: getServerSortProps(columnConfig.accessor),
      ...columnConfigs[key as VersionListColumnEnum]
    }
  }, {})
}
