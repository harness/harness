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

import React, { useState } from 'react'
import { Text, Layout, Container, Button, ButtonVariation, PageError, useToaster } from '@harnessio/uicore'
import { FontVariation } from '@harnessio/design-system'
import type { PopoverProps } from '@harnessio/uicore/dist/components/Popover/Popover'
import { Menu, MenuItem, PopoverPosition } from '@blueprintjs/core'
import { useHistory, useParams } from 'react-router-dom'
import { Cpu, Circle, GitFork, Repository } from 'iconoir-react'
import { isUndefined } from 'lodash-es'
import type { GetDataError, MutateMethod, UseGetProps } from 'restful-react'
import {
  useGetGitspaceEvents,
  type EnumGitspaceStateType,
  type GitspaceActionPathParams,
  type OpenapiGetGitspaceResponse,
  type OpenapiGitspaceActionRequest
} from 'services/cde'
import { UseStringsReturn, useStrings } from 'framework/strings'
import { GitspaceActionType, GitspaceStatus, IDEType } from 'cde/constants'
import { getErrorMessage } from 'utils/Utils'
import { useAppContext } from 'AppContext'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { useQueryParams } from 'hooks/useQueryParams'
import { CDEPathParams, useGetCDEAPIParams } from 'cde/hooks/useGetCDEAPIParams'
import Gitspace from '../../icons/Gitspace.svg?url'
import { getStatusColor } from '../ListGitspaces/ListGitspaces'
import { usePolling } from './usePolling'
import EventsTimeline from './EventsTimeline/EventsTimeline'
import { GitspaceEventType, pollEventsList } from './GitspaceDetails.constants'
import css from './GitspaceDetails.module.scss'

interface QueryGitspace {
  gitspaceId?: string
}

export const getGitspaceDetailTitle = ({
  getString,
  state,
  loading,
  redirectFrom,
  actionError
}: {
  getString: UseStringsReturn['getString']
  state?: EnumGitspaceStateType
  loading?: boolean
  redirectFrom?: string
  actionError?: GetDataError<unknown> | null
}) => {
  if (loading) {
    return getString('cde.details.fetchingGitspace')
  } else if (state === GitspaceStatus.UNKNOWN || (state === GitspaceStatus.STOPPED && !!redirectFrom && !actionError)) {
    return getString('cde.details.provisioningGitspace')
  } else if (state === GitspaceStatus.STOPPED) {
    return getString('cde.details.gitspaceStopped')
  } else if (state === GitspaceStatus.RUNNING) {
    return getString('cde.details.gitspaceRunning')
  } else if (!loading && isUndefined(state)) {
    getString('cde.details.noData')
  }
}

