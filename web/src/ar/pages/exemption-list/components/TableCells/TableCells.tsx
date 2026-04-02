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
import { Layout } from '@harnessio/uicore'
import type { FirewallExceptionResponseV3 } from '@harnessio/react-har-service-client'
import type { TableInstance, ColumnInstance, Row, Cell, CellValue, Renderer } from 'react-table'

import { useRoutes } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'
import TableCells from '@ar/components/TableCells/TableCells'
import type { RepositoryPackageType } from '@ar/common/types'
import ExemptionStatusBadge from '@ar/components/Badge/ExemptionStatusBadge'
import RepositoryIcon from '@ar/frameworks/RepositoryStep/RepositoryIcon'

import ExemptionActions from '../ExemptionActions/ExemptionActions'

import css from './TableCells.module.scss'

type CellTypeWithActions<D extends Record<string, any>, V = any> = TableInstance<D> & {
  column: ColumnInstance<D>
  row: Row<D>
  cell: Cell<D, V>
  value: CellValue<V>
}

type CellType = Renderer<CellTypeWithActions<FirewallExceptionResponseV3>>

export const PackageNameCell: CellType = ({ row }) => {
  const { original } = row
  const { packageType, packageName, registryName } = original
  const routes = useRoutes()
  return (
    <TableCells.LinkCell
      prefix={<RepositoryIcon packageType={packageType as RepositoryPackageType} />}
      linkTo={routes.toARRedirect({
        packageType: packageType as RepositoryPackageType,
        registryId: registryName || '',
        artifactId: packageName
      })}
      label={packageName}
    />
  )
}

export const VersionListCell: CellType = ({ row }) => {
  const { original } = row
  const { versionList } = original
  const { getString } = useStrings()
  if (!versionList || versionList.length === 0) {
    return <TableCells.TextCell value={getString('na')} />
  }
  return <TableCells.TextCell value={versionList.join(', ')} />
}

export const RepositoryNameCell: CellType = ({ row }) => {
  const { original } = row
  const { registryName } = original
  const routes = useRoutes()
  return (
    <TableCells.LinkCell
      linkTo={routes.toARRepositoryDetails({
        repositoryIdentifier: registryName || ''
      })}
      label={registryName || ''}
    />
  )
}

export const StatusCell: CellType = ({ row }) => {
  const { original } = row
  const { status, notes } = original
  return <ExemptionStatusBadge status={status} helperText={notes || ''} />
}

export const RequestedAtCell: CellType = ({ row }) => {
  const { original } = row
  const { createdAt } = original
  const { getString } = useStrings()
  if (!createdAt) return <TableCells.TextCell value={getString('na')} />
  return <TableCells.LastModifiedCell value={createdAt} />
}

export const ExemptionUpdatedAtCell: CellType = ({ row }) => {
  const { original } = row
  const { updatedAt } = original
  const { getString } = useStrings()
  if (!updatedAt) return <TableCells.TextCell value={getString('na')} />
  return <TableCells.LastModifiedCell value={updatedAt} />
}

export const ExpireAtCell: CellType = ({ row }) => {
  const { original } = row
  const { expirationAt, expireAfter } = original
  const { getString } = useStrings()
  if (expirationAt) {
    return <TableCells.LastModifiedCell value={expirationAt} />
  }
  if (expireAfter) {
    return <TableCells.TextCell value={getString('exemptionList.expireAfter', { days: expireAfter })} />
  }
  return <TableCells.TextCell value={getString('na')} />
}

export const ExemptionActionsCell: CellType = ({ row }) => {
  const { original } = row
  const { getString } = useStrings()
  const routes = useRoutes()

  return (
    <Layout.Horizontal flex={{ justifyContent: 'space-between', alignItems: 'center' }}>
      <Link
        to={routes.toARDependencyFirewallExemptionDetails({ exemptionId: original.exceptionId })}
        className={css.actionButton}>
        {getString('details')}
      </Link>
      <ExemptionActions exemptionId={original.exceptionId} data={original} />
    </Layout.Horizontal>
  )
}
