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
import { Color } from '@harnessio/design-system'
import type { ArtifactScanV3 } from '@harnessio/react-har-service-client'
import { Button, ButtonSize, ButtonVariation, Layout, Text } from '@harnessio/uicore'
import type { TableInstance, ColumnInstance, Row, Cell, CellValue, Renderer } from 'react-table'

import TableCells from '@ar/components/TableCells/TableCells'
import RepositoryIcon from '@ar/frameworks/RepositoryStep/RepositoryIcon'
import type { RepositoryPackageType } from '@ar/common/types'
import { useRoutes } from '@ar/hooks'
import { VersionDetailsTab } from '@ar/pages/version-details/components/VersionDetailsTabs/constants'
import ScanBadgeComponent from '@ar/components/Badge/ScanBadge'
import { useStrings } from '@ar/frameworks/strings'
import { useViolationDetailsModal } from '../../hooks/useViolationDetailsModal/useViolationDetailsModal'
import css from './TableCells.module.scss'

type CellTypeWithActions<D extends Record<string, any>, V = any> = TableInstance<D> & {
  column: ColumnInstance<D>
  row: Row<D>
  cell: Cell<D, V>
  value: CellValue<V>
}

export interface PolicySetSpec {
  name?: string
  identifier: string
  scanId: string
}

type CellType = Renderer<CellTypeWithActions<ArtifactScanV3>>

export const DependencyAndVersionCell: CellType = ({ row }) => {
  const { original } = row
  const { packageName, version, packageType, registryName, scanStatus } = original
  const routes = useRoutes()
  if (scanStatus === 'BLOCKED') {
    return (
      <Layout.Horizontal className={css.dependencyNameCell}>
        <RepositoryIcon packageType={packageType as RepositoryPackageType} />
        <Layout.Vertical className={css.labelCellContainer}>
          <Text color={Color.GREY_800} lineClamp={1}>
            {packageName}
          </Text>
          <Text lineClamp={1}>{version}</Text>
        </Layout.Vertical>
      </Layout.Horizontal>
    )
  }
  return (
    <Layout.Vertical>
      <TableCells.LinkCell
        prefix={<RepositoryIcon packageType={packageType as RepositoryPackageType} />}
        linkTo={routes.toARRedirect({
          packageType: packageType as RepositoryPackageType,
          registryId: registryName,
          artifactId: packageName,
          versionId: version,
          versionDetailsTab: VersionDetailsTab.OVERVIEW
        })}
        label={packageName}
        subLabel={version}
      />
    </Layout.Vertical>
  )
}

export const RegistryNameCell: CellType = ({ row }) => {
  const { original } = row
  const { registryName } = original
  const routes = useRoutes()
  return (
    <TableCells.LinkCell
      linkTo={routes.toARRepositoryDetails({
        repositoryIdentifier: registryName
      })}
      label={registryName}
    />
  )
}

export const LastEvaluatedAtCell: CellType = ({ row }) => {
  const { original } = row
  const { lastEvaluatedAt } = original
  const { getString } = useStrings()
  if (!lastEvaluatedAt) return <TableCells.TextCell value={getString('na')} />
  return <TableCells.LastModifiedCell value={lastEvaluatedAt} />
}

export const StatusCell: CellType = ({ row }) => {
  const { original } = row
  const { scanStatus, id } = original
  return <ScanBadgeComponent scanId={id} status={scanStatus} />
}

export const ViolationActionsCell: CellType = ({ row }) => {
  const { original } = row
  const { id } = original
  const { getString } = useStrings()
  const [showModal] = useViolationDetailsModal({ scanId: id })
  return (
    <Button variation={ButtonVariation.SECONDARY} size={ButtonSize.SMALL} onClick={showModal}>
      {getString('violationsList.table.columns.actions.violationDetails')}
    </Button>
  )
}
