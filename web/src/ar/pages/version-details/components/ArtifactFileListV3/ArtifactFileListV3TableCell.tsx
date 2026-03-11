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
import type { FileMetadataV3 } from '@harnessio/react-har-service-client'
import { ButtonSize, ButtonVariation, Layout } from '@harnessio/uicore'
import { compact } from 'lodash-es'
import type { Renderer, Row } from 'react-table'

import { useStrings } from '@ar/frameworks/strings'
import TableCells from '@ar/components/TableCells/TableCells'
import { PermissionIdentifier, ResourceType } from '@ar/common/permissionTypes'
import { useAppStore } from '@ar/hooks/useAppStore'
import { Parent } from '@ar/common/types'

import css from './ArtifactFileListV3.module.scss'

type CellProps = { value: unknown; row: Row<FileMetadataV3>; column?: { meta?: { repositoryIdentifier?: string } } }

export const FilePathCellV3: Renderer<CellProps> = ({ value }) => {
  return <TableCells.TextCell value={String(value ?? '')} />
}

export const FileSizeCellV3: Renderer<CellProps> = ({ value }) => {
  return <TableCells.SizeCell value={String(value ?? '')} />
}

export const FileCreatedAtCellV3: Renderer<CellProps> = ({ row }) => {
  const { original } = row
  const { createdAt } = original
  return <TableCells.LastModifiedCell value={createdAt || 0} />
}

const HASH_LABELS: Record<string, string> = {
  md5: 'MD5',
  sha1: 'SHA1',
  sha256: 'SHA256',
  sha512: 'SHA512'
}

const CHECKSUM_KEYS = ['md5', 'sha1', 'sha256', 'sha512'] as const

export const FileChecksumsV3Cell: Renderer<CellProps> = ({ row }) => {
  const { getString } = useStrings()
  const entries = compact(
    CHECKSUM_KEYS.map(key => {
      const val = row.original[key]
      return val ? ([key, val] as [string, string]) : null
    })
  )

  if (entries.length === 0) return <TableCells.TextCell value={getString('na')} />

  return (
    <Layout.Horizontal className={css.checksumContainer}>
      {entries.map(([key, val]) => (
        <TableCells.CopyTextCell
          key={key}
          className={css.copyChecksumBtn}
          minimal={false}
          value={val}
          variation={ButtonVariation.TERTIARY}
          size={ButtonSize.SMALL}>
          {getString('copy')} {HASH_LABELS[key] ?? key}
        </TableCells.CopyTextCell>
      ))}
    </Layout.Horizontal>
  )
}

export const FileDownloadUrlCellV3: Renderer<CellProps> = ({ row, column }) => {
  const { getString } = useStrings()
  const { parent } = useAppStore()
  const repositoryIdentifier = column?.meta?.repositoryIdentifier ?? ''
  const downloadUrl = row.original.downloadUrl ?? ''

  if (!downloadUrl) return <TableCells.TextCell value={getString('na')} />

  return (
    <Layout.Horizontal spacing="medium">
      <TableCells.CopyTextCell value={downloadUrl}>{getString('copy')}</TableCells.CopyTextCell>
      {parent === Parent.Enterprise && (
        <TableCells.DownloadCell
          href={downloadUrl}
          target="_blank"
          permission={{
            permission: PermissionIdentifier.DOWNLOAD_ARTIFACT,
            resource: {
              resourceType: ResourceType.ARTIFACT_REGISTRY,
              resourceIdentifier: repositoryIdentifier ?? ''
            }
          }}>
          {getString('download')}
        </TableCells.DownloadCell>
      )}
    </Layout.Horizontal>
  )
}
