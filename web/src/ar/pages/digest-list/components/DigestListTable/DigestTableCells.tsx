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
import { Link } from 'react-router-dom'
import type { Cell, CellValue, ColumnInstance, Renderer, Row, TableInstance } from 'react-table'

import { Text } from '@harnessio/uicore'
import { Color } from '@harnessio/design-system'
import type { DockerManifestDetails } from '@harnessio/react-har-service-client'

import { useStrings } from '@ar/frameworks/strings'
import { useDecodedParams, useRoutes } from '@ar/hooks'
import { getShortDigest } from '@ar/pages/digest-list/utils'
import TableCells from '@ar/components/TableCells/TableCells'
import type { ArtifactDetailsPathParams } from '@ar/routes/types'
import { VersionDetailsTab } from '@ar/pages/version-details/components/VersionDetailsTabs/constants'

type CellTypeWithActions<D extends Record<string, any>, V = any> = TableInstance<D> & {
  column: ColumnInstance<D>
  row: Row<D>
  cell: Cell<D, V>
  value: CellValue<V>
}

type CellType = Renderer<CellTypeWithActions<DockerManifestDetails>>

type DigestNameColumnProps = {
  version: string
}

export const DigestNameCell: Renderer<{
  row: Row<DockerManifestDetails>
  column: ColumnInstance<DockerManifestDetails> & DigestNameColumnProps
}> = ({ row, column }) => {
  const { original } = row
  const { version } = column
  const value = original.digest
  const pathParams = useDecodedParams<ArtifactDetailsPathParams>()
  const routes = useRoutes()

  const linkTo = routes.toARVersionDetailsTab({
    repositoryIdentifier: pathParams.repositoryIdentifier,
    artifactIdentifier: pathParams.artifactIdentifier,
    versionIdentifier: version,
    versionTab: VersionDetailsTab.OVERVIEW
  })
  return <TableCells.LinkCell label={getShortDigest(value)} linkTo={`${linkTo}?digest=${value}`} />
}

export const OsArchCell: CellType = ({ value }) => {
  return <TableCells.TextCell value={value} />
}

export const SizeCell: CellType = ({ value }) => {
  return <TableCells.SizeCell value={value} />
}

export const UploadedByCell: CellType = ({ value }) => {
  return <TableCells.LastModifiedCell value={value} />
}

export const DownloadsCell: CellType = ({ value }) => {
  return <TableCells.CountCell value={value} icon="download-box" iconProps={{ size: 12 }} />
}

export const ScanStatusCell: Renderer<{
  row: Row<DockerManifestDetails>
  column: ColumnInstance<DockerManifestDetails> & DigestNameColumnProps
}> = ({ row, column }) => {
  const { original } = row
  const { version } = column
  const router = useRoutes()
  const { stoExecutionId, stoPipelineId, digest } = original
  const pathParams = useDecodedParams<ArtifactDetailsPathParams>()
  const { getString } = useStrings()
  if (!stoExecutionId || !stoPipelineId)
    return <TableCells.TextCell value={getString('artifactList.table.actions.VulnerabilityStatus.nonScanned')} />

  const linkTo = router.toARVersionDetailsTab({
    repositoryIdentifier: pathParams.repositoryIdentifier,
    artifactIdentifier: pathParams.artifactIdentifier,
    versionIdentifier: version,
    versionTab: VersionDetailsTab.SECURITY_TESTS,
    pipelineIdentifier: stoPipelineId,
    executionIdentifier: stoExecutionId
  })
  return (
    <Link to={`${linkTo}?digest=${digest}`}>
      <Text
        color={Color.PRIMARY_7}
        rightIcon="main-share"
        rightIconProps={{
          size: 12
        }}
        lineClamp={1}>
        {getString('artifactList.table.actions.VulnerabilityStatus.scanned')}
      </Text>
    </Link>
  )
}

export const DigestActionsCell: CellType = () => {
  return <></>
}
