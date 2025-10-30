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
import type { Column } from 'react-table'
import { TableV2 } from '@harnessio/uicore'
import type { DockerManifestDetails } from '@harnessio/react-har-service-client'

import { useStrings } from '@ar/frameworks/strings/String'
import {
  DigestActionsCell,
  DigestNameCell,
  type DigestNameColumnProps,
  DigestVulnerabilityCell,
  DownloadsCell,
  OsArchCell,
  SizeCell,
  UploadedByCell
} from './DigestTableCells'
import css from './DigestListTable.module.scss'

interface DigestListTableProps extends DigestNameColumnProps {
  data: DockerManifestDetails[]
}

export default function DigestListTable(props: DigestListTableProps): JSX.Element {
  const { data, repositoryIdentifier, artifactIdentifier, versionIdentifier } = props

  const { getString } = useStrings()

  const columns: Column<DockerManifestDetails>[] = React.useMemo(() => {
    return [
      {
        Header: getString('digestList.table.columns.digest'),
        accessor: 'digest',
        Cell: DigestNameCell,
        repositoryIdentifier,
        artifactIdentifier,
        versionIdentifier
      },
      {
        Header: getString('digestList.table.columns.osArch'),
        accessor: 'osArch',
        Cell: OsArchCell
      },
      {
        Header: getString('digestList.table.columns.size'),
        accessor: 'size',
        Cell: SizeCell
      },
      {
        Header: getString('digestList.table.columns.uploadedBy'),
        accessor: 'createdAt',
        Cell: UploadedByCell
      },
      {
        Header: getString('digestList.table.columns.downloads'),
        accessor: 'downloadsCount',
        Cell: DownloadsCell
      },
      {
        Header: getString('digestList.table.columns.scanStatus'),
        accessor: 'scanStatus',
        Cell: DigestVulnerabilityCell
      },
      {
        Header: '',
        accessor: 'menu',
        Cell: DigestActionsCell,
        disableSortBy: true,
        repositoryIdentifier,
        artifactIdentifier,
        versionIdentifier
      }
    ].filter(Boolean) as unknown as Column<DockerManifestDetails>[]
  }, [getString, repositoryIdentifier, artifactIdentifier, versionIdentifier])
  return (
    <TableV2<DockerManifestDetails>
      minimal
      className={classNames(css.table)}
      columns={columns}
      data={data}
      sortable={false}
    />
  )
}
