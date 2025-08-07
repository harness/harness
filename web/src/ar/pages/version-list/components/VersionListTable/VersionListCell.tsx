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
import { defaultTo } from 'lodash-es'
import { Link } from 'react-router-dom'
import type { Cell, CellValue, ColumnInstance, Renderer, Row, TableInstance, UseExpandedRowProps } from 'react-table'
import { Color, FontVariation } from '@harnessio/design-system'
import { Icon } from '@harnessio/icons'
import { Layout, Text } from '@harnessio/uicore'
import type { ArtifactVersionMetadata } from '@harnessio/react-har-service-client'

import { useStrings } from '@ar/frameworks/strings'
import { useDecodedParams, useRoutes } from '@ar/hooks'
import TableCells from '@ar/components/TableCells/TableCells'
import type { ArtifactDetailsPathParams } from '@ar/routes/types'
import { PageType, type RepositoryPackageType } from '@ar/common/types'
import { LocalArtifactType } from '@ar/pages/repository-details/constants'
import VersionActionsWidget from '@ar/frameworks/Version/VersionActionsWidget'
import { VersionDetailsTab } from '@ar/pages/version-details/components/VersionDetailsTabs/constants'

import type { VersionListExpandedColumnProps } from './types'
import css from './VersionListTable.module.scss'

type CellTypeWithActions<D extends Record<string, any>, V = any> = TableInstance<D> & {
  column: ColumnInstance<D>
  row: Row<D>
  cell: Cell<D, V>
  value: CellValue<V>
}

type CellType = Renderer<CellTypeWithActions<ArtifactVersionMetadata>>

export const ToggleAccordionCell: Renderer<{
  row: UseExpandedRowProps<ArtifactVersionMetadata> & Row<ArtifactVersionMetadata>
  column: ColumnInstance<ArtifactVersionMetadata> & VersionListExpandedColumnProps
}> = ({ row, column }) => {
  const { expandedRows, setExpandedRows } = column
  const data = row.original
  return (
    <TableCells.ToggleAccordionCell
      expandedRows={expandedRows}
      setExpandedRows={setExpandedRows}
      value={data.name}
      initialIsExpanded={row.isExpanded}
      getToggleRowExpandedProps={row.getToggleRowExpandedProps}
      onToggleRowExpanded={row.toggleRowExpanded}
    />
  )
}

export const VersionNameCell: CellType = ({ value, row }) => {
  const { original } = row
  const routes = useRoutes()
  const { isQuarantined, quarantineReason } = original
  const pathParams = useDecodedParams<ArtifactDetailsPathParams>()
  return (
    <Layout.Horizontal className={css.nameCellContainer} spacing="small">
      {isQuarantined ? (
        <TableCells.QuarantineIcon reason={quarantineReason} />
      ) : (
        <Icon name="store-artifact-bundle" size={24} />
      )}
      <Link
        to={routes.toARVersionDetailsTab({
          repositoryIdentifier: pathParams.repositoryIdentifier,
          artifactIdentifier: pathParams.artifactIdentifier,
          versionIdentifier: value,
          versionTab: VersionDetailsTab.OVERVIEW,
          artifactType: (original.artifactType ?? LocalArtifactType.ARTIFACTS) as LocalArtifactType
        })}>
        <Text lineClamp={1} color={Color.PRIMARY_7} font={{ variation: FontVariation.SMALL }}>
          {value}
        </Text>
      </Link>
    </Layout.Horizontal>
  )
}

export const VersionSizeCell: CellType = ({ value }) => {
  return <TableCells.SizeCell value={value} />
}

export const VersionDeploymentsCell: CellType = ({ row }) => {
  const { original } = row
  return (
    <TableCells.DeploymentsCell
      prodCount={defaultTo(original.deploymentMetadata?.prodEnvCount, 0)}
      nonProdCount={defaultTo(original.deploymentMetadata?.nonProdEnvCount, 0)}
    />
  )
}

export const VersionDigestsCell: CellType = ({ value }) => {
  return <TableCells.CountCell value={value} />
}

export const VersionFileCountCell: CellType = ({ value }) => {
  return <TableCells.CountCell value={value} />
}

export const PullCommandCell: CellType = ({ value }) => {
  const { getString } = useStrings()
  if (!value) return <>{getString('na')}</>
  return <TableCells.CopyTextCell value={value}>{getString('copy')}</TableCells.CopyTextCell>
}

export const VersionDownloadsCell: CellType = ({ value }) => {
  return <TableCells.CountCell value={value} icon="download-box" iconProps={{ size: 12 }} />
}

export const VersionPublishedAtCell: CellType = ({ value }) => {
  return <TableCells.LastModifiedCell value={value} />
}

export const VersionActionsCell: CellType = ({ row }) => {
  const { original } = row
  const { artifactIdentifier, repositoryIdentifier } = useDecodedParams<ArtifactDetailsPathParams>()
  return (
    <VersionActionsWidget
      data={original}
      packageType={original.packageType as RepositoryPackageType}
      pageType={PageType.Table}
      repoKey={repositoryIdentifier}
      artifactKey={artifactIdentifier}
      versionKey={original.name}
    />
  )
}
