import React, { useEffect, useState } from 'react'
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
  useToaster
} from '@harnessio/uicore'
import { Play } from 'iconoir-react'
import { useHistory, useParams } from 'react-router-dom'
import { Color, FontVariation, PopoverProps } from '@harnessio/design-system'
import { Menu, MenuItem, PopoverInteractionKind, PopoverPosition } from '@blueprintjs/core'
import { useGet, useMutate } from 'restful-react'
import { defaultTo } from 'lodash-es'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { useAppContext } from 'AppContext'
import { useStrings } from 'framework/strings'
import EventTimelineAccordion from 'cde-gitness/components/EventTimelineAccordion/EventTimelineAccordion'
import { DetailsCard } from 'cde-gitness/components/DetailsCard/DetailsCard'
import type { EnumGitspaceStateType, TypesGitspaceConfig, TypesGitspaceEventResponse } from 'cde-gitness/services'
import { GitspaceActionType, GitspaceStatus } from 'cde/constants'
import { useQueryParams } from 'hooks/useQueryParams'
import { useUpdateQueryParams } from 'hooks/useUpdateQueryParams'
import { getErrorMessage } from 'utils/Utils'
import { usePolling } from 'cde/components/GitspaceDetails/usePolling'
import deleteIcon from 'cde-gitness/assests/delete.svg?url'
import vscodeIcon from 'cde/icons/VSCode.svg?url'
import vsCodeWebIcon from 'cde-gitness/assests/vsCodeWeb.svg?url'
import pauseIcon from 'cde-gitness/assests/pause.svg?url'
import { StandaloneIDEType } from 'cde-gitness/constants'
import homeIcon from 'cde-gitness/assests/home.svg?url'
import { useConfirmAct } from 'hooks/useConfirmAction'
import ContainerLogs from '../../components/ContainerLogs/ContainerLogs'
import { useGetLogStream } from '../../hooks/useGetLogStream'
import css from './GitspaceDetails.module.scss'

export const GitspaceDetails = () => {
  const space = useGetSpaceParam()
  const { getString } = useStrings()
  const { routes } = useAppContext()
  const { showError, showSuccess } = useToaster()
  const history = useHistory()
  const [startTriggred, setStartTriggred] = useState<boolean>(false)
  const [triggerPollingOnStart, setTriggerPollingOnStart] = useState<EnumGitspaceStateType>()
  const { gitspaceId = '' } = useParams<{ gitspaceId?: string }>()

  const [isStreamingLogs, setIsStreamingLogs] = useState(false)

  const [startPolling, setStartPolling] = useState<GitspaceActionType | undefined>(undefined)

  const { loading, data, refetch, error } = useGet<TypesGitspaceConfig>({
    path: `/api/v1/gitspaces/${space}/${gitspaceId}/+`,
    debounce: 500
  })

  const { data: eventData, refetch: refetchEventData } = useGet<TypesGitspaceEventResponse[]>({
    path: `/api/v1/gitspaces/${space}/${gitspaceId}/+/events`,
    debounce: 500
  })

  const {
    refetch: refetchLogsData,
    response,
    error: streamLogsError
  } = useGet<any>({
    path: `api/v1/gitspaces/${space}/${gitspaceId}/+/logs/stream`,
    debounce: 500,
    lazy: true
  })

  const { mutate: actionMutate, loading: mutateLoading } = useMutate({
    verb: 'POST',
    path: `/api/v1/gitspaces/${space}/${gitspaceId}/+/actions`
  })

  const { mutate: deleteGitspace, loading: deleteLoading } = useMutate<any>({
    verb: 'DELETE',
    path: `/api/v1/gitspaces/${space}/${gitspaceId}/+`
  })

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
    const filteredEvent = eventData?.filter(
      item =>
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
      pollingInterval: 10000,
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

  const formattedlogsdata = useGetLogStream({ response })

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
          await deleteGitspace({})
          showSuccess(getString('cde.deleteSuccess'))
          history.push(routes.toCDEGitspaces({ space }))
        } catch (exception) {
          showError(getErrorMessage(exception))
        }
      }
    })
  }

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
                <img
                  src={data?.ide === StandaloneIDEType.VSCODEWEB ? vsCodeWebIcon : vscodeIcon}
                  height={32}
                  width={32}
                />
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
                    if (data?.ide === StandaloneIDEType.VSCODE) {
                      const pathparamsList = data?.space_path?.split('/') || []
                      const projectIdentifier = pathparamsList[pathparamsList.length - 1] || ''
                      window.open(
                        `vscode://harness-inc.oss-gitspaces/${projectIdentifier}/${data?.identifier}?gitness`,
                        '_blank'
                      )
                    } else {
                      window.open(data?.instance?.url || '', '_blank')
                    }
                  }}>
                  {data?.ide === StandaloneIDEType.VSCODE && getString('cde.details.openEditor')}
                  {data?.ide === StandaloneIDEType.VSCODEWEB && getString('cde.details.openBrowser')}
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
            <EventTimelineAccordion data={eventData} />
          </Card>

          <Card className={css.cardContainer}>
            <Accordion activeId="logsCard">
              <Accordion.Panel
                shouldRender
                className={css.accordionnCustomSummary}
                summary={
                  <Layout.Vertical spacing="small">
                    <Text font={{ variation: FontVariation.CARD_TITLE }} margin={{ left: 'large' }}>
                      {getString('cde.details.containerLogs')}
                    </Text>
                    <Text margin={{ left: 'large' }}>{getString('cde.details.containerLogsSubText')} </Text>
                  </Layout.Vertical>
                }
                id="logsCard"
                details={
                  <Container>
                    <ContainerLogs data={formattedlogsdata.data} />
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