export const GitspaceDetails = ({
  data,
  error,
  loading,
  refetch,
  mutate,
  refetchLogs,
  mutateLoading,
  isfetchingInProgress,
  actionError
}: {
  data?: OpenapiGetGitspaceResponse | null
  error?: GetDataError<unknown> | null
  loading?: boolean
  mutateLoading?: boolean
  isfetchingInProgress?: boolean
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
  const { showError } = useToaster()
  const history = useHistory()
  const { projectIdentifier, orgIdentifier, accountIdentifier } = useGetCDEAPIParams() as CDEPathParams
  const { redirectFrom = '' } = useQueryParams<{ redirectFrom?: string }>()

  const [startPolling, setstartPolling] = useState(false)

  const { config, state, url } = data || {}

  const openEditorLabel =
    config?.ide === IDEType.VSCODE ? getString('cde.details.openEditor') : getString('cde.details.openBrowser')

  const color = getStatusColor(state as EnumGitspaceStateType)

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

  const isPolling = usePolling(refetchEventData, {
    pollingInterval: 10000,
    startCondition: startPolling || !pollingCondition
  })

  usePolling(refetchLogs!, {
    pollingInterval: 10000,
    startCondition:
      (eventData?.[eventData?.length - 1]?.event as string) === GitspaceEventType.AgentGitspaceCreationStart,
    stopCondition: pollingCondition
  })

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
            {redirectFrom || isPolling || startPolling
              ? eventData?.[eventData?.length - 1]?.message || 'Fetching events'
              : getGitspaceDetailTitle({ getString, state, loading, redirectFrom, actionError })}
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
                  <Layout.Horizontal spacing="small">
                    <Repository />
                    <Text width={100} lineClamp={1} font={'small'}>
                      {config?.id}
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
        {state === GitspaceStatus.UNKNOWN && (
          <>
            <Button
              onClick={() => {
                if (config?.ide === IDEType.VSCODE) {
                  window.open(`vscode://harness-inc.gitspaces/${projectIdentifier}/${gitspaceId}`, '_blank')
                } else {
                  window.open(data?.url || '', '_blank')
                }
              }}
              variation={ButtonVariation.SECONDARY}
              disabled>
              {openEditorLabel}
            </Button>
            <Button
              variation={ButtonVariation.PRIMARY}
              onClick={() => {
                try {
                  setstartPolling(true)
                  mutate({ action: GitspaceActionType.STOP }).then(() => {
                    setstartPolling(false)
                  })
                } catch (err) {
                  setstartPolling(false)
                  showError(getErrorMessage(err))
                }
              }}>
              {getString('cde.details.stopProvising')}
            </Button>
          </>
        )}

        {state === GitspaceStatus.STOPPED && (
          <>
            <Button
              disabled={isfetchingInProgress}
              variation={ButtonVariation.PRIMARY}
              onClick={() => {
                try {
                  setstartPolling(true)
                  mutate({ action: GitspaceActionType.START }).then(() => {
                    setstartPolling(false)
                  })
                } catch (err) {
                  showError(getErrorMessage(err))
                  setstartPolling(false)
                }
              }}>
              {getString('cde.details.startGitspace')}
            </Button>
            <Button
              onClick={() => {
                if (gitspaceId) {
                  history.push(routes.toCDEGitspaces({ space }))
                }
              }}
              variation={ButtonVariation.TERTIARY}>
              {getString('cde.details.goToDashboard')}
            </Button>
          </>
        )}

        {state === GitspaceStatus.RUNNING && (
          <>
            <Button
              onClick={() => {
                if (config?.ide === IDEType.VSCODE) {
                  window.open(`vscode://harness-inc.gitspaces/${projectIdentifier}/${gitspaceId}`, '_blank')
                } else {
                  window.open(url, '_blank')
                }
              }}
              variation={ButtonVariation.PRIMARY}>
              {openEditorLabel}
            </Button>
            <Button
              iconProps={mutateLoading ? { name: 'loading' } : {}}
              disabled={mutateLoading}
              variation={ButtonVariation.TERTIARY}
              rightIcon="chevron-down"
              tooltipProps={
                {
                  interactionKind: 'click',
                  position: PopoverPosition.BOTTOM_LEFT,
                  popoverClassName: css.popover
                } as PopoverProps
              }
              tooltip={
                <Menu>
                  <MenuItem
                    text={getString('cde.details.stopGitspace')}
                    onClick={() => {
                      try {
                        setstartPolling(true)
                        mutate({ action: GitspaceActionType.STOP }).then(() => {
                          setstartPolling(false)
                          refetch().then(() => {
                            refetchLogs?.()
                          })
                        })
                      } catch (err) {
                        showError(getErrorMessage(err))
                        setstartPolling(false)
                      }
                    }}
                  />
                  <MenuItem
                    text={getString('cde.details.goToDashboard')}
                    onClick={() => {
                      history.push(routes.toCDEGitspaces({ space }))
                    }}
                  />
                </Menu>
              }>
              {getString('cde.details.actions')}
            </Button>
          </>
        )}
      </Layout.Horizontal>
    </Layout.Vertical>
  )
}
