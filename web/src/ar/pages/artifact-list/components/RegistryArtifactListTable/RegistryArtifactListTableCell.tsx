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
import { Position } from '@blueprintjs/core'
import type { Cell, CellValue, ColumnInstance, Renderer, Row, TableInstance } from 'react-table'
import { Layout, Text } from '@harnessio/uicore'
import { Color } from '@harnessio/design-system'
import type { RegistryArtifactMetadata } from '@harnessio/react-har-service-client'

import { useRoutes } from '@ar/hooks'
import Tag from '@ar/components/Tag/Tag'
import { useStrings } from '@ar/frameworks/strings'
import TableCells from '@ar/components/TableCells/TableCells'
import LabelsPopover from '@ar/components/LabelsPopover/LabelsPopover'
import RepositoryIcon from '@ar/frameworks/RepositoryStep/RepositoryIcon'
import type { RepositoryPackageType } from '@ar/common/types'
import { RepositoryDetailsTab } from '@ar/pages/repository-details/constants'

type CellTypeWithActions<D extends Record<string, any>, V = any> = TableInstance<D> & {
  column: ColumnInstance<D>
  row: Row<D>
  cell: Cell<D, V>
  value: CellValue<V>
}

type CellType = Renderer<CellTypeWithActions<RegistryArtifactMetadata>>

type RegistryArtifactNameCellActionProps = {
  onClickLabel: (val: string) => void
}

export const RegistryArtifactNameCell: Renderer<{
  row: Row<RegistryArtifactMetadata>
  column: ColumnInstance<RegistryArtifactMetadata> & RegistryArtifactNameCellActionProps
}> = ({ row, column }) => {
  const { original } = row
  const { onClickLabel } = column
  const routes = useRoutes()
  const value = original.name
  return (
    <TableCells.LinkCell
      prefix={<RepositoryIcon packageType={original.packageType as RepositoryPackageType} iconProps={{ size: 24 }} />}
      linkTo={routes.toARArtifactDetails({
        repositoryIdentifier: original.registryIdentifier,
        artifactIdentifier: value
      })}
      label={value}
      postfix={
        <LabelsPopover
          popoverProps={{
            position: Position.RIGHT
          }}
          labels={defaultTo(original.labels, [])}
          tagProps={{
            interactive: true,
            onClick: e => onClickLabel(e.currentTarget.ariaValueText as string)
          }}
        />
      }
    />
  )
}

export const RegistryArtifactTagsCell: CellType = ({ value }) => {
  const { getString } = useStrings()
  if (!Array.isArray(value) || !value.length) {
    return (
      <Text color={Color.GREY_900} font={{ size: 'small' }}>
        {getString('na')}
      </Text>
    )
  }
  return (
    <Layout.Horizontal spacing="small">
      {Array.isArray(value) &&
        value.map(each => (
          <Tag key={each} isArtifactTag>
            {each}
          </Tag>
        ))}
    </Layout.Horizontal>
  )
}

export const RepositoryNameCell: CellType = ({ value }) => {
  const routes = useRoutes()
  return (
    <TableCells.LinkCell
      linkTo={routes.toARRepositoryDetails({
        repositoryIdentifier: value,
        tab: RepositoryDetailsTab.PACKAGES
      })}
      label={value}
    />
  )
}

export const RegistryArtifactDownloadsCell: CellType = ({ value }) => {
  return <TableCells.CountCell value={value} icon="download-box" iconProps={{ size: 12 }} />
}

export const RegistryArtifactLatestUpdatedCell: CellType = ({ row }) => {
  const { getString } = useStrings()
  const { original } = row
  const { latestVersion, lastModified } = original || {}
  if (!latestVersion) {
    return (
      <Text color={Color.GREY_900} font={{ size: 'small' }}>
        {getString('na')}
      </Text>
    )
  }
  return (
    <Layout.Vertical spacing="small">
      <Text color={Color.PRIMARY_7} font={{ size: 'small' }}>
        {latestVersion}
      </Text>
      <TableCells.LastModifiedCell value={defaultTo(lastModified, 0)} />
    </Layout.Vertical>
  )
}
