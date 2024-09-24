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

import React, { useState } from 'react'
import { defaultTo } from 'lodash-es'
import { Link, useHistory } from 'react-router-dom'
import { Menu, Position } from '@blueprintjs/core'
import { Color, FontVariation } from '@harnessio/design-system'
import { Button, ButtonVariation, Layout, Text } from '@harnessio/uicore'
import type { Cell, CellValue, ColumnInstance, Renderer, Row, TableInstance } from 'react-table'
import type { ArtifactMetadata, StoDigestMetadata } from '@harnessio/react-har-service-client'

import { useParentComponents, useRoutes } from '@ar/hooks'
import TableCells from '@ar/components/TableCells/TableCells'
import type { RepositoryPackageType } from '@ar/common/types'
import LabelsPopover from '@ar/components/LabelsPopover/LabelsPopover'
import RepositoryIcon from '@ar/frameworks/RepositoryStep/RepositoryIcon'
import { useStrings } from '@ar/frameworks/strings'
import { getShortDigest } from '@ar/pages/digest-list/utils'
import { VersionDetailsTab } from '@ar/pages/version-details/components/VersionDetailsTabs/constants'

import css from './ArtifactListTable.module.scss'

type CellTypeWithActions<D extends Record<string, any>, V = any> = TableInstance<D> & {
  column: ColumnInstance<D>
  row: Row<D>
  cell: Cell<D, V>
  value: CellValue<V>
}

type CellType = Renderer<CellTypeWithActions<ArtifactMetadata>>

type ArtifactNameCellActionProps = {
  onClickLabel: (val: string) => void
}

export const ArtifactNameCell: Renderer<{
  row: Row<ArtifactMetadata>
  column: ColumnInstance<ArtifactMetadata> & ArtifactNameCellActionProps
}> = ({ row, column }) => {
  const { original } = row
  const { onClickLabel } = column
  const routes = useRoutes()
  const { name: value, version } = original
  return (
    <Layout.Vertical>
      <TableCells.LinkCell
        prefix={<RepositoryIcon packageType={original.packageType as RepositoryPackageType} iconProps={{ size: 24 }} />}
        linkTo={routes.toARVersionDetailsTab({
          repositoryIdentifier: original.registryIdentifier,
          artifactIdentifier: value,
          versionIdentifier: defaultTo(version, ''),
          versionTab: VersionDetailsTab.OVERVIEW
        })}
        label={value}
        subLabel={version}
        postfix={
          <LabelsPopover
            popoverProps={{
              position: Position.RIGHT
            }}
            labels={defaultTo(original.labels, [])}
            tagProps={{
              interactive: true,
              onClick: e => {
                if (typeof onClickLabel === 'function') {
                  onClickLabel(e.currentTarget.ariaValueText as string)
                }
              }
            }}
          />
        }
      />
    </Layout.Vertical>
  )
}

export const ArtifactDownloadsCell: CellType = ({ value }) => {
  return <TableCells.CountCell value={value} icon="download-box" iconProps={{ size: 12 }} />
}

export const ArtifactDeploymentsCell: CellType = ({ row }) => {
  const { original } = row
  const { deploymentMetadata } = original
  const { nonProdEnvCount, prodEnvCount } = deploymentMetadata || {}
  return <TableCells.DeploymentsCell prodCount={prodEnvCount} nonProdCount={nonProdEnvCount} />
}

export const ArtifactListPullCommandCell: CellType = ({ value }) => {
  const { getString } = useStrings()
  return <TableCells.CopyTextCell value={value}>{getString('copy')}</TableCells.CopyTextCell>
}

export const ArtifactListVulnerabilitiesCell: CellType = ({ row }) => {
  const { original } = row
  const { stoMetadata, registryIdentifier, name, version } = original
  const { scannedCount, totalCount, digestMetadata } = stoMetadata || {}
  const [isOptionsOpen, setIsOptionsOpen] = useState(false)
  const { getString } = useStrings()
  const { RbacMenuItem } = useParentComponents()
  const routes = useRoutes()
  const history = useHistory()

  const handleRenderDigestMenuItem = (digest: StoDigestMetadata) => {
    return (
      <RbacMenuItem
        text={getString('artifactList.table.actions.VulnerabilityStatus.digestMenuItemText', {
          archName: digest.osArch,
          digest: getShortDigest(digest.digest || '')
        })}
        onClick={() => {
          const url = routes.toARVersionDetailsTab({
            repositoryIdentifier: registryIdentifier,
            artifactIdentifier: name,
            versionIdentifier: version as string,
            versionTab: VersionDetailsTab.SECURITY_TESTS,
            pipelineIdentifier: digest.stoPipelineId,
            executionIdentifier: digest.stoExecutionId
          })
          history.push(`${url}?digest=${digest.digest}`)
        }}
      />
    )
  }

  if (!scannedCount) {
    return <Text>{getString('artifactList.table.actions.VulnerabilityStatus.nonScanned')}</Text>
  }

  return (
    <Button
      className={css.cellBtn}
      tooltip={
        <Menu
          className={css.optionsMenu}
          onClick={e => {
            e.stopPropagation()
            setIsOptionsOpen(false)
          }}>
          {digestMetadata?.map(handleRenderDigestMenuItem)}
        </Menu>
      }
      tooltipProps={{
        interactionKind: 'click',
        onInteraction: nextOpenState => {
          setIsOptionsOpen(nextOpenState)
        },
        isOpen: isOptionsOpen,
        position: Position.BOTTOM
      }}
      variation={ButtonVariation.LINK}>
      <Text font={{ variation: FontVariation.BODY }} color={Color.PRIMARY_7}>
        {getString('artifactList.table.actions.VulnerabilityStatus.partiallyScanned', {
          total: defaultTo(totalCount, 0),
          scanned: defaultTo(scannedCount, 0)
        })}
      </Text>
    </Button>
  )
}

export const ScanStatusCell: CellType = ({ row }) => {
  const { original } = row
  const router = useRoutes()
  const { version = '', name, registryIdentifier } = original
  const { getString } = useStrings()
  const linkTo = router.toARVersionDetailsTab({
    repositoryIdentifier: registryIdentifier,
    artifactIdentifier: name,
    versionIdentifier: version,
    versionTab: VersionDetailsTab.OVERVIEW
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
  return <TableCells.LastModifiedCell value={defaultTo(original.lastModified, 0)} />
}
