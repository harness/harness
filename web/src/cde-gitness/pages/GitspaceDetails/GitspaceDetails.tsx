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
  AccordionHandle,
  ButtonSize,
  Utils
} from '@harnessio/uicore'
import cx from 'classnames'
import { NavArrowDown, NavArrowRight, Play } from 'iconoir-react'
import { Render } from 'react-jsx-match'
import { useHistory, useParams } from 'react-router-dom'
import { Color, FontVariation, PopoverProps } from '@harnessio/design-system'
import { Menu, MenuItem, PopoverInteractionKind, PopoverPosition } from '@blueprintjs/core'
import { defaultTo } from 'lodash-es'
import { Icon } from '@harnessio/icons'
import { EditGitspace } from 'cde-gitness/components/EditGitspace/EditGitspace'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { useAppContext } from 'AppContext'
import { useStrings } from 'framework/strings'
import EventTimelineAccordion from 'cde-gitness/components/EventTimelineAccordion/EventTimelineAccordion'
import { DetailsCard } from 'cde-gitness/components/DetailsCard/DetailsCard'
import { useFindGitspaceSettings } from 'services/cde'
import type { EnumGitspaceStateType, TypesGitspaceEventResponse } from 'cde-gitness/services'
import { getIDEOption, GitspaceActionType, GitspaceStatus } from 'cde-gitness/constants'
import { useQueryParams } from 'hooks/useQueryParams'
import { useUpdateQueryParams } from 'hooks/useUpdateQueryParams'
import { getErrorMessage } from 'utils/Utils'
import { usePolling } from 'cde-gitness/hooks/usePolling'
import deleteIcon from 'cde-gitness/assests/delete.svg?url'
import pauseIcon from 'cde-gitness/assests/pause.svg?url'
import homeIcon from 'cde-gitness/assests/home.svg?url'
import { useConfirmAct } from 'hooks/useConfirmAction'
import { useGitspaceDetails } from 'cde-gitness/hooks/useGitspaceDetails'
import { useGitspaceEvents } from 'cde-gitness/hooks/useGitspaceEvents'
import { useGitspaceActions } from 'cde-gitness/hooks/useGitspaceActions'
import { useDeleteGitspaces } from 'cde-gitness/hooks/useDeleteGitspaces'
import { useGitspacesLogs } from 'cde-gitness/hooks/useGitspaceLogs'
import { ErrorCard } from 'cde-gitness/components/ErrorCard/ErrorCard'
import CopyButton from 'cde-gitness/components/CopyButton/CopyButton'
import ContainerLogs from '../../components/ContainerLogs/ContainerLogs'
import { useGetLogStream } from '../../hooks/useGetLogStream'
import Logger, { LoggerProps } from './Logger/Logger'
import css from './GitspaceDetails.module.scss'

const LogSection = (props: LoggerProps & { title: string; isBottom?: boolean; handlePageScrollClick: () => void }) => {
  const { getString } = useStrings()
  const [initExpand, setInitExpand] = useState<boolean>(false)
  return (
    <Container>
      <Layout.Horizontal
        spacing="small"
        className={cx(css.stepHeader, {
          [css.expanded]: initExpand,
          [css.selected]: initExpand
        })}
        onClick={() => {
          setInitExpand(!initExpand)
        }}>
        {initExpand ? (
          <NavArrowDown color={Utils.getRealCSSColor(Color.GREY_500)} className={cx(css.noShrink)} />
        ) : (
          <NavArrowRight color={Utils.getRealCSSColor(Color.GREY_500)} className={cx(css.noShrink)} />
        )}
        <Text className={css.name} color={initExpand ? Color.PRIMARY_7 : Color.GREY_500} lineClamp={1}>
          {props.title}
        </Text>
      </Layout.Horizontal>
      <Render when={initExpand}>
        <Container margin="medium" padding="medium">
          <Logger {...props} />
        </Container>
        <Button
          size={ButtonSize.SMALL}
          variation={ButtonVariation.PRIMARY}
          text={props?.isBottom ? getString('top') : getString('bottom')}
          icon={props?.isBottom ? 'arrow-up' : 'arrow-down'}
          iconProps={{ size: 10 }}
          onClick={props?.handlePageScrollClick}
          className={css.scrollDownBtn}
        />
      </Render>
    </Container>
  )
}

