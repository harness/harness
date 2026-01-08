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

import React, { useContext } from 'react'
import { ButtonSize, ButtonVariation, Layout } from '@harnessio/uicore'
import type { FileDetail } from '@harnessio/react-har-service-client'
import type { Cell, CellValue, ColumnInstance, Renderer, Row, TableInstance } from 'react-table'

import { Parent } from '@ar/common/types'
import { useStrings } from '@ar/frameworks/strings'
import { useAppStore, useGetDownloadFileURL } from '@ar/hooks'
import TableCells from '@ar/components/TableCells/TableCells'
import { PermissionIdentifier, ResourceType } from '@ar/common/permissionTypes'

import { VersionFilesContext } from '../../context/VersionFilesProvider'

import css from './ArtifactFileListTable.module.scss'

type CellTypeWithActions<D extends Record<string, any>, V = any> = TableInstance<D> & {
  column: ColumnInstance<D>
  row: Row<D>
  cell: Cell<D, V>
  value: CellValue<V>
}

type CellType = Renderer<CellTypeWithActions<FileDetail>>

export const FileNameCell: CellType = ({ value }) => {
  return <TableCells.TextCell value={value} />
}

export const FileSizeCell: CellType = ({ value }) => {
  return <TableCells.SizeCell value={value} />
}

export const FileChecksumListCell: CellType = ({ row }) => {
  const { original } = row
  const { checksums: value } = original
  const { getString } = useStrings()
  if (Array.isArray(value) && value.length) {
    return (
      <Layout.Horizontal className={css.checksumContainer}>
        {value.map(each => {
          const [label, val] = each.split(': ')
          return (
            <TableCells.CopyTextCell
              key={val}
              className={css.copyChecksumBtn}
              minimal={false}
              value={val}
              variation={ButtonVariation.TERTIARY}
              size={ButtonSize.SMALL}>
              {`${getString('copy')} ${label}`}
            </TableCells.CopyTextCell>
          )
        })}
      </Layout.Horizontal>
    )
  }
  return <TableCells.TextCell value={getString('na')} />
}

export const FileCreatedCell: CellType = ({ value }) => {
  return <TableCells.LastModifiedCell value={value} />
}

export const FileDownloadCommandCell: CellType = ({ value, row }) => {
  const { original } = row
  const { getString } = useStrings()
  const { downloadCommand, name } = original
  const { parent } = useAppStore()
  const { repositoryIdentifier } = useContext(VersionFilesContext)

  const fileURL = useGetDownloadFileURL({
    repositoryIdentifier,
    fileName: name,
    path: value
  })

  return (
    <Layout.Horizontal spacing="medium">
      <TableCells.CopyTextCell value={downloadCommand}>{getString('copy')}</TableCells.CopyTextCell>
      {parent === Parent.Enterprise && (
        <TableCells.DownloadCell
          href={fileURL}
          target="_blank"
          permission={{
            permission: PermissionIdentifier.DOWNLOAD_ARTIFACT,
            resource: {
              resourceType: ResourceType.ARTIFACT_REGISTRY,
              resourceIdentifier: repositoryIdentifier
            }
          }}>
          {getString('download')}
        </TableCells.DownloadCell>
      )}
    </Layout.Horizontal>
  )
}
