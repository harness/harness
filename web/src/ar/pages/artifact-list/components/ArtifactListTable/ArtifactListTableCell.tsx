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
import { Link, useHistory } from 'react-router-dom'
import { Color } from '@harnessio/design-system'
import { Layout, Text } from '@harnessio/uicore'
import type { ArtifactMetadata, VersionMetadata } from '@harnessio/react-har-service-client'
import type { Cell, CellValue, ColumnInstance, Renderer, Row, TableInstance, UseExpandedRowProps } from 'react-table'

import { useGetVersionDisplayName, useRoutes } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'
import TableCells from '@ar/components/TableCells/TableCells'
import versionFactory from '@ar/frameworks/Version/VersionFactory'
import { PageType, RepositoryPackageType } from '@ar/common/types'
import { useGetRepositoryTypes } from '@ar/hooks/useGetRepositoryTypes'
import RepositoryIcon from '@ar/frameworks/RepositoryStep/RepositoryIcon'
import VersionActionsWidget from '@ar/frameworks/Version/VersionActionsWidget'
import { VersionDetailsTab } from '@ar/pages/version-details/components/VersionDetailsTabs/constants'
import { LocalArtifactType } from '@ar/pages/repository-details/constants'
import { OCITags } from '@ar/pages/version-list/components/VersionListTable/VersionListCell'
import { useViolationDetailsModal } from '@ar/pages/violations-list/hooks/useViolationDetailsModal/useViolationDetailsModal'
import ScanBadge from '@ar/components/Badge/ScanBadge'
import repositoryFactory from '@ar/frameworks/RepositoryStep/RepositoryFactory'

type CellTypeWithActions<D extends Record<string, any>, V = any> = TableInstance<D> & {
  column: ColumnInstance<D>
  row: Row<D>
  cell: Cell<D, V>
  value: CellValue<V>
}

type CellType = Renderer<CellTypeWithActions<VersionMetadata>>

export type ArtifactListExpandedColumnProps = {
  expandedRows: Set<string>
  setExpandedRows: React.Dispatch<React.SetStateAction<Set<string>>>
  getRowId: (rowData: ArtifactMetadata) => string
}

export const ToggleAccordionCell: Renderer<{
  row: UseExpandedRowProps<ArtifactMetadata> & Row<ArtifactMetadata>
  column: ColumnInstance<ArtifactMetadata> & ArtifactListExpandedColumnProps
}> = ({ row, column }) => {
  const { expandedRows, setExpandedRows, getRowId } = column
  const data = row.original
  const repositoryType = versionFactory?.getVersionType(data.packageType)
  if (!repositoryType?.getHasArtifactRowSubComponent()) return <></>
  return (
    <TableCells.ToggleAccordionCell
      expandedRows={expandedRows}
      setExpandedRows={setExpandedRows}
      value={getRowId(data)}
      initialIsExpanded={row.isExpanded}
      getToggleRowExpandedProps={row.getToggleRowExpandedProps}
      onToggleRowExpanded={row.toggleRowExpanded}
    />
  )
}

export const ArtifactNameCell: CellType = ({ row }) => {
  const { original } = row
  const routes = useRoutes()
  const {
    package: packageName,
    version,
    packageType,
    registryIdentifier,
    artifactType,
    isQuarantined,
    quarantineReason
  } = original
  const versionDisplayName = useGetVersionDisplayName(packageType as RepositoryPackageType, version)
  // TODO: move this to factory in future
  const isOCIPackageType = [RepositoryPackageType.HELM, RepositoryPackageType.DOCKER].includes(
    packageType as RepositoryPackageType
  )
  return (
    <Layout.Vertical>
      <TableCells.LinkCell
        prefix={
          isQuarantined ? (
            <TableCells.QuarantineIcon reason={quarantineReason} />
          ) : (
            <RepositoryIcon packageType={original.packageType as RepositoryPackageType} />
          )
        }
        linkTo={routes.toARRedirect({
          packageType: packageType as RepositoryPackageType,
          registryId: registryIdentifier,
          artifactId: packageName,
          versionId: version,
          versionDetailsTab: VersionDetailsTab.OVERVIEW,
          artifactType
        })}
        label={packageName}
        subLabel={isOCIPackageType ? versionDisplayName : undefined}
      />
    </Layout.Vertical>
  )
}

