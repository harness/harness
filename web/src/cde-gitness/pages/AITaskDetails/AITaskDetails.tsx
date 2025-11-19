import React from 'react'
import { Avatar, Breadcrumbs, Card, Container, Layout, Page, Text, Button, ButtonVariation } from '@harnessio/uicore'
import { FontVariation, Color } from '@harnessio/design-system'
import { useHistory, useParams } from 'react-router-dom'
import { Menu, MenuItem, PopoverInteractionKind, Position, PopoverPosition } from '@blueprintjs/core'
import type { IconName } from '@harnessio/icons'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { useAppContext } from 'AppContext'
import { useGetCDEAPIParams } from 'cde-gitness/hooks/useGetCDEAPIParams'
import { usePolling } from 'cde-gitness/hooks/usePolling'
import CopyButton from 'cde-gitness/components/CopyButton/CopyButton'
import { useFindAITask, type TypesGitspaceConfig } from 'services/cde'
import { useStrings } from 'framework/strings'
import { getIDEOption, TaskStatus, GitspaceStatus, AIAgentEnum, getIconByAgentType } from 'cde-gitness/constants'
import TaskErrorCard from '../../components/TaskErrorCard/TaskErrorCard'
import { AITaskDetailsCard } from '../../components/AITaskDetailsCard/AITaskDetailsCard'
import { AIUsageMetricsCard } from '../../components/AIUsageMetricsCard/AIUsageMetricsCard'
import { MarkdownText } from '../../utils/MarkdownUtils'
import css from './AITaskDetails.module.scss'

