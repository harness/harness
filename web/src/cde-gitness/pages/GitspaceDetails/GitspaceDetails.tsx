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

import React, { useEffect, useRef, useState } from 'react'
import {
  Breadcrumbs,
  Button,
  ButtonVariation,
  Card,
  Container,
  Layout,
  Accordion,
  Page,
  Text,
  useToaster,
  AccordionHandle
} from '@harnessio/uicore'
import { Play } from 'iconoir-react'
import { useHistory, useParams } from 'react-router-dom'
import { Color, FontVariation, PopoverProps } from '@harnessio/design-system'
import { Menu, MenuItem, PopoverInteractionKind, PopoverPosition } from '@blueprintjs/core'
import { defaultTo } from 'lodash-es'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { useAppContext } from 'AppContext'
import { useStrings } from 'framework/strings'
import EventTimelineAccordion from 'cde-gitness/components/EventTimelineAccordion/EventTimelineAccordion'
import { DetailsCard } from 'cde-gitness/components/DetailsCard/DetailsCard'
import type { EnumGitspaceStateType, TypesGitspaceEventResponse } from 'cde-gitness/services'
import { GitspaceActionType, GitspaceStatus } from 'cde-gitness/constants'
import { useQueryParams } from 'hooks/useQueryParams'
import { useUpdateQueryParams } from 'hooks/useUpdateQueryParams'
import { getErrorMessage } from 'utils/Utils'
import { usePolling } from 'cde-gitness/hooks/usePolling'
import deleteIcon from 'cde-gitness/assests/delete.svg?url'
import vscodeIcon from 'cde-gitness/assests/VSCode.svg?url'
import vsCodeWebIcon from 'cde-gitness/assests/vsCodeWeb.svg?url'
import pauseIcon from 'cde-gitness/assests/pause.svg?url'
import { IDEType } from 'cde-gitness/constants'
import homeIcon from 'cde-gitness/assests/home.svg?url'
import { useConfirmAct } from 'hooks/useConfirmAction'
import { useGitspaceDetails } from 'cde-gitness/hooks/useGitspaceDetails'
import { useGitspaceEvents } from 'cde-gitness/hooks/useGitspaceEvents'
import { useGitspaceActions } from 'cde-gitness/hooks/useGitspaceActions'
import { useDeleteGitspaces } from 'cde-gitness/hooks/useDeleteGitspaces'
import { useGitspacesLogs } from 'cde-gitness/hooks/useGitspaceLogs'
import { useOpenVSCodeBrowserURL } from 'cde-gitness/hooks/useOpenVSCodeBrowserURL'
import ContainerLogs from '../../components/ContainerLogs/ContainerLogs'
import { useGetLogStream } from '../../hooks/useGetLogStream'
import css from './GitspaceDetails.module.scss'

