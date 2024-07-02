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
import React from 'react'
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
  ViewColumns2,
  GithubCircle,
  GitLabFull,
  Code,
  Bitbucket as BitbucketIcon
} from 'iconoir-react'
import { Menu, MenuItem, PopoverInteractionKind, Position } from '@blueprintjs/core'
import { useHistory } from 'react-router-dom'
import { isNil } from 'lodash-es'
import { UseStringsReturn, useStrings } from 'framework/strings'
import { useAppContext } from 'AppContext'
import { getErrorMessage } from 'utils/Utils'
import { useConfirmAct } from 'hooks/useConfirmAction'
import VSCode from 'cde/icons/VSCode.svg?url'
import { GitspaceStatus } from 'cde/constants'
import {
  EnumGitspaceStateType,
  EnumIDEType,
  useDeleteGitspace,
  type TypesGitspaceConfig,
  type EnumCodeRepoType
} from 'cde-gitness/services'
import css from './ListGitspaces.module.scss'

enum CodeRepoType {
  Github = 'github',
  Gitlab = 'gitlab',
  HarnessCode = 'harness_code',
  Bitbucket = 'bitbucket',
  Unknown = 'unknown'
}

const getIconByRepoType = ({ repoType }: { repoType?: EnumCodeRepoType }): React.ReactNode => {
  switch (repoType) {
    case CodeRepoType.Github:
      return <GithubCircle height={40} />
    case CodeRepoType.Gitlab:
      return <GitLabFull height={40} />
    case CodeRepoType.Bitbucket:
      return <BitbucketIcon height={40} />
    default:
    case CodeRepoType.Unknown:
    case CodeRepoType.HarnessCode:
      return <Code height={40} />
  }
}

export const getStatusColor = (status?: EnumGitspaceStateType) => {
  switch (status) {
    case GitspaceStatus.RUNNING:
      return '#42AB45'
    case GitspaceStatus.STOPPED:
      return '#F3F3FA'
    case GitspaceStatus.ERROR:
      return '#FF0000'
    default:
      return '#000000'
  }
}

export const getStatusText = (getString: UseStringsReturn['getString'], status?: EnumGitspaceStateType) => {
  switch (status) {
    case GitspaceStatus.RUNNING:
      return getString('cde.listing.online')
    case GitspaceStatus.STOPPED:
      return getString('cde.listing.offline')
    case GitspaceStatus.ERROR:
      return getString('cde.listing.error')
    default:
      return getString('cde.listing.offline')
  }
}