const AITaskDetails = () => {
  const space = useGetSpaceParam()
  const { routes, accountInfo, standalone } = useAppContext()
  const { orgIdentifier, projectIdentifier, accountIdentifier } = useGetCDEAPIParams()
  const history = useHistory()
  const { getString } = useStrings()
  const { aitaskId = '' } = useParams<{ aitaskId?: string }>()
  const getBreadcrumbLinks = () => {
    if (standalone) {
      return [{ url: routes.toCDEAITasks({ space }), label: 'Tasks' }]
    }
    return [
      {
        url: `/account/${accountIdentifier}/module/cde`,
        label: `Account: ${accountInfo?.name || accountIdentifier}`
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
        url: routes.toCDEAITasks({ space }),
        label: getString('cde.aiTasks.tasks')
      }
    ]
  }

  const {
    data: task,
    loading,
    error,
    refetch
  } = useFindAITask({
    accountIdentifier: accountIdentifier || '',
    orgIdentifier: orgIdentifier || '',
    projectIdentifier: projectIdentifier || '',
    aitask_identifier: aitaskId || ''
  })

  const title: string = task?.display_name || task?.initial_prompt || ''
  const clampedTitle: string = title && title.length > 80 ? `${title.slice(0, 80)}…` : title
  const titleTooltip: string = title
  const userDisplayName: string = task?.gitspace_config?.user_display_name || '—'
  const userEmail = task?.gitspace_config?.user_email || '—'
  const selectedIde = getIDEOption(task?.gitspace_config?.ide, getString)
  const taskState = task?.state as TaskStatus | undefined
  const gitspaceState = task?.gitspace_config?.instance?.state as GitspaceStatus | string | undefined
  const agentIcon = getIconByAgentType(task?.ai_agent || AIAgentEnum.CLAUDE_CODE)
  const isGitspaceDeleted = gitspaceState === 'deleted'
  const isWaiting = taskState === TaskStatus.UNINITIALIZED || taskState === TaskStatus.RUNNING
  const agentMarkdown = String(task?.output || '').trim()
  const shouldShowAgentMarkdown =
    taskState === TaskStatus.COMPLETED || (taskState === TaskStatus.ERROR && agentMarkdown)
  const waitingText = getString('cde.aiTasks.details.waitingForAgent')

  const ideLabel = selectedIde?.label || 'IDE'
  const ideDisabled =
    taskState === TaskStatus.ERROR || gitspaceState !== GitspaceStatus.RUNNING || taskState !== TaskStatus.COMPLETED
  let ideTooltipText: string | undefined
  if (taskState === TaskStatus.ERROR) {
    ideTooltipText = `${ideLabel} can’t be opened because the task encountered an error.`
  } else if (gitspaceState !== GitspaceStatus.RUNNING) {
    ideTooltipText = `${ideLabel} can’t be opened because the Gitspace isn’t active. Start the Gitspace and try again.`
  } else if (taskState !== TaskStatus.COMPLETED) {
    ideTooltipText = `Setting things up. ${ideLabel} will be available once the task finishes.`
  } else {
    ideTooltipText = undefined
  }

  usePolling(
    async () => {
      await refetch?.()
    },
    { pollingInterval: standalone ? 2000 : 3000, startCondition: Boolean(isWaiting) }
  )

  return (
    <>
      <Page.Header
        title=""
        breadcrumbs={<Breadcrumbs className={css.customBreadcumbStyles} links={getBreadcrumbLinks()} />}
      />
      <Page.SubHeader className={css.customSubheader}>
        <Layout.Horizontal width="100%" flex={{ alignItems: 'center', justifyContent: 'space-between' }}>
          <Container>
            {task && (
              <Layout.Horizontal spacing="small">
                {agentIcon && <img src={agentIcon} className={css.gitspaceIcon} height={32} width={32} />}
                <Layout.Vertical spacing="none" className={css.gitspaceIdContainer}>
                  <Text lineClamp={1} font={{ variation: FontVariation.H3 }} title={titleTooltip}>
                    {clampedTitle}
                  </Text>
                  <Layout.Horizontal spacing="xsmall" flex={{ alignItems: 'center', justifyContent: 'start' }}>
                    <Text font={{ size: 'small' }}>id: {task.identifier || String(task.id || '')}</Text>
                    <CopyButton value={task.identifier || String(task.id || '')} className={css.copyBtn} />
                  </Layout.Horizontal>
                </Layout.Vertical>
              </Layout.Horizontal>
            )}
          </Container>

          <Container>
            <Layout.Horizontal spacing="medium" flex={{ alignItems: 'center', justifyContent: 'center' }}>
              <Button
                variation={ButtonVariation.PRIMARY}
                disabled={ideDisabled}
                tooltip={
                  ideDisabled && ideTooltipText ? (
                    <Container width={250} padding="medium">
                      <Layout.Vertical spacing="xsmall">
                        <Text color={Color.WHITE} font="small">
                          {ideTooltipText}
                        </Text>
                      </Layout.Vertical>
                    </Container>
                  ) : undefined
                }
                tooltipProps={{ isDark: true, position: PopoverPosition.BOTTOM_LEFT }}
                onClick={e => {
                  e.preventDefault()
                  e.stopPropagation()
                  const url = task?.gitspace_config?.instance?.plugin_url
                    ? task?.gitspace_config?.instance?.plugin_url
                    : task?.gitspace_config?.instance?.url
                  if (url) window.open(`${url}`, '_blank')
                }}>
                {selectedIde?.buttonText || getString('cde.aiTasks.details.openIde')}
              </Button>

              {!isGitspaceDeleted && (
                <Text
                  onClick={e => {
                    e.preventDefault()
                    e.stopPropagation()
                  }}
                  className={css.optionsMenu}
                  icon={'Options'}
                  tooltip={
                    <Container
                      className={css.listContainer}
                      onClick={e => {
                        e.preventDefault()
                        e.stopPropagation()
                      }}>
                      <Menu>
                        <MenuItem
                          onClick={e => {
                            e.preventDefault()
                            e.stopPropagation()
                            const gitspaceId = task?.gitspace_config?.identifier
                            if (gitspaceId) {
                              history.push(
                                routes.toCDEGitspaceDetail({
                                  space,
                                  gitspaceId
                                })
                              )
                            }
                          }}
                          text={<Text icon="gitspace">{getString('cde.viewGitspace')}</Text>}
                        />
                      </Menu>
                    </Container>
                  }
                  tooltipProps={{
                    interactionKind: PopoverInteractionKind.HOVER,
                    position: Position.BOTTOM,
                    usePortal: true,
                    popoverClassName: css.popover
                  }}
                />
              )}
            </Layout.Horizontal>
          </Container>
        </Layout.Horizontal>
      </Page.SubHeader>

      <Page.Body className={css.pageMain} loading={loading} error={(error as any)?.message}>
        <Container>
          {taskState === TaskStatus.ERROR && (
            <TaskErrorCard
              message={
                task?.error_message ||
                task?.gitspace_config?.instance?.error_message ||
                agentMarkdown ||
                getString('cde.aiTasks.details.errorFallback')
              }
            />
          )}
          <Card className={css.cardContainer}>
            <Text font={{ variation: FontVariation.CARD_TITLE }} className={css.marginLeftContainer}>
              Task Details
            </Text>
            {task?.gitspace_config && (
              <AITaskDetailsCard
                data={task.gitspace_config as TypesGitspaceConfig}
                taskState={task.state}
                standalone={standalone}
              />
            )}
          </Card>

          {task?.ai_usage_metric && (
            <Card className={css.cardContainer}>
              <Text font={{ variation: FontVariation.CARD_TITLE }} className={css.marginLeftContainer}>
                AI Usage Metrics
              </Text>
              <AIUsageMetricsCard metric={task.ai_usage_metric} />
            </Card>
          )}

          <Card className={css.cardContainer}>
            <Container className={css.chatCardBody}>
              <Layout.Vertical className={css.userMessageContainer}>
                <div className={css.messageHeader}>
                  <Avatar size="small" name={userDisplayName} email={userEmail} />
                  <Text>{userDisplayName}</Text>
                </div>
                <Container className={css.message}>
                  <MarkdownText
                    text={task?.initial_prompt || ''}
                    color={Color.WHITE}
                    font={{ variation: FontVariation.SMALL }}
                    className={css.text}
                    markdownClassName={css.markdown}
                    sectionClassName={css.section}
                  />
                </Container>
              </Layout.Vertical>
              {(taskState === TaskStatus.COMPLETED || isWaiting || taskState === TaskStatus.ERROR) && (
                <Layout.Vertical className={css.systemMessageContainer}>
                  <div className={css.messageHeader}>
                    {agentIcon && <img src={agentIcon} alt="Claude icon" className={css.aiIcon} />}
                    <Text>{getString('cde.aiTasks.details.claudeAgentLabel')}</Text>
                  </div>
                  <Container className={css.message}>
                    {shouldShowAgentMarkdown ? (
                      <MarkdownText
                        text={agentMarkdown}
                        color={Color.BLACK}
                        font={{ variation: FontVariation.SMALL }}
                        className={css.text}
                        markdownClassName={css.markdown}
                        sectionClassName={css.section}
                      />
                    ) : isWaiting ? (
                      <Text
                        rightIcon={'loading' as IconName}
                        rightIconProps={{ color: Color.AI_PURPLE_500 }}
                        color={Color.GREY_600}
                        className={css.errorMessage}
                        lineClamp={1}>
                        {waitingText}
                      </Text>
                    ) : (
                      <Text
                        icon="circle-cross"
                        iconProps={{ color: Color.RED_700, size: 18 }}
                        color={Color.GREY_600}
                        className={css.errorMessage}
                        lineClamp={1}>
                        {getString('cde.aiTasks.details.responseError')}
                      </Text>
                    )}
                  </Container>
                </Layout.Vertical>
              )}
            </Container>
          </Card>
        </Container>
      </Page.Body>
    </>
  )
}

export default AITaskDetails