const GitspaceDetails = () => {
  const space = useGetSpaceParam()
  const { getString } = useStrings()
  const { routes, standalone, accountInfo } = useAppContext()
  const { showError, showSuccess } = useToaster()
  const history = useHistory()
  const containerRef = useRef<HTMLDivElement | null>(null)
  const initcontainerRef = useRef<HTMLDivElement | null>(null)
  const [startTriggred, setStartTriggred] = useState<boolean>(false)
  const [triggerPollingOnStart, setTriggerPollingOnStart] = useState<EnumGitspaceStateType>()
  const { gitspaceId = '' } = useParams<{ gitspaceId?: string }>()

  const logCardId = 'logsCard'
  const [expandedTab, setExpandedTab] = useState('')
  const [isStreamingLogs, setIsStreamingLogs] = useState(false)
  const [isBottom, setIsBottom] = useState(false)

  const [startPolling, setStartPolling] = useState<GitspaceActionType | undefined>(undefined)
  const [isEditModalOpen, setIsEditModalOpen] = useState<boolean>(false)

  const { loading, data, refetch, error } = useGitspaceDetails({ gitspaceId })

  const { data: eventData, refetch: refetchEventData } = useGitspaceEvents({ gitspaceId })

  const {
    refetch: refetchLogsData,
    response,
    error: streamLogsError,
    loading: logsLoading
  } = useGitspacesLogs({ gitspaceId })

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
    const refetchLogs = disabledActionButtons && filteredEvent?.length && !isStreamingLogs
    if (standalone) {
      if (refetchLogs) {
        refetchLogsData()
        setIsStreamingLogs(true)
      } else if (
        (filteredEvent?.length && !disabledActionButtons && isStreamingLogs) ||
        (isStreamingLogs && streamLogsError)
      ) {
        setIsStreamingLogs(false)
      }
    } else {
      if (refetchLogs) {
        setIsStreamingLogs(true)
        viewLogs()
      } else if (filteredEvent?.length && !disabledActionButtons && isStreamingLogs) {
        setIsStreamingLogs(false)
        viewLogs()
      }
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

  const confirmDelete = useConfirmAct()

  const handleReset = async (e: React.MouseEvent<HTMLDivElement, MouseEvent>) => {
    confirmDelete({
      intent: 'danger',
      title: `${getString('cde.resetGitspace')} '${data?.name}'`,
      message: getString('cde.resetGitspaceText'),
      confirmText: getString('cde.reset'),
      action: async () => {
        try {
          setStartPolling(GitspaceActionType.RESET)
          e.preventDefault()
          e.stopPropagation()
          await actionMutate({ action: GitspaceActionType.RESET })
          showSuccess(getString('cde.resetGitspaceSuccess'))
          await refetch()
          setStartPolling(undefined)
        } catch (exception) {
          showError(getErrorMessage(exception))
          setStartPolling(undefined)
        }
      }
    })
  }

  const handleDelete = async (e: React.MouseEvent<HTMLDivElement, MouseEvent>) => {
    confirmDelete({
      intent: 'danger',
      title: `${getString('cde.deleteGitspace')} '${data?.name}'`,
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

  const formattedlogsdata = useGetLogStream({ response })

  const [accountIdentifier, orgIdentifier, projectIdentifier] = data?.space_path?.split('/') || []

  const {
    data: gitspaceSettings,
    loading: settingsLoading,
    error: settingsError
  } = useFindGitspaceSettings({
    accountIdentifier: accountIdentifier || '',
    lazy: !accountIdentifier || standalone
  })

  const accordionRef = useRef<AccordionHandle | null>(null)
  const myRef = useRef<any | null>(null)
  const selectedIde = getIDEOption(data?.ide, getString)

  useEffect(() => {
    if (standalone) {
      if (formattedlogsdata.data) {
        accordionRef.current?.open('logsCard')
      } else {
        accordionRef.current?.close('logsCard')
      }
    }
  }, [standalone, formattedlogsdata.data])

  const triggerGitspace = async () => {
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
  }

  const viewLogs = () => {
    myRef.current?.scrollIntoView()
    accordionRef.current?.open('logsCard')
    if (!standalone) {
      setExpandedTab('logsCard')
    }
  }

  const handleClick = () => {
    const logContainer = containerRef.current as HTMLDivElement
    const initlogContainer = initcontainerRef.current as HTMLDivElement
    const scrollParent = logContainer?.parentElement as HTMLDivElement
    const initScrollParent = initlogContainer?.parentElement as HTMLDivElement
    if (!isBottom) {
      scrollParent.scrollTop = scrollParent.scrollHeight
      initScrollParent.scrollTop = initScrollParent.scrollHeight
      setIsBottom(true)
    } else if (isBottom) {
      scrollParent.scrollTop = 0
      initScrollParent.scrollTop = 0
      setIsBottom(false)
    }
  }
  const getBreadcrumbLinks = () => {
    if (standalone) {
      return [
        { url: routes.toCDEGitspaces({ space }), label: getString('cde.gitspaces') },
        { url: routes.toCDEGitspaceDetail({ gitspaceId, space }), label: data?.name || 'Gitspace Name' }
      ]
    }
    return [
      {
        url: `/account/${accountIdentifier}/module/cde`,
        label: `Account: ${accountInfo.name}`
      },
      {
        url: `/account/${accountIdentifier}/module/cde/orgs/${orgIdentifier}`,
        label: `Organization: ${orgIdentifier}`
      },
      {
        url: `/account/${accountIdentifier}/module/cde/orgs/${orgIdentifier}/projects/${projectIdentifier}`,
        label: `Project: ${projectIdentifier}`
      },
      {
        url: routes.toCDEGitspaces({ space }),
        label: getString('cde.gitspaces')
      },
      { url: routes.toCDEGitspaceDetail({ gitspaceId, space }), label: data?.name || 'Gitspace Name' }
    ]
  }

  return (
    <>
      <Page.Header
        title=""
        breadcrumbs={<Breadcrumbs className={css.customBreadcumbStyles} links={getBreadcrumbLinks()} />}
      />
      <Page.SubHeader className={css.customSubheader}>
        <Layout.Horizontal width="100%" flex={{ alignItems: 'center', justifyContent: 'space-between' }}>
          <Container>
            {data && (
              <Layout.Horizontal spacing="small">
                <img src={selectedIde?.icon} className={css.gitspaceIcon} height={32} width={32} />
                <Layout.Vertical spacing="none" className={css.gitspaceIdContainer}>
                  <Text font={{ variation: FontVariation.H3 }}>{data?.name}</Text>
                  <Layout.Horizontal spacing={'xsmall'} flex={{ alignItems: 'center', justifyContent: 'start' }}>
                    <Text font={{ size: 'small' }}>
                      {getString('cde.id')}: {data?.identifier}
                    </Text>
                    <CopyButton value={data?.identifier} className={css.copyBtn} />
                  </Layout.Horizontal>
                </Layout.Vertical>
              </Layout.Horizontal>
            )}
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
                            spacing="xsmall"
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
                          spacing="xsmall"
                          flex={{ alignItems: 'center', justifyContent: 'flex-start' }}>
                          <img src={homeIcon} height={16} width={16} />
                          <Text>{getString('cde.details.goToDashboard')}</Text>
                        </Layout.Horizontal>
                      }
                      onClick={() => {
                        history.push(routes.toCDEGitspaces({ space }))
                      }}
                    />

                    {!standalone &&
                      [GitspaceStatus.UNINITIALIZED, GitspaceStatus.STOPPED].includes(
                        data?.state as GitspaceStatus
                      ) && (
                        <MenuItem
                          text={
                            <Layout.Horizontal
                              spacing="xsmall"
                              flex={{ alignItems: 'center', justifyContent: 'flex-start' }}>
                              <Icon name="edit" size={16} />
                              <Text>{getString('cde.editGitspace') || 'Edit Gitspace'}</Text>
                            </Layout.Horizontal>
                          }
                          onClick={() => {
                            setIsEditModalOpen(true)
                          }}
                        />
                      )}
                    <MenuItem
                      onClick={handleReset as Unknown as () => void}
                      text={
                        <Text icon={'canvas-reset'} iconProps={{ size: 16 }}>
                          {getString('cde.resetGitspace')}
                        </Text>
                      }
                    />
                    <MenuItem
                      onClick={handleDelete as Unknown as () => void}
                      text={
                        <Layout.Horizontal
                          spacing="xsmall"
                          flex={{ alignItems: 'center', justifyContent: 'flex-start' }}>
                          {deleteLoading ? <></> : <img src={deleteIcon} height={16} width={16} />}
                          <Text color={Color.RED_450} icon={deleteLoading ? 'loading' : undefined}>
                            {getString('cde.deleteGitspace')}
                          </Text>
                        </Layout.Horizontal>
                      }
                    />
                  </Menu>
                }>
                {getString('cde.details.actions')}
              </Button>
              {data ? (
                <>
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
                                Please wait for a few minutes before the {selectedIde?.label} can be launched
                              </Text>
                            </Layout.Vertical>
                          </Container>
                        ) : undefined
                      }
                      tooltipProps={{ isDark: true, position: PopoverPosition.BOTTOM_LEFT }}
                      onClick={e => {
                        e.preventDefault()
                        e.stopPropagation()
                        const url = data?.instance?.plugin_url ? data?.instance?.plugin_url : data?.instance?.url
                        {
                          !url ? showError(getString('cde.ide.errorEmptyURL')) : window.open(`${url}`, '_blank')
                        }
                      }}>
                      {selectedIde?.buttonText}
                    </Button>
                  ) : (
                    <Button
                      loading={mutateLoading}
                      disabled={mutateLoading || disabledActionButtons}
                      icon="run-pipeline"
                      variation={ButtonVariation.PRIMARY}
                      intent="success"
                      onClick={async () => {
                        triggerGitspace()
                      }}>
                      {getString('cde.details.startGitspace')}
                    </Button>
                  )}
                </>
              ) : (
                <Button
                  loading={mutateLoading}
                  disabled={mutateLoading || disabledActionButtons}
                  icon="run-pipeline"
                  variation={ButtonVariation.PRIMARY}
                  intent="success"
                  onClick={async () => {
                    triggerGitspace()
                  }}>
                  {getString('cde.details.startGitspace')}
                </Button>
              )}
            </Layout.Horizontal>
          </Container>
        </Layout.Horizontal>
      </Page.SubHeader>

      <Page.Body
        loading={loading || settingsLoading}
        error={getErrorMessage(error) || getErrorMessage(settingsError)}
        noData={{
          when: () => !data?.identifier,
          message: getString('cde.details.noData')
        }}
        className={css.pageMain}>
        <Container>
          {data?.instance?.error_message ? (
            <ErrorCard
              message={data?.instance?.error_message}
              triggerGitspace={triggerGitspace}
              loading={mutateLoading}
              viewLogs={viewLogs}
            />
          ) : (
            <></>
          )}
          <Card className={css.cardContainer}>
            <Text font={{ variation: FontVariation.CARD_TITLE }} className={css.marginLeftContainer}>
              {getString('cde.gitspaceDetail')}
            </Text>
            <DetailsCard data={data} standalone={standalone} loading={mutateLoading} />
          </Card>
          <Card className={css.cardContainer}>
            <EventTimelineAccordion data={eventData as TypesGitspaceEventResponse[]} />
          </Card>
          <Card className={css.cardContainer}>
            <Container ref={myRef}>
              <Accordion
                activeId={expandedTab}
                ref={accordionRef}
                onChange={(e: string) => {
                  setExpandedTab(e)
                }}>
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
                  id={logCardId}
                  details={
                    standalone ? (
                      <ContainerLogs data={formattedlogsdata.data} />
                    ) : (
                      <Container width="100%" className={css.consoleContainer}>
                        <LogSection
                          title="Initialise"
                          value={`init_${data?.name}`}
                          state={data?.state ?? ''}
                          logKey={(data as { initialize_log_key: string })?.initialize_log_key ?? ''}
                          isStreaming={false}
                          expanded={true}
                          localRef={initcontainerRef}
                          setIsBottom={setIsBottom}
                          isBottom={isBottom}
                          handlePageScrollClick={handleClick}
                        />

                        <LogSection
                          title="Container logs"
                          value={data?.name ?? ''}
                          state={data?.state ?? ''}
                          logKey={data?.log_key ?? ''}
                          isStreaming={isStreamingLogs}
                          expanded={true}
                          localRef={containerRef}
                          setIsBottom={setIsBottom}
                          isBottom={isBottom}
                          handlePageScrollClick={handleClick}
                        />

                        <Button
                          size={ButtonSize.SMALL}
                          variation={ButtonVariation.PRIMARY}
                          text={isBottom ? getString('top') : getString('bottom')}
                          icon={isBottom ? 'arrow-up' : 'arrow-down'}
                          iconProps={{ size: 10 }}
                          onClick={handleClick}
                          className={css.scrollDownBtn}
                        />
                      </Container>
                    )
                  }
                />
              </Accordion>
            </Container>
          </Card>
        </Container>
      </Page.Body>
      {/* EditGitspace Modal */}
      {!standalone && isEditModalOpen && data && (
        <Container
          onClick={e => {
            e.stopPropagation()
          }}>
          <EditGitspace
            isOpen={isEditModalOpen}
            gitspaceSettings={gitspaceSettings || null}
            onClose={() => setIsEditModalOpen(false)}
            gitspaceId={gitspaceId}
            onGitspaceUpdated={() => {
              refetch()
              refetchEventData()
              if (standalone) {
                refetchLogsData()
              }
            }}
            gitspaceData={{
              name: data.name || '',
              ide: data.ide || 'vs_code',
              branch: data.branch || '',
              devcontainer_path: data.devcontainer_path || '',
              ssh_token_identifier: data.ssh_token_identifier || '',
              resource:
                'resource' in data && data.resource
                  ? {
                      identifier: data.resource.identifier || '',
                      config_identifier: data.resource.config_identifier || '',
                      name: data.resource.name || '',
                      region: data.resource.region || '',
                      disk: data.resource.disk || '',
                      cpu: data.resource.cpu || '',
                      memory: data.resource.memory || '',
                      persistent_disk_type: data.resource.metadata?.persistent_disk_type || ''
                    }
                  : undefined
            }}
          />
        </Container>
      )}
    </>
  )
}
export default GitspaceDetails