enum IDEType {
  VSCODE = 'vs_code',
  VSCODEWEB = 'vs_code_web'
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

export const RenderGitspaceName: Renderer<CellProps<TypesGitspaceConfig>> = ({ row }) => {
  const details = row.original
  const { name } = details
  return (
    <Layout.Horizontal spacing={'small'} flex={{ alignItems: 'center', justifyContent: 'start' }}>
      <img src={VSCode} height={20} width={20} />
      <Text color={Color.BLACK} title={name} font={{ align: 'left', size: 'normal', weight: 'semi-bold' }}>
        {name}
      </Text>
    </Layout.Horizontal>
  )
}

export const RenderRepository: Renderer<CellProps<TypesGitspaceConfig>> = ({ row }) => {
  const { getString } = useStrings()
  const details = row.original
  const { name, branch, code_repo_url, code_repo_type, instance } = details || {}

  return (
    <Layout.Vertical>
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
      {(isNil(instance?.tracked_changes) || instance?.tracked_changes === '') && (
        <Text color={Color.GREY_300} font={{ align: 'left', size: 'small', weight: 'semi-bold' }}>
          {getString('cde.noChange')}
        </Text>
      )}
    </Layout.Vertical>
  )
}

export const RenderCPUUsage: Renderer<CellProps<TypesGitspaceConfig>> = ({ row }) => {
  const { getString } = useStrings()
  const instance = row.original.instance
  const { resource_usage, total_time_used } = instance || {}

  return getUsageTemplate(getString, <Cpu />, resource_usage, total_time_used)
}

export const RenderStorageUsage: Renderer<CellProps<TypesGitspaceConfig>> = ({ row }) => {
  const { getString } = useStrings()
  const instance = row.original.instance
  const { resource_usage, total_time_used } = instance || {}

  return getUsageTemplate(getString, <Db />, resource_usage, total_time_used)
}

export const RenderLastActivity: Renderer<CellProps<TypesGitspaceConfig>> = ({ row }) => {
  const { getString } = useStrings()
  const instance = row.original.instance
  const { last_used } = instance || {}
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

export const RenderGitspaceStatus: Renderer<CellProps<TypesGitspaceConfig>> = ({ row }) => {
  const { getString } = useStrings()
  const details = row.original
  const { instance, name } = details
  const { state } = instance || {}
  const color = getStatusColor(state)
  return (
    <Layout.Horizontal spacing={'small'} flex={{ alignItems: 'center', justifyContent: 'start' }}>
      <Circle height={10} width={10} color={color} fill={color} />
      <Text color={Color.BLACK} title={name} font={{ align: 'left', size: 'normal', weight: 'semi-bold' }}>
        {getStatusText(getString, state)}
      </Text>
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
  data: TypesGitspaceConfig
  refreshList: () => void
  handleStartStop?: () => Promise<void>
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
  const { instance, ide } = data
  const { id, state, url = ' ' } = instance || {}
  const history = useHistory()
  const { routes } = useAppContext()
  const pathparamsList = instance?.space_path?.split('/') || []
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
                space: instance?.space_path || '',
                gitspaceId: instance?.id || ''
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
                space: instance?.space_path || '',
                gitspaceId: instance?.id || ''
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
                await handleStartStop?.()
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
        {ide && state == GitspaceStatus.RUNNING && (
          <MenuItem
            onClick={e => {
              e.preventDefault()
              e.stopPropagation()
              if (ide === IDEType.VSCODE) {
                window.open(`vscode://harness-inc.gitspaces/${projectIdentifier}/${id}`, '_blank')
              } else {
                window.open(url, '_blank')
              }
            }}
            text={
              <Layout.Horizontal spacing="small">
                <OpenGitspaceButton ide={ide} />
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

interface RenderActionsProps extends CellProps<TypesGitspaceConfig> {
  refreshList: () => void
}

export const RenderActions = ({ row, refreshList }: RenderActionsProps) => {
  const { getString } = useStrings()
  const { showError, showSuccess } = useToaster()
  const details = row.original
  const { instance, name } = details
  const { mutate: deleteGitspace, loading: deleteLoading } = useDeleteGitspace({})

  // To be added in BE later.
  // const { mutate: actionGitspace, loading: actionLoading } = useGitspaceAction({
  //   accountIdentifier,
  //   projectIdentifier,
  //   orgIdentifier,
  //   gitspaceIdentifier: instance?.id || ''
  // })

  // const handleStartStop = async () => {
  //   return await actionGitspace({
  //     action: instance?.state === GitspaceStatus.RUNNING ? GitspaceActionType.STOP : GitspaceActionType.START
  //   })
  // }

  const confirmDelete = useConfirmAct()

  const handleDelete = async (e: React.MouseEvent<HTMLDivElement, MouseEvent>) => {
    confirmDelete({
      title: getString('cde.deleteGitspaceTitle'),
      message: getString('cde.deleteGitspaceText', { name: name }),
      action: async () => {
        try {
          e.preventDefault()
          e.stopPropagation()
          await deleteGitspace(instance?.id || '')
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
      icon={deleteLoading || false ? 'steps-spinner' : 'Options'}
      tooltip={
        <ActionMenu
          data={details}
          actionLoading={false}
          deleteLoading={deleteLoading}
          deleteGitspace={handleDelete}
          refreshList={refreshList}
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

export const ListGitspaces = ({ data, refreshList }: { data: TypesGitspaceConfig[]; refreshList: () => void }) => {
  const history = useHistory()
  const { getString } = useStrings()
  const { routes } = useAppContext()

  return (
    <Container>
      {data && (
        <TableV2<TypesGitspaceConfig>
          className={css.table}
          onRowClick={row => {
            const pathparamsList = row?.instance?.space_path?.split('/') || []
            const projectIdentifier = pathparamsList[pathparamsList.length - 1] || ''

            if (row?.instance?.state === GitspaceStatus.RUNNING) {
              if (row?.ide === IDEType.VSCODE) {
                window.open(`vscode://harness-inc.gitspaces/${projectIdentifier}/${row?.instance?.id}`, '_blank')
              } else {
                window.open(row?.instance.url, '_blank')
              }
            } else {
              history.push(
                routes.toCDEGitspaceDetail({
                  space: row?.instance?.space_path as string,
                  gitspaceId: row?.instance?.id as string
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
              id: 'status',
              Header: getString('cde.status'),
              Cell: RenderGitspaceStatus
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
