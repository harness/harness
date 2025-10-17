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
import { Color, FontVariation } from '@harnessio/design-system'
import { Layout, Text } from '@harnessio/uicore'
import type { RegistryMetadata } from '@harnessio/react-har-service-client'
import type { Cell, CellValue, ColumnInstance, Renderer, Row, TableInstance } from 'react-table'

import ScopeBadge from '@ar/components/Badge/ScopeBadge'
import { useStrings } from '@ar/frameworks/strings/String'
import TableCells from '@ar/components/TableCells/TableCells'
import { getEntityScopeType } from '@ar/hooks/useGetPageScope'
import LabelsPopover from '@ar/components/LabelsPopover/LabelsPopover'
import RepositoryIcon from '@ar/frameworks/RepositoryStep/RepositoryIcon'
import { RepositoryVisibilityOptions } from '@ar/pages/repository-details/constants'
import DescriptionPopover from '@ar/components/DescriptionPopover/DescriptionPopover'
import RepositoryActionsWidget from '@ar/frameworks/RepositoryStep/RepositoryActionsWidget'
import { PageType, RepositoryVisibility, type RepositoryConfigType, type RepositoryPackageType } from '@ar/common/types'
import useGetScopeFromRegistryPath from '@ar/pages/repository-details/hooks/useGetScopeFromRegistryPath/useGetScopeFromRegistryPath'

import css from './RepositoryListTable.module.scss'

type CellTypeWithActions<D extends Record<string, any>, V = any> = TableInstance<D> & {
  column: ColumnInstance<D>
  row: Row<D>
  cell: Cell<D, V>
  value: CellValue<V>
}

type CellType = Renderer<CellTypeWithActions<RegistryMetadata>>

export const RepositoryNameCell: CellType = ({ value, row }) => {
  const { original } = row
  const { labels, description, packageType, isPublic } = original
  const { getString } = useStrings()
  const subText = isPublic
    ? RepositoryVisibilityOptions[RepositoryVisibility.PUBLIC].label
    : RepositoryVisibilityOptions[RepositoryVisibility.PRIVATE].label
  return (
    <Layout.Vertical spacing="xsmall">
      <Layout.Horizontal className={css.nameCellContainer} spacing="small">
        <RepositoryIcon packageType={packageType as RepositoryPackageType} />
        <Text lineClamp={1} color={Color.GREY_900} font={{ size: 'small' }}>
          {value}
        </Text>
        <LabelsPopover labels={defaultTo(labels, [])} />
        {description && <DescriptionPopover text={description} />}
      </Layout.Horizontal>
      <Text className={css.subText} lineClamp={1} color={Color.GREY_500} font={{ variation: FontVariation.SMALL }}>
        {getString(subText)}
      </Text>
    </Layout.Vertical>
  )
}

export const RepositoryScopeCell: CellType = ({ row }) => {
  const { original } = row
  const { path } = original
  const { getScopeFromRegistryPath } = useGetScopeFromRegistryPath()

  const scope = getScopeFromRegistryPath(path)
  const entityScope = getEntityScopeType(scope)
  const helperText = [scope.orgIdentifier, scope.projectIdentifier].filter(Boolean).join('/')
  return <ScopeBadge scope={entityScope} helperText={helperText} />
}

export const RepositoryLocationBadgeCell: CellType = ({ value }) => {
  return <TableCells.RepositoryLocationBadgeCell value={value} />
}

export const LastModifiedCell: CellType = ({ value }) => {
  return <TableCells.LastModifiedCell value={value} />
}

export const RepositoryUrlCell: CellType = ({ value }) => {
  const { getString } = useStrings()
  return <TableCells.CopyUrlCell value={value}>{getString('repositoryList.table.copyUrl')}</TableCells.CopyUrlCell>
}

export const RepositorySizeCell: CellType = ({ value }) => {
  return <TableCells.SizeCell value={value} />
}

export const RepositoryArtifactsCell: CellType = ({ value }) => {
  return <TableCells.CountCell value={value} icon="store-artifact-bundle" />
}

export const RepositoryDownloadsCell: CellType = ({ value }) => {
  return <TableCells.CountCell value={value} icon="download-box" />
}

export const RepositoryActionsCell: CellType = ({ row }) => {
  return (
    <RepositoryActionsWidget
      type={row.original.type as RepositoryConfigType}
      packageType={row.original.packageType as RepositoryPackageType}
      data={row.original}
      readonly={false}
      pageType={PageType.Table}
    />
  )
}
