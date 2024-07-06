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

import React, { useEffect, useState } from 'react'
import { Text, Layout, Container, Button, ButtonVariation, PageError, useToaster } from '@harnessio/uicore'
import { FontVariation } from '@harnessio/design-system'
import type { PopoverProps } from '@harnessio/uicore/dist/components/Popover/Popover'
import { Menu, MenuItem, PopoverInteractionKind, PopoverPosition } from '@blueprintjs/core'
import { useHistory, useParams } from 'react-router-dom'
import { Cpu, Circle, GitFork, Repository, EditPencil, DeleteCircle } from 'iconoir-react'
import { isUndefined } from 'lodash-es'
import type { GetDataError, MutateMethod, UseGetProps } from 'restful-react'
import {
  useGetGitspaceEvents,
  type EnumGitspaceStateType,
  type GitspaceActionPathParams,
  type OpenapiGetGitspaceResponse,
  type OpenapiGitspaceActionRequest,
  useDeleteGitspace,
  useGetToken
} from 'services/cde'
import { useStrings } from 'framework/strings'
import { GitspaceActionType, GitspaceStatus, IDEType } from 'cde/constants'
import { getErrorMessage } from 'utils/Utils'
import { useAppContext } from 'AppContext'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { CDEPathParams, useGetCDEAPIParams } from 'cde/hooks/useGetCDEAPIParams'
import { useConfirmAct } from 'hooks/useConfirmAction'
import { useQueryParams } from 'hooks/useQueryParams'
import Gitspace from '../../icons/Gitspace.svg?url'
import { StartStopButton, getStatusColor } from '../ListGitspaces/ListGitspaces'
import { usePolling } from './usePolling'
import EventsTimeline from './EventsTimeline/EventsTimeline'
import { GitspaceEventType, pollEventsList } from './GitspaceDetails.constants'
import { getGitspaceStatusLabel } from './GitspaceDetails.utils'
import css from './GitspaceDetails.module.scss'

interface QueryGitspace {
  gitspaceId?: string
}

