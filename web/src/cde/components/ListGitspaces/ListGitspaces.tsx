/*
 * Copyright 2023 Harness, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { Container, Layout, TableV2, Text } from '@harnessio/uicore'
import React from 'react'
import { Color } from '@harnessio/design-system'
import type { Renderer, CellProps } from 'react-table'
import ReactTimeago from 'react-timeago'
import { Circle, GithubCircle, GitBranch, Cpu, Clock, Play, Square, Db, ModernTv, OpenInBrowser } from 'iconoir-react'
import { Menu, MenuItem, PopoverInteractionKind, Position } from '@blueprintjs/core'
import {
  useGitspaceAction,
  type EnumGitspaceStateType,
  type OpenapiGetGitspaceResponse,
  EnumIDEType
} from 'services/cde'
import { CDEPathParams, useGetCDEAPIParams } from 'cde/hooks/useGetCDEAPIParams'
import { GitspaceActionType, GitspaceStatus, IDEType } from 'cde/constants'
import { UseStringsReturn, useStrings } from 'framework/strings'
import VSCode from '../../icons/VSCode.svg?url'
import css from './ListGitspaces.module.scss'

export const getStatusColor = (status?: EnumGitspaceStateType) => {
  switch (status) {
    case GitspaceStatus.RUNNING:
      return '#00FF00'
    case GitspaceStatus.STOPPED:
    case GitspaceStatus.ERROR:
      return '#FF0000'
    case GitspaceStatus.UNKNOWN:
      return '#808080'
    default:
      return '#000000'
  }
}

const getUsageTemplate = (
  getString: UseStringsReturn['getString'],
  icon: React.ReactNode,
  resource_usage?: string,
  total_time_used?: number
): React.ReactElement | null => {
  return (
    <Layout.Horizontal spacing={'small'} flex={{ alignItems: 'center', justifyContent: 'start' }}>
      {icon}
      <Text color={Color.GREY_500} font={{ align: 'left', size: 'normal' }}>
        {getString('cde.used')} {resource_usage || 0}
      </Text>
      <Text>/</Text>
      <Text color={Color.GREY_500} font={{ align: 'left', size: 'normal' }}>
        {total_time_used || 0} {getString('cde.hours')}
      </Text>
    </Layout.Horizontal>
  )
}

export const RenderGitspaceName: Renderer<CellProps<OpenapiGetGitspaceResponse>> = ({ row }) => {
  const details = row.original
  const { config, status } = details
  const { name } = config || {}
  const color = getStatusColor(status)
  return (
    <Layout.Horizontal spacing={'small'} flex={{ alignItems: 'center', justifyContent: 'start' }}>
      <Circle height={10} width={10} color={color} fill={color} />
      <img src={VSCode} height={20} width={20} />
      <Text color={Color.BLACK} title={name} font={{ align: 'left', size: 'normal', weight: 'semi-bold' }}>
        {name}
      </Text>
    </Layout.Horizontal>
  )
}

export const RenderRepository: Renderer<CellProps<OpenapiGetGitspaceResponse>> = ({ row }) => {
  const { getString } = useStrings()
  const details = row.original
  const { config, tracked_changes } = details
  const { name, branch } = config || {}

  return (
    <Layout.Vertical spacing={'small'}>
      <Layout.Horizontal spacing={'small'} flex={{ alignItems: 'center', justifyContent: 'start' }}>
        <GithubCircle />
        <Text color={Color.GREY_500} title={name} font={{ align: 'left', size: 'normal' }}>
          {name}
        </Text>
        <Text>:</Text>
        <GitBranch />
        <Text color={Color.GREY_500} title={name} font={{ align: 'left', size: 'normal' }}>
          {branch}
        </Text>
      </Layout.Horizontal>
      <Text color={Color.GREY_500} font={{ align: 'left', size: 'small' }}>
        {tracked_changes || getString('cde.noChange')}
      </Text>
    </Layout.Vertical>
  )
}

export const RenderCPUUsage: Renderer<CellProps<OpenapiGetGitspaceResponse>> = ({ row }) => {
  const { getString } = useStrings()
  const details = row.original
  const { resource_usage, total_time_used } = details

  return getUsageTemplate(getString, <Cpu />, resource_usage, total_time_used)
}

export const RenderStorageUsage: Renderer<CellProps<OpenapiGetGitspaceResponse>> = ({ row }) => {
  const { getString } = useStrings()
  const details = row.original
  const { resource_usage, total_time_used } = details

  return getUsageTemplate(getString, <Db />, resource_usage, total_time_used)
}

export const RenderLastActivity: Renderer<CellProps<OpenapiGetGitspaceResponse>> = ({ row }) => {
  const { getString } = useStrings()
  const details = row.original
  const { last_used } = details
  return (
    <Layout.Horizontal spacing={'small'} flex={{ alignItems: 'center', justifyContent: 'start' }}>
      <Clock />
      {last_used ? (
        <ReactTimeago date={last_used} />
      ) : (
        <Text color={Color.GREY_500} font={{ align: 'left', size: 'normal' }}>
          {getString('cde.na')}
        </Text>
      )}
    </Layout.Horizontal>
  )
}

const StartStopButton = ({
  gitspaceIdentifier,
  status
}: {
  gitspaceIdentifier: string
  status?: EnumGitspaceStateType
}) => {
  const { getString } = useStrings()
  const { accountIdentifier, projectIdentifier, orgIdentifier } = useGetCDEAPIParams() as CDEPathParams
  const { mutate } = useGitspaceAction({ accountIdentifier, projectIdentifier, orgIdentifier, gitspaceIdentifier })

  const handleClick = () => {
    mutate({ action: status === GitspaceStatus.RUNNING ? GitspaceActionType.STOP : GitspaceActionType.START })
  }

  return (
    <Layout.Horizontal
      onClick={handleClick}
      spacing="small"
      flex={{ alignItems: 'center', justifyContent: 'flex-start' }}>
      {status === GitspaceStatus.RUNNING ? <Square /> : <Play />}
      <Text>
        {status === GitspaceStatus.RUNNING
          ? getString('cde.details.stopGitspace')
          : getString('cde.details.startGitspace')}
      </Text>
    </Layout.Horizontal>
  )
}

const OpenGitspaceButton = ({ ide, url }: { ide?: EnumIDEType; url: string }) => {
  const { getString } = useStrings()
  const handleClick = () => {
    window.open(url, '_blank')
  }

  return (
    <Layout.Horizontal
      onClick={handleClick}
      spacing="small"
      flex={{ alignItems: 'center', justifyContent: 'flex-start' }}>
      {ide === IDEType.VSCODE ? <ModernTv /> : <OpenInBrowser />}
      <Text>{ide === IDEType.VSCODE ? getString('cde.ide.openVSCode') : getString('cde.ide.openBrowser')}</Text>
    </Layout.Horizontal>
  )
}

const ActionMenu = ({ data }: { data: OpenapiGetGitspaceResponse }) => {
  const { status, id, config, url = '' } = data
  return (
    <Container className={css.listContainer}>
      <Menu>
        <MenuItem
          text={
            <Layout.Horizontal spacing="small">
              <StartStopButton status={status} gitspaceIdentifier={id || ''} />
            </Layout.Horizontal>
          }
        />
        {config?.ide && (
          <MenuItem
            text={
              <Layout.Horizontal spacing="small">
                <OpenGitspaceButton ide={config?.ide} url={url} />
              </Layout.Horizontal>
            }
          />
        )}
      </Menu>
    </Container>
  )
}

export const RenderActions: Renderer<CellProps<OpenapiGetGitspaceResponse>> = ({ row }) => {
  const details = row.original
  return (
    <Text
      style={{ cursor: 'pointer' }}
      icon={'Options'}
      tooltip={<ActionMenu data={details} />}
      tooltipProps={{
        interactionKind: PopoverInteractionKind.HOVER,
        position: Position.BOTTOM_RIGHT,
        usePortal: false,
        popoverClassName: css.popover
      }}
    />
  )
}

export const ListGitspaces = ({ data }: { data: OpenapiGetGitspaceResponse[] }) => {
  return (
    <Container>
      {data && (
        <TableV2<OpenapiGetGitspaceResponse>
          className={css.table}
          columns={[
            {
              id: 'gitspaces',
              Header: 'Gitspaces',
              Cell: RenderGitspaceName
            },
            {
              id: 'repository',
              Header: 'REPOSITORY & BRANCH',
              Cell: RenderRepository
            },
            {
              id: 'cpuusage',
              Header: 'CPU Usage',
              Cell: RenderCPUUsage
            },
            {
              id: 'storageusage',
              Header: 'Storage Usage',
              Cell: RenderStorageUsage
            },
            {
              id: 'lastactivity',
              Header: 'Last Active',
              Cell: RenderLastActivity
            },
            {
              id: 'action',
              Cell: RenderActions
            }
          ]}
          data={data}
        />
      )}
    </Container>
  )
}
