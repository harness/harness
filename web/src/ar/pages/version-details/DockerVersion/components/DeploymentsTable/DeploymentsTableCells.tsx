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
import { Layout, Text } from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'
import type { ArtifactDeploymentsDetail } from '@harnessio/react-har-service-client'
import type { Cell, CellValue, ColumnInstance, Renderer, Row, TableInstance } from 'react-table'

import { useAppStore } from '@ar/hooks'
import { EnvironmentType } from '@ar/common/types'
import { useStrings } from '@ar/frameworks/strings'
import { NonProdTag, ProdTag } from '@ar/components/Tag/Tags'
import TableCells from '@ar/components/TableCells/TableCells'
import { useParentUtils } from '@ar/hooks/useParentUtils'

type CellTypeWithActions<D extends Record<string, any>, V = any> = TableInstance<D> & {
  column: ColumnInstance<D>
  row: Row<D>
  cell: Cell<D, V>
  value: CellValue<V>
}

type CellType = Renderer<CellTypeWithActions<ArtifactDeploymentsDetail>>

export const EnvironmentNameCell: CellType = ({ row }) => {
  const { original } = row
  const { envName, envIdentifier } = original || {}
  const { getString } = useStrings()
  return (
    <Layout.Vertical>
      <Text color={Color.GREY_900} font={{ variation: FontVariation.BODY }}>
        {envName}
      </Text>
      <Text color={Color.GREY_500} font={{ variation: FontVariation.SMALL }}>
        {getString('id', { id: envIdentifier })}
      </Text>
    </Layout.Vertical>
  )
}

export const EnvironmentTypeCell: CellType = ({ row }) => {
  const { original } = row
  const { envType } = original || {}
  switch (envType) {
    case EnvironmentType.Prod:
      return <ProdTag />
    case EnvironmentType.NonProd:
    default:
      return <NonProdTag />
  }
}

export const ServiceListCell: CellType = ({ row }) => {
  const { original } = row
  const { serviceIdentifier, serviceName } = original || {}
  const { scope } = useAppStore()
  const { getRouteToServiceDetailsView } = useParentUtils()
  if (getRouteToServiceDetailsView && serviceIdentifier) {
    return (
      <Link
        key={serviceIdentifier}
        to={getRouteToServiceDetailsView({
          accountId: scope.accountId,
          orgIdentifier: scope.orgIdentifier,
          projectIdentifier: scope.projectIdentifier,
          serviceId: serviceIdentifier
        })}>
        {serviceName}
      </Link>
    )
  }
  return (
    <Text key={serviceIdentifier} color={Color.GREY_900} font={{ variation: FontVariation.BODY }}>
      {serviceName}
    </Text>
  )
}

export const DeploymentPipelineCell: CellType = ({ row }) => {
  const { original } = row
  const { pipelineId, lastPipelineExecutionName } = original || {}
  const { getString } = useStrings()
  const { scope } = useAppStore()
  const { getRouteToPipelineExecutionView } = useParentUtils()
  return (
    <Layout.Vertical>
      {pipelineId && getRouteToPipelineExecutionView ? (
        <Link
          to={getRouteToPipelineExecutionView({
            accountId: scope.accountId,
            orgIdentifier: scope.orgIdentifier,
            projectIdentifier: scope.projectIdentifier,
            pipelineIdentifier: defaultTo(pipelineId, ''),
            executionIdentifier: defaultTo(lastPipelineExecutionName, ''),
            module: 'cd'
          })}>
          {lastPipelineExecutionName}
        </Link>
      ) : (
        <Text color={Color.GREY_900} font={{ variation: FontVariation.BODY }}>
          {lastPipelineExecutionName}
        </Text>
      )}
      <Text color={Color.GREY_500} font={{ variation: FontVariation.SMALL }}>
        {getString('id', { id: pipelineId })}
      </Text>
    </Layout.Vertical>
  )
}

export const LastModifiedCell: CellType = ({ row }) => {
  const { original } = row
  const { lastDeployedAt } = original
  return <TableCells.LastModifiedCell value={defaultTo(lastDeployedAt, '')} />
}
