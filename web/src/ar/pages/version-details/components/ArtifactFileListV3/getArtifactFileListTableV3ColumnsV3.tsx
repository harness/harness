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

import type { FileMetadataV3 } from '@harnessio/react-har-service-client'
import type { Column } from 'react-table'
import type { StringKeys } from '@ar/frameworks/strings'

import {
  FilePathCellV3,
  FileSizeCellV3,
  FileCreatedAtCellV3,
  FileChecksumsV3Cell,
  FileDownloadUrlCellV3
} from './ArtifactFileListV3TableCell'

export interface FileListSortByV3 {
  sort: 'name' | 'size' | 'created_at'
}

type FileListColumnV3 = Column<FileMetadataV3> & {
  serverSortProps?: {
    enableServerSort: boolean
    isServerSorted: boolean
    isServerSortedDesc: boolean
    getSortedColumn: (opts: FileListSortByV3) => void
  }
  disableSortBy?: boolean
  meta?: { repositoryIdentifier?: string }
}

export function getFileListTableColumnsV3(
  getString: (key: StringKeys) => string,
  sortBy: string[],
  setSortBy: (sortBy: string[]) => void,
  repositoryIdentifier: string
): Column<FileMetadataV3>[] {
  const [currentSort, currentOrder] = sortBy
  const getServerSortProps = (id: string) => ({
    enableServerSort: true,
    isServerSorted: currentSort === id,
    isServerSortedDesc: currentOrder === 'DESC',
    getSortedColumn: ({ sort }: FileListSortByV3) => {
      setSortBy([sort, currentOrder === 'DESC' ? 'ASC' : 'DESC'])
    }
  })

  const columns: FileListColumnV3[] = [
    {
      Header: getString('versionDetails.artifactFiles.table.columns.name'),
      accessor: 'path',
      Cell: FilePathCellV3,
      serverSortProps: getServerSortProps('path')
    },
    {
      Header: getString('versionDetails.artifactFiles.table.columns.size'),
      accessor: 'size',
      Cell: FileSizeCellV3,
      serverSortProps: getServerSortProps('size')
    },
    {
      Header: getString('versionDetails.artifactFiles.table.columns.checksum'),
      accessor: 'sha256',
      Cell: FileChecksumsV3Cell,
      disableSortBy: true
    },
    {
      Header: getString('versionDetails.artifactFiles.table.columns.downloadCommand'),
      accessor: 'downloadUrl',
      Cell: FileDownloadUrlCellV3,
      meta: { repositoryIdentifier },
      disableSortBy: true
    },
    {
      id: 'created_at',
      Header: getString('versionDetails.artifactFiles.table.columns.created'),
      accessor: 'createdAt',
      Cell: FileCreatedAtCellV3,
      serverSortProps: getServerSortProps('created_at')
    }
  ]
  return columns
}