export const GitspaceDetails = ({
  data,
  error,
  loading,
  refetch,
  mutate,
  refetchLogs,
  mutateLoading
}: {
  data?: OpenapiGetGitspaceResponse | null
  error?: GetDataError<unknown> | null
  loading?: boolean
  mutateLoading?: boolean
  refetch: (
    options?: Partial<Omit<UseGetProps<OpenapiGetGitspaceResponse, unknown, void, unknown>, 'lazy'>> | undefined
  ) => Promise<void>
  refetchLogs?: () => Promise<void>
  mutate: MutateMethod<void, OpenapiGitspaceActionRequest, void, GitspaceActionPathParams>
  actionError?: GetDataError<unknown> | null
}) => {
  const { getString } = useStrings()
  const { gitspaceId = '' } = useParams<QueryGitspace>()
  const { routes } = useAppContext()
  const space = useGetSpaceParam()
  const { showError, showSuccess } = useToaster()
  const { openvscodeweb = '' } = useQueryParams<{ openvscodeweb?: string }>()
  const history = useHistory()
  const { projectIdentifier, orgIdentifier, accountIdentifier } = useGetCDEAPIParams() as CDEPathParams

  const [startPolling, setstartPolling] = useState<GitspaceActionType | undefined>(undefined)

  const { config, state, url } = data || {}

  const openEditorLabel =
    config?.ide === IDEType.VSCODE ? getString('cde.details.openEditor') : getString('cde.details.openBrowser')

  const color = getStatusColor(state as EnumGitspaceStateType)

  const { data: tokenData, refetch: refetchToken } = useGetToken({
    accountIdentifier,
    projectIdentifier,
    orgIdentifier,
    gitspaceIdentifier: '',
    lazy: true
  })

  useEffect(() => {
    if (tokenData) {
      window.open(`${url}&token=${tokenData?.gitspace_token}`, openvscodeweb ? '_self' : '_blank')
    }
  }, [tokenData])

  useEffect(() => {
    if (openvscodeweb === 'true' && url) {
      refetchToken({
        pathParams: {
          accountIdentifier,
          projectIdentifier,
          orgIdentifier,
          gitspaceIdentifier: gitspaceId
        }
      })
    }
  }, [url])

  const {
    data: eventData,
    refetch: refetchEventData,
    loading: loadingEvents
  } = useGetGitspaceEvents({
    accountIdentifier,
    orgIdentifier,
    projectIdentifier,
    gitspaceIdentifier: gitspaceId
  })

  const pollingCondition = pollEventsList.includes(
    (eventData?.[eventData?.length - 1]?.event || '') as unknown as GitspaceEventType
  )

  usePolling(refetchEventData, {
    pollingInterval: 10000,
    startCondition: Boolean(startPolling) || !pollingCondition
  })

  usePolling(refetchLogs!, {
    pollingInterval: 10000,
    startCondition:
      (eventData?.[eventData?.length - 1]?.event as string) === GitspaceEventType.AgentGitspaceCreationStart,
    stopCondition: pollingCondition
  })

  const { mutate: deleteGitspace, loading: deleteLoading } = useDeleteGitspace({
    projectIdentifier,
    orgIdentifier,
    accountIdentifier
  })

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
          history.push(routes.toCDEGitspaces({ space }))
        } catch (exception) {
          showError(getErrorMessage(exception))
        }
      }
    })
  }

  return (
    <Layout.Vertical width={'30%'} spacing="large">
      <Layout.Vertical spacing="medium">
        <img src={Gitspace} width={42} height={42}></img>
        {error ? (
          <PageError onClick={() => refetch()} message={getErrorMessage(error)} />
        ) : (
          <Text
            icon={loadingEvents || loading || mutateLoading ? 'loading' : undefined}
            className={css.subText}
            font={{ variation: FontVariation.CARD_TITLE }}>
            {getGitspaceStatusLabel({
              isPolling: !pollingCondition,
              mutateLoading,
              startPolling,
              state,
              getString,
              eventData
            })}
          </Text>
        )}
        <EventsTimeline events={eventData} />
      </Layout.Vertical>
      <Container className={css.detailsBar}>
        {error ? (
          <Text>{getErrorMessage(error)}</Text>
        ) : (
          <>
            {isUndefined(config) ? (
              <Text>{getString('cde.details.noData')}</Text>
            ) : (
              <Layout.Horizontal flex={{ justifyContent: 'space-between' }}>
                <Layout.Vertical spacing="small">
                  <Layout.Horizontal spacing="small">
                    <Circle color={color} fill={color} />
                    <Text font={'small'}>{config?.name}</Text>
                  </Layout.Horizontal>
                  <Layout.Horizontal
                    className={css.repository}
                    spacing="small"
                    onClick={() => window.open(config?.code_repo_url, '_blank')}>
                    <Repository />
                    <Text width={100} lineClamp={1} font={'small'}>
                      {config?.code_repo_id || config?.code_repo_url}
                    </Text>
                    <Text> / </Text>
                    <GitFork />
                    <Text font={'small'}>{config?.branch}</Text>
                  </Layout.Horizontal>
                </Layout.Vertical>
                <Layout.Vertical spacing="small">
                  <Layout.Horizontal spacing="small">
                    <Cpu />
                    <Text font={'small'}>{config?.infra_provider_resource_id}</Text>
                  </Layout.Horizontal>
                </Layout.Vertical>
              </Layout.Horizontal>
            )}
          </>
        )}
      </Container>

      <Layout.Horizontal spacing={'medium'}>
        {(state === GitspaceStatus.STOPPED || state === GitspaceStatus.ERROR) && (
          <Button
            disabled={mutateLoading}
            variation={ButtonVariation.PRIMARY}
            onClick={() => {
              try {
                setstartPolling(GitspaceActionType.START)
                mutate({ action: GitspaceActionType.START }).then(() => {
                  setstartPolling(undefined)
                })
              } catch (err) {
                showError(getErrorMessage(err))
                setstartPolling(undefined)
              }
            }}>
            {getString('cde.details.startGitspace')}
          </Button>
        )}

        {state === GitspaceStatus.RUNNING && (
          <>
            <Button
              onClick={() => {
                if (config?.ide === IDEType.VSCODE) {
                  window.open(`vscode://harness-inc.gitspaces/${projectIdentifier}/${gitspaceId}`, '_blank')
                } else {
                  refetchToken({
                    pathParams: {
                      accountIdentifier,
                      projectIdentifier,
                      orgIdentifier,
                      gitspaceIdentifier: gitspaceId
                    }
                  })
                }
              }}
              variation={ButtonVariation.PRIMARY}>
              {openEditorLabel}
            </Button>
          </>
        )}

        <Button
          iconProps={mutateLoading || deleteLoading ? { name: 'loading' } : {}}
          disabled={mutateLoading || deleteLoading}
          variation={ButtonVariation.TERTIARY}
          rightIcon="chevron-down"
          tooltipProps={
            {
              interactionKind: PopoverInteractionKind.HOVER,
              position: PopoverPosition.BOTTOM_LEFT,
              popoverClassName: css.popover
            } as PopoverProps
          }
          tooltip={
            <Menu>
              <MenuItem
                text={
                  <Layout.Horizontal spacing="small">
                    <StartStopButton state={state} loading={mutateLoading} />
                  </Layout.Horizontal>
                }
                onClick={() => {
                  const actiontype =
                    state === GitspaceStatus.RUNNING ? GitspaceActionType.STOP : GitspaceActionType.START
                  try {
                    setstartPolling(actiontype)
                    mutate({ action: actiontype }).then(() => {
                      setstartPolling(undefined)
                      refetch().then(() => {
                        refetchLogs?.()
                      })
                    })
                  } catch (err) {
                    showError(getErrorMessage(err))
                    setstartPolling(undefined)
                  }
                }}
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
                text={<Text icon={'list-view'}>{getString('cde.details.goToDashboard')}</Text>}
                onClick={() => {
                  history.push(routes.toCDEGitspaces({ space }))
                }}
              />
              <MenuItem
                onClick={handleDelete as Unknown as () => void}
                text={
                  <Layout.Horizontal spacing="small" flex={{ alignItems: 'center', justifyContent: 'flex-start' }}>
                    {deleteLoading ? <></> : <DeleteCircle />}
                    <Text icon={deleteLoading ? 'loading' : undefined}>{getString('cde.deleteGitspace')}</Text>
                  </Layout.Horizontal>
                }
              />
            </Menu>
          }>
          {getString('cde.details.actions')}
        </Button>
      </Layout.Horizontal>
    </Layout.Vertical>
  )
}