const GitspaceDetails = () => {
  const space = useGetSpaceParam()
  const { getString } = useStrings()
  const { routes, standalone } = useAppContext()
  const { showError, showSuccess } = useToaster()
  const history = useHistory()
  const [startTriggred, setStartTriggred] = useState<boolean>(false)
  const [triggerPollingOnStart, setTriggerPollingOnStart] = useState<EnumGitspaceStateType>()
  const { gitspaceId = '' } = useParams<{ gitspaceId?: string }>()

  const [isStreamingLogs, setIsStreamingLogs] = useState(false)

  const [startPolling, setStartPolling] = useState<GitspaceActionType | undefined>(undefined)

  const { loading, data, refetch, error } = useGitspaceDetails({ gitspaceId })

  const { data: eventData, refetch: refetchEventData } = useGitspaceEvents({ gitspaceId })

  const {
    data: responseData,
    refetch: refetchLogsData,
    response,
    error: streamLogsError,
    loading: logsLoading
  } = useGitspacesLogs({ gitspaceId })

  useEffect(() => {
    if (streamLogsError?.message) {
      showError(streamLogsError.message)
    }
  }, [streamLogsError?.message])

  const { mutate: actionMutate, loading: mutateLoading } = useGitspaceActions({ gitspaceId })

  const { mutate: deleteGitspace, loading: deleteLoading } = useDeleteGitspaces({ gitspaceId })

  const { updateQueryParams } = useUpdateQueryParams<{ redirectFrom?: string }>()
  const { redirectFrom = '' } = useQueryParams<{ redirectFrom?: string }>()

  const pollingCondition = triggerPollingOnStart
    ? false
    : [GitspaceStatus.RUNNING, GitspaceStatus.STOPPED, GitspaceStatus.ERROR, GitspaceStatus.UNINITIALIZED].includes(
        data?.state as GitspaceStatus
      )

  const disabledActionButtons = [GitspaceStatus.STARTING, GitspaceStatus.STOPPING].includes(
    data?.state as GitspaceStatus
  )

  useEffect(() => {
    const filteredEvent = (eventData as Unknown)?.filter(
      (item: Unknown) =>
        (item.event === 'agent_gitspace_creation_start' || item.event === 'agent_gitspace_deletion_start') &&
        defaultTo(item?.timestamp, 0) >= defaultTo(data?.instance?.updated, 0)
    )
    if (disabledActionButtons && filteredEvent?.length && !isStreamingLogs) {
      refetchLogsData()
      setIsStreamingLogs(true)
    } else if (
      (filteredEvent?.length && !disabledActionButtons && isStreamingLogs) ||
      (isStreamingLogs && streamLogsError)
    ) {
      setIsStreamingLogs(false)
    }
  }, [eventData, data?.instance?.updated, disabledActionButtons, streamLogsError])

  usePolling(
    async () => {
      await refetchEventData()
      await refetch()
      if (triggerPollingOnStart) {
        setTriggerPollingOnStart(undefined)
      }
    },
    {
      pollingInterval: standalone ? 2000 : 10000,
      startCondition: Boolean(startPolling) || !pollingCondition
    }
  )

  usePolling(
    async () => {
      if (!standalone) {
        await refetchLogsData()
      }
    },
    {
      pollingInterval: 10000,
      startCondition: (eventData?.[eventData?.length - 1]?.event as string) === 'agent_gitspace_creation_start',
      stopCondition: pollingCondition
    }
  )

  useEffect(() => {
    const startTrigger = async () => {
      if (redirectFrom && !startTriggred && !mutateLoading) {
        try {
          setStartTriggred(true)
          const resp = await actionMutate({ action: GitspaceActionType.START })
          if (resp?.state === GitspaceStatus.STARTING) {
            setTriggerPollingOnStart(resp.state)
          }
          await refetchEventData()
          await refetch()
          updateQueryParams({ redirectFrom: undefined })
        } catch (err) {
          showError(getErrorMessage(err))
        }
      }
    }

    if (data?.state && data?.state !== GitspaceStatus.RUNNING && redirectFrom) {
      startTrigger()
    }
  }, [data?.state, redirectFrom, mutateLoading, startTriggred])

  const formattedlogsdata = useGetLogStream(standalone ? { response } : { response: undefined })

  const confirmDelete = useConfirmAct()

  const handleDelete = async (e: React.MouseEvent<HTMLDivElement, MouseEvent>) => {
    confirmDelete({
      intent: 'danger',
      title: getString('cde.deleteGitspaceTitle', { name: data?.name }),
      message: getString('cde.deleteGitspaceText'),
      confirmText: getString('delete'),
      action: async () => {
        try {
          e.preventDefault()
          e.stopPropagation()
          await deleteGitspace(standalone ? {} : gitspaceId || '')
          showSuccess(getString('cde.deleteSuccess'))
          history.push(routes.toCDEGitspaces({ space }))
        } catch (exception) {
          showError(getErrorMessage(exception))
        }
      }
    })
  }

  const [accountIdentifier, orgIdentifier, projectIdentifier] = data?.space_path?.split('/') || []

  const { refetchToken, setSelectedRowUrl } = useOpenVSCodeBrowserURL()

  const accordionRef = useRef<AccordionHandle | null>(null)

  useEffect(() => {
    if (standalone ? formattedlogsdata.data : responseData) {
      accordionRef.current?.open('logsCard')
    } else {
      accordionRef.current?.close('logsCard')
    }
  }, [standalone, responseData, formattedlogsdata.data])

  return (
    <>
      <Page.Header
        title=""
        breadcrumbs={
          <Breadcrumbs
            links={[
              { url: routes.toCDEGitspaces({ space }), label: getString('cde.gitspaces') },
              { url: routes.toCDEGitspaceDetail({ gitspaceId, space }), label: data?.name || 'Gitspace Name' }
            ]}
          />
        }
      />
      <Page.SubHeader className={css.customSubheader}>
        <Layout.Horizontal width="100%" flex={{ alignItems: 'center', justifyContent: 'space-between' }}>
          <Container>
            <Layout.Horizontal spacing="small">
              {data && (
                <img src={data?.ide === IDEType.VSCODEWEB ? vsCodeWebIcon : vscodeIcon} height={32} width={32} />
              )}
              <Text font={{ variation: FontVariation.H3 }}>{data?.name}</Text>
            </Layout.Horizontal>
          </Container>

          <Container>
            <Layout.Horizontal spacing="large">
              <Button
                variation={ButtonVariation.SECONDARY}
                rightIcon="chevron-down"
                tooltipProps={
                  {
                    interactionKind: PopoverInteractionKind.CLICK,
                    position: PopoverPosition.BOTTOM_LEFT,
                    popoverClassName: css.popover
                  } as PopoverProps
                }
                tooltip={
                  <Menu>
                    {!disabledActionButtons && (
                      <MenuItem
                        text={
                          <Layout.Horizontal
                            spacing="small"
                            flex={{ alignItems: 'center', justifyContent: 'flex-start' }}>
                            {data?.state === GitspaceStatus.RUNNING ? (
                              <img src={pauseIcon} height={16} width={16} />
                            ) : (
                              <Play />
                            )}
                            <Text>
                              {data?.state === GitspaceStatus.RUNNING
                                ? mutateLoading
                                  ? getString('cde.stopingGitspace')
                                  : getString('cde.details.stopGitspace')
                                : mutateLoading
                                ? getString('cde.startingGitspace')
                                : getString('cde.details.startGitspace')}
                            </Text>
                          </Layout.Horizontal>
                        }
                        disabled={loading || mutateLoading || disabledActionButtons}
                        onClick={async () => {
                          try {
                            setStartPolling(GitspaceActionType.START)
                            await actionMutate({ action: data?.state === GitspaceStatus.RUNNING ? 'stop' : 'start' })
                            await refetch()
                            setStartPolling(undefined)
                            updateQueryParams({ redirectFrom: undefined })
                          } catch (err) {
                            showError(getErrorMessage(err))
                            setStartPolling(undefined)
                          }
                        }}
                      />
                    )}
                    <MenuItem
                      text={
                        <Layout.Horizontal
                          spacing="small"
                          flex={{ alignItems: 'center', justifyContent: 'flex-start' }}>
                          <img src={homeIcon} height={16} width={16} />
                          <Text>{getString('cde.details.goToDashboard')}</Text>
                        </Layout.Horizontal>
                      }
                      onClick={() => {
                        history.push(routes.toCDEGitspaces({ space }))
                      }}
                    />
                    <MenuItem
                      onClick={handleDelete as Unknown as () => void}
                      text={
                        <Layout.Horizontal
                          spacing="small"
                          flex={{ alignItems: 'center', justifyContent: 'flex-start' }}>
                          {deleteLoading ? <></> : <img src={deleteIcon} height={16} width={16} />}
                          <Text color={Color.RED_450} icon={deleteLoading ? 'loading' : undefined}>
                            {getString('cde.deleteGitspace')}
                          </Text>
                        </Layout.Horizontal>
                      }
                      disabled={disabledActionButtons}
                    />
                  </Menu>
                }>
                {getString('cde.details.actions')}
              </Button>
              {(data?.state === GitspaceStatus.RUNNING ||
                data?.state === GitspaceStatus.STARTING ||
                data?.state === GitspaceStatus.STOPPING) &&
              data?.ide ? (
                <Button
                  disabled={disabledActionButtons}
                  variation={ButtonVariation.PRIMARY}
                  tooltip={
                    disabledActionButtons ? (
                      <Container width={300} padding="medium">
                        <Layout.Vertical spacing="small">
                          <Text color={Color.WHITE} font="small">
                            We are provisioning the Gitspace
                          </Text>
                          <Text color={Color.WHITE} font="small">
                            Please wait for a few minutes before the VS Code Desktop can be launched
                          </Text>
                        </Layout.Vertical>
                      </Container>
                    ) : undefined
                  }
                  tooltipProps={{ isDark: true, position: PopoverPosition.BOTTOM_LEFT }}
                  onClick={e => {
                    e.preventDefault()
                    e.stopPropagation()
                    if (data?.ide === IDEType.VSCODE) {
                      const params = standalone ? '?gitness' : ''
                      const projectOrSpace = standalone ? space : projectIdentifier
                      const vscodeExtensionCode = standalone ? 'harness-inc.oss-gitspaces' : 'harness-inc.gitspaces'
                      const vsCodeURL = `vscode://${vscodeExtensionCode}/${projectOrSpace}/${data?.identifier}${params}`
                      window.open(vsCodeURL, '_blank')
                    } else {
                      if (standalone) {
                        window.open(data?.instance?.url || '', '_blank')
                      } else {
                        setSelectedRowUrl(data?.instance?.url || '')
                        refetchToken({
                          pathParams: {
                            accountIdentifier,
                            projectIdentifier,
                            orgIdentifier,
                            gitspace_identifier: data?.identifier || ''
                          }
                        })
                      }
                    }
                  }}>
                  {data?.ide === IDEType.VSCODE && getString('cde.details.openEditor')}
                  {data?.ide === IDEType.VSCODEWEB && getString('cde.details.openBrowser')}
                </Button>
              ) : (
                <Button
                  loading={mutateLoading}
                  disabled={mutateLoading || disabledActionButtons}
                  icon="run-pipeline"
                  variation={ButtonVariation.PRIMARY}
                  intent="success"
                  onClick={async () => {
                    try {
                      setStartPolling(GitspaceActionType.START)
                      await actionMutate({ action: GitspaceActionType.START })
                      await refetch()
                      setStartPolling(undefined)
                      updateQueryParams({ redirectFrom: undefined })
                    } catch (err) {
                      showError(getErrorMessage(err))
                      setStartPolling(undefined)
                    }
                  }}>
                  {getString('cde.details.startGitspace')}
                </Button>
              )}
            </Layout.Horizontal>
          </Container>
        </Layout.Horizontal>
      </Page.SubHeader>

      <Page.Body
        loading={loading}
        error={getErrorMessage(error)}
        noData={{
          when: () => !data?.identifier,
          message: getString('cde.details.noData')
        }}
        className={css.pageMain}>
        <Container>
          <Card className={css.cardContainer}>
            <Text font={{ variation: FontVariation.CARD_TITLE }}>{getString('cde.gitspaceDetail')}</Text>
            <DetailsCard data={data} loading={mutateLoading} />
          </Card>
          <Card className={css.cardContainer}>
            <EventTimelineAccordion data={eventData as TypesGitspaceEventResponse[]} />
          </Card>

          <Card className={css.cardContainer}>
            <Accordion activeId={''} ref={accordionRef}>
              <Accordion.Panel
                className={css.accordionnCustomSummary}
                summary={
                  <Layout.Vertical spacing="small">
                    <Text
                      rightIcon={isStreamingLogs || logsLoading ? 'steps-spinner' : undefined}
                      className={css.containerlogsTitle}
                      font={{ variation: FontVariation.CARD_TITLE }}
                      margin={{ left: 'large' }}>
                      {getString('cde.details.containerLogs')}
                    </Text>
                    <Text margin={{ left: 'large' }}>{getString('cde.details.containerLogsSubText')} </Text>
                  </Layout.Vertical>
                }
                id="logsCard"
                details={
                  <Container width="100%">
                    <ContainerLogs data={standalone ? formattedlogsdata.data : responseData} />
                  </Container>
                }
              />
            </Accordion>
          </Card>
        </Container>
      </Page.Body>
    </>
  )
}
export default GitspaceDetails