export const ArtifactVersionCell: CellType = props => {
  const { original } = props.row
  const { package: value, version, packageType, registryIdentifier, artifactType, metadata } = original
  const routes = useRoutes()
  const history = useHistory()

  // TODO: move this package type specific logic to factory in future
  switch (packageType) {
    case RepositoryPackageType.DOCKER:
    case RepositoryPackageType.HELM:
      return (
        <OCITags
          tags={metadata?.tags || []}
          onClick={tag => {
            history.push(
              routes.toARRedirect({
                packageType: packageType as RepositoryPackageType,
                registryId: registryIdentifier,
                artifactId: value,
                versionId: version,
                versionDetailsTab: VersionDetailsTab.OVERVIEW,
                artifactType,
                tag
              })
            )
          }}
        />
      )
    default:
      return (
        <TableCells.LinkCell
          linkTo={routes.toARRedirect({
            packageType: packageType as RepositoryPackageType,
            registryId: registryIdentifier,
            artifactId: value,
            versionId: version,
            versionDetailsTab: VersionDetailsTab.OVERVIEW,
            artifactType
          })}
          label={version}
        />
      )
  }
}

export const ArtifactDownloadsCell: CellType = ({ value }) => {
  return <TableCells.CountCell value={value} icon="download-box" iconProps={{ size: 12 }} />
}

export const ArtifactPackageTypeCell: CellType = ({ value }) => {
  const repositoryTypes = useGetRepositoryTypes()
  const { getString } = useStrings()
  const typeConfig = repositoryTypes.find(type => type.value === value)
  return <TableCells.TextCell value={typeConfig ? getString(typeConfig.label) : value} />
}

export const ArtifactSizeCell: CellType = ({ value }) => {
  return <TableCells.TextCell value={value} />
}

export const ArtifactDeploymentsCell: CellType = ({ row }) => {
  const { original } = row
  const { deploymentMetadata } = original
  const { nonProdEnvCount, prodEnvCount } = deploymentMetadata || {}
  return <TableCells.DeploymentsCell prodCount={prodEnvCount} nonProdCount={nonProdEnvCount} />
}

export const ArtifactListPullCommandCell: CellType = ({ value, row }) => {
  const { original } = row
  const routes = useRoutes()
  const { packageType, artifactType } = original
  const { getString } = useStrings()
  switch (packageType) {
    case RepositoryPackageType.MAVEN:
    case RepositoryPackageType.GENERIC:
      return (
        <TableCells.LinkCell
          linkTo={routes.toARVersionDetailsTab({
            repositoryIdentifier: original.registryIdentifier,
            artifactIdentifier: original.package,
            versionIdentifier: original.version as string,
            versionTab: VersionDetailsTab.ARTIFACT_DETAILS,
            artifactType: (artifactType ?? LocalArtifactType.ARTIFACTS) as LocalArtifactType
          })}
          label={getString('artifactList.viewArtifactDetails')}
        />
      )
    default:
      return <TableCells.CopyTextCell value={value}>{getString('copy')}</TableCells.CopyTextCell>
  }
}

export const ScanStatusCell: CellType = ({ row }) => {
  const { original } = row
  const router = useRoutes()
  const { version = '', package: packageName, registryIdentifier, artifactType } = original
  const { getString } = useStrings()
  const linkTo = router.toARVersionDetailsTab({
    repositoryIdentifier: registryIdentifier,
    artifactIdentifier: packageName,
    versionIdentifier: version,
    versionTab: VersionDetailsTab.OVERVIEW,
    artifactType: (artifactType ?? LocalArtifactType.ARTIFACTS) as LocalArtifactType
  })
  return (
    <Link to={linkTo} target="_blank">
      <Text color={Color.PRIMARY_7} rightIcon="main-share" rightIconProps={{ size: 12, color: Color.PRIMARY_7 }}>
        {getString('artifactList.table.actions.VulnerabilityStatus.scanStatus')}
      </Text>
    </Link>
  )
}

export const LatestArtifactCell: CellType = ({ row }) => {
  const { original } = row
  const { lastModified, deletedAt, isDeleted } = original
  const lastModifiedTime = isDeleted ? deletedAt : lastModified
  return <TableCells.LastModifiedCell value={defaultTo(lastModifiedTime, 0)} />
}

export const ArtifactVersionActions: CellType = ({ row }) => {
  const { original } = row
  const { digestCount = 0 } = original
  return (
    <VersionActionsWidget
      pageType={PageType.GlobalList}
      data={original}
      repoKey={original.registryIdentifier}
      artifactKey={original.package}
      versionKey={original.version}
      packageType={original.packageType as RepositoryPackageType}
      digestCount={digestCount}
    />
  )
}

export const ViolationScanStatusCell: CellType = ({ row }) => {
  const data = row.original
  const { scanStatus, scanId, packageType } = data
  const { getString } = useStrings()
  const [showModal] = useViolationDetailsModal({ scanId: scanId || '' })
  const repositoryType = repositoryFactory.getRepositoryType(packageType)
  const isViolationScanSupported = repositoryType?.getIsDependencyFirewallSupported()
  if (!isViolationScanSupported) return <TableCells.TextCell value={getString('na')} />
  return <ScanBadge scanId={scanId} status={scanStatus} onClick={() => showModal()} />
}
