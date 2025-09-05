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

import {
  DigestNameCell,
  OCITagsCell,
  PullCommandCell,
  VersionActionsCell,
  VersionDownloadsCell,
  VersionFileCountCell,
  VersionNameCell,
  VersionPublishedAtCell,
  VersionSizeCell
} from './VersionListCell'
import { type IVersionListTableColumnConfigType, VersionListColumnEnum } from './types'

export const VERSION_LIST_TABLE_CELL_CONFIG: Record<VersionListColumnEnum, IVersionListTableColumnConfigType> = {
  [VersionListColumnEnum.Name]: {
    Header: 'versionList.table.columns.version',
    accessor: 'name',
    Cell: VersionNameCell
  },
  [VersionListColumnEnum.Digest]: {
    Header: 'versionList.table.columns.version',
    accessor: 'name',
    Cell: DigestNameCell,
    disableSortBy: true
  },
  [VersionListColumnEnum.Tags]: {
    Header: 'versionList.table.columns.tags',
    accessor: 'metadata',
    Cell: OCITagsCell,
    disableSortBy: true
  },
  [VersionListColumnEnum.Size]: {
    Header: 'versionList.table.columns.size',
    accessor: 'size',
    Cell: VersionSizeCell
  },
  [VersionListColumnEnum.FileCount]: {
    Header: 'versionList.table.columns.fileCount',
    accessor: 'fileCount',
    Cell: VersionFileCountCell
  },
  [VersionListColumnEnum.DownloadCount]: {
    Header: 'versionList.table.columns.downloads',
    accessor: 'downloadsCount',
    Cell: VersionDownloadsCell
  },
  [VersionListColumnEnum.LastModified]: {
    Header: 'versionList.table.columns.publishedByAt',
    accessor: 'lastModified',
    Cell: VersionPublishedAtCell
  },
  [VersionListColumnEnum.PullCommand]: {
    Header: 'versionList.table.columns.pullCommand',
    accessor: 'pullCommand',
    Cell: PullCommandCell,
    disableSortBy: true
  },
  [VersionListColumnEnum.Actions]: {
    accessor: 'actions',
    Cell: VersionActionsCell,
    disableSortBy: true
  }
}
