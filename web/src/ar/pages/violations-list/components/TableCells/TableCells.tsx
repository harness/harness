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
import type { ArtifactScan } from '@harnessio/react-har-service-client'
import type { TableInstance, ColumnInstance, Row, Cell, CellValue, Renderer } from 'react-table'
import { Button, ButtonSize, ButtonVariation, Layout } from '@harnessio/uicore'
import TableCells from '@ar/components/TableCells/TableCells'
import RepositoryIcon from '@ar/frameworks/RepositoryStep/RepositoryIcon'
import type { RepositoryPackageType } from '@ar/common/types'
import { useRoutes } from '@ar/hooks'
import { VersionDetailsTab } from '@ar/pages/version-details/components/VersionDetailsTabs/constants'
import ScanBadgeComponent from '@ar/components/Badge/ScanBadge'
import { useStrings } from '@ar/frameworks/strings'
import useGetPolicySetDetailsPageUrl from '../../hooks/useGetPolicySetDetailsPageUrl'
import { useViolationDetailsModal } from '../../hooks/useViolationDetailsModal/useViolationDetailsModal'

type CellTypeWithActions<D extends Record<string, any>, V = any> = TableInstance<D> & {
  column: ColumnInstance<D>
  row: Row<D>
  cell: Cell<D, V>
  value: CellValue<V>
}

type CellType = Renderer<CellTypeWithActions<ArtifactScan>>

export const DependencyAndVersionCell: CellType = ({ row }) => {
  const { original } = row
  const { packageName, version, packageType, registryName } = original
  const routes = useRoutes()
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

export const PolicySetName: CellType = ({ row }) => {
  const { original } = row
  const { policySetName, policySetRef } = original
  const getPolicySetDetailsPageUrl = useGetPolicySetDetailsPageUrl(policySetRef)
  return <TableCells.LinkCell linkTo={getPolicySetDetailsPageUrl} label={policySetName || policySetRef} />
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
