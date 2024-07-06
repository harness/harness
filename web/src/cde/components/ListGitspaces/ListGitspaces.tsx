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

import { Container, Layout, TableV2, Text, useToaster } from '@harnessio/uicore'
import React, { useEffect, useState } from 'react'
import { Color } from '@harnessio/design-system'
import type { Renderer, CellProps } from 'react-table'
import ReactTimeago from 'react-timeago'
import {
  Circle,
  GitBranch,
  Cpu,
  Clock,
  Play,
  Square,
  Db,
  ModernTv,
  OpenInBrowser,
  DeleteCircle,
  EditPencil,
  ViewColumns2
} from 'iconoir-react'
import { Menu, MenuItem, PopoverInteractionKind, Position } from '@blueprintjs/core'
import { useHistory } from 'react-router-dom'
import {
  useGitspaceAction,
  type EnumGitspaceStateType,
  type OpenapiGetGitspaceResponse,
  EnumIDEType,
  useDeleteGitspace,
  useGetToken
} from 'services/cde'
import { CDEPathParams, useGetCDEAPIParams } from 'cde/hooks/useGetCDEAPIParams'
import { GitspaceActionType, GitspaceStatus, IDEType } from 'cde/constants'
import { UseStringsReturn, useStrings } from 'framework/strings'
import { useAppContext } from 'AppContext'
import { getErrorMessage } from 'utils/Utils'
import { useConfirmAct } from 'hooks/useConfirmAction'
import VSCode from '../../icons/VSCode.svg?url'
import { getIconByRepoType } from '../CreateGitspace/components/SelectRepository/SelectRepository.utils'
import css from './ListGitspaces.module.scss'

export const getStatusColor = (status?: EnumGitspaceStateType) => {
  switch (status) {
    case GitspaceStatus.RUNNING:
      return '#00FF00'
    case GitspaceStatus.STOPPED:
    case GitspaceStatus.ERROR:
      return '#FF0000'
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
  const { config, state } = details
  const { name, id } = config || {}
  const color = getStatusColor(state)
  return (
    <Layout.Vertical spacing={'small'}>
      <Layout.Horizontal spacing={'small'} flex={{ alignItems: 'center', justifyContent: 'start' }}>
        <Circle height={10} width={10} color={color} fill={color} />
        <img src={VSCode} height={20} width={20} />
        <Text color={Color.BLACK} title={name} font={{ align: 'left', size: 'normal', weight: 'semi-bold' }}>
          {name}
        </Text>
      </Layout.Horizontal>
      <Text
        margin={{ left: 'large' }}
        color={Color.GREY_400}
        title={id}
        font={{ align: 'left', size: 'xsmall', weight: 'semi-bold' }}>
        {id}
      </Text>
    </Layout.Vertical>
  )
}

export const RenderRepository: Renderer<CellProps<OpenapiGetGitspaceResponse>> = ({ row }) => {
  const details = row.original
  const { config } = details
  const { name, branch, code_repo_url, code_repo_type } = config || {}

  return (
    <Layout.Vertical spacing={'small'}>
      <Layout.Horizontal
        spacing={'small'}
        className={css.repositoryCell}
        flex={{ alignItems: 'center', justifyContent: 'start' }}
        onClick={e => {
          e.preventDefault()
          e.stopPropagation()
          window.open(code_repo_url, '_blank')
        }}>
        {getIconByRepoType({ repoType: code_repo_type })}
        <Text className={css.gitspaceUrl} color={Color.PRIMARY_7} title={name} font={{ align: 'left', size: 'normal' }}>
          {name}
        </Text>
        <Text color={Color.PRIMARY_7}>:</Text>
        <GitBranch />
        <Text color={Color.PRIMARY_7} title={name} font={{ align: 'left', size: 'normal' }}>
          {branch}
        </Text>
      </Layout.Horizontal>
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

export const StartStopButton = ({ state, loading }: { state?: EnumGitspaceStateType; loading?: boolean }) => {
  const { getString } = useStrings()
  return (
    <Layout.Horizontal spacing="small" flex={{ alignItems: 'center', justifyContent: 'flex-start' }}>
      {loading ? <></> : state === GitspaceStatus.RUNNING ? <Square /> : <Play />}
      <Text icon={loading ? 'loading' : undefined}>
        {state === GitspaceStatus.RUNNING
          ? getString('cde.details.stopGitspace')
          : getString('cde.details.startGitspace')}
      </Text>
    </Layout.Horizontal>
  )
}

export const OpenGitspaceButton = ({ ide }: { ide?: EnumIDEType }) => {
  const { getString } = useStrings()

  return (
    <Layout.Horizontal spacing="small" flex={{ alignItems: 'center', justifyContent: 'flex-start' }}>
      {ide === IDEType.VSCODE ? <ModernTv /> : <OpenInBrowser />}
      <Text>{ide === IDEType.VSCODE ? getString('cde.ide.openVSCode') : getString('cde.ide.openBrowser')}</Text>
    </Layout.Horizontal>
  )
}

interface ActionMenuProps {
  data: OpenapiGetGitspaceResponse
  refreshList: () => void
  handleStartStop: () => Promise<void>
  loading?: boolean
  actionLoading?: boolean
  deleteLoading?: boolean
  deleteGitspace: (e: React.MouseEvent<HTMLDivElement, MouseEvent>) => Promise<void>
}

const ActionMenu = ({
  data,
  deleteGitspace,
  refreshList,
  handleStartStop,
  actionLoading,
  deleteLoading
}: ActionMenuProps) => {
  const { getString } = useStrings()
  const { showError } = useToaster()
  const { state, config, url = '' } = data
  const history = useHistory()
  const { routes } = useAppContext()
  const pathparamsList = config?.space_path?.split('/') || []
  const projectIdentifier = pathparamsList[pathparamsList.length - 1] || ''

  return (
    <Container
      className={css.listContainer}
      onClick={e => {
        e.preventDefault()
        e.stopPropagation()
      }}>
      <Menu>
        <MenuItem
          onClick={() => {
            history.push(
              routes.toCDEGitspaceDetail({
                space: config?.space_path || '',
                gitspaceId: config?.id || ''
              })
            )
          }}
          text={
            <Layout.Horizontal spacing="small" flex={{ alignItems: 'center', justifyContent: 'flex-start' }}>
              <ViewColumns2 />
              <Text>{getString('cde.viewGitspace')}</Text>
            </Layout.Horizontal>
          }
        />
        <MenuItem
          onClick={() => {
            history.push(
              routes.toCDEGitspacesEdit({
                space: config?.space_path || '',
                gitspaceId: config?.id || ''
              })
            )
          }}
          text={
            <Layout.Horizontal spacing="small" flex={{ alignItems: 'center', justifyContent: 'flex-start' }}>
              <EditPencil />
              <Text>{getString('cde.editGitspace')}</Text>
            </Layout.Horizontal>
          }
        />
        <MenuItem
          onClick={async e => {
            try {
              if (!actionLoading) {
                e.preventDefault()
                e.stopPropagation()
                await handleStartStop()
                await refreshList()
              }
            } catch (error) {
              showError(getErrorMessage(error))
            }
          }}
          text={
            <Layout.Horizontal spacing="small">
              <StartStopButton state={state} loading={actionLoading} />
            </Layout.Horizontal>
          }
        />
        {config?.ide && state == GitspaceStatus.RUNNING && (
          <MenuItem
            onClick={e => {
              e.preventDefault()
              e.stopPropagation()
              if (config?.ide === IDEType.VSCODE) {
                window.open(`vscode://harness-inc.gitspaces/${projectIdentifier}/${config?.id}`, '_blank')
              } else {
                window.open(url, '_blank')
              }
            }}
            text={
              <Layout.Horizontal spacing="small">
                <OpenGitspaceButton ide={config?.ide} />
              </Layout.Horizontal>
            }
          />
        )}
        <MenuItem
          onClick={deleteGitspace as Unknown as () => void}
          text={
            <Layout.Horizontal spacing="small" flex={{ alignItems: 'center', justifyContent: 'flex-start' }}>
              {deleteLoading ? <></> : <DeleteCircle />}
              <Text icon={deleteLoading ? 'loading' : undefined}>{getString('cde.deleteGitspace')}</Text>
            </Layout.Horizontal>
          }
        />
      </Menu>
    </Container>
  )
}

interface RenderActionsProps extends CellProps<OpenapiGetGitspaceResponse> {
  refreshList: () => void
}

export const RenderActions = ({ row, refreshList }: RenderActionsProps) => {
  const { getString } = useStrings()
  const { showError, showSuccess } = useToaster()
  const details = row.original
  const { config } = details
  const { projectIdentifier, orgIdentifier, accountIdentifier } = useGetCDEAPIParams() as CDEPathParams
  const { mutate: deleteGitspace, loading: deleteLoading } = useDeleteGitspace({
    projectIdentifier,
    orgIdentifier,
    accountIdentifier
  })
  const { mutate: actionGitspace, loading: actionLoading } = useGitspaceAction({
    accountIdentifier,
    projectIdentifier,
    orgIdentifier,
    gitspaceIdentifier: details?.config?.id || ''
  })

  const handleStartStop = async () => {
    return await actionGitspace({
      action: details?.state === GitspaceStatus.RUNNING ? GitspaceActionType.STOP : GitspaceActionType.START
    })
  }

  const confirmDelete = useConfirmAct()

  const handleDelete = async (e: React.MouseEvent<HTMLDivElement, MouseEvent>) => {
    confirmDelete({
      title: getString('cde.deleteGitspaceTitle'),
      message: getString('cde.deleteGitspaceText', { name: config?.name }),
      action: async () => {
        try {
          e.preventDefault()
          e.stopPropagation()
          await deleteGitspace(config?.id || '')
          showSuccess(getString('cde.deleteSuccess'))
          await refreshList()
        } catch (exception) {
          showError(getErrorMessage(exception))
        }
      }
    })
  }

  return (
    <Text
      onClick={e => {
        e.preventDefault()
        e.stopPropagation()
      }}
      style={{ cursor: 'pointer' }}
      icon={deleteLoading || actionLoading ? 'steps-spinner' : 'Options'}
      tooltip={
        <ActionMenu
          data={details}
          actionLoading={actionLoading}
          deleteLoading={deleteLoading}
          deleteGitspace={handleDelete}
          refreshList={refreshList}
          handleStartStop={handleStartStop}
        />
      }
      tooltipProps={{
        interactionKind: PopoverInteractionKind.HOVER,
        position: Position.BOTTOM_RIGHT,
        usePortal: true,
        popoverClassName: css.popover
      }}
    />
  )
}

export const ListGitspaces = ({
  data,
  refreshList
}: {
  data: OpenapiGetGitspaceResponse[]
  refreshList: () => void
}) => {
  const history = useHistory()
  const { getString } = useStrings()
  const { routes } = useAppContext()

  const { projectIdentifier, orgIdentifier, accountIdentifier } = useGetCDEAPIParams() as CDEPathParams

  const [selectedRowUrl, setSelectedRowUrl] = useState<string | undefined>('')

  const { data: tokenData, refetch } = useGetToken({
    accountIdentifier,
    projectIdentifier,
    orgIdentifier,
    gitspaceIdentifier: '',
    lazy: true
  })

  useEffect(() => {
    if (tokenData) {
      window.open(`${selectedRowUrl}&token=${tokenData?.gitspace_token}`, '_blank')
    }
  }, [tokenData])

  return (
    <Container>
      {data && (
        <TableV2<OpenapiGetGitspaceResponse>
          className={css.table}
          onRowClick={row => {
            if (row?.state === GitspaceStatus.RUNNING) {
              if (row?.config?.ide === IDEType.VSCODE) {
                window.open(`vscode://harness-inc.gitspaces/${projectIdentifier}/${row?.config?.id}`, '_blank')
              } else {
                setSelectedRowUrl(row.url)
                refetch({
                  pathParams: {
                    accountIdentifier,
                    projectIdentifier,
                    orgIdentifier,
                    gitspaceIdentifier: row.config?.id
                  }
                })
              }
            } else {
              history.push(
                routes.toCDEGitspaceDetail({
                  space: row?.config?.space_path as string,
                  gitspaceId: row?.config?.id as string
                })
              )
            }
          }}
          columns={[
            {
              id: 'gitspaces',
              Header: getString('cde.gitspaces'),
              Cell: RenderGitspaceName
            },
            {
              id: 'repository',
              Header: getString('cde.repositoryAndBranch'),
              Cell: RenderRepository
            },
            {
              id: 'lastactivity',
              Header: getString('cde.sessionDuration'),
              Cell: RenderLastActivity
            },
            {
              id: 'action',
              Cell: (props: RenderActionsProps) => <RenderActions {...props} refreshList={refreshList} />
            }
          ]}
          data={data}
        />
      )}
    </Container>
  )
}
