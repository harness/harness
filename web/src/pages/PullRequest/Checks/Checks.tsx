import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { Falsy, Match, Render, Truthy } from 'react-jsx-match'
import { CheckCircle, NavArrowRight } from 'iconoir-react'
import { get } from 'lodash-es'
import cx from 'classnames'
import { useGet } from 'restful-react'
import { useHistory } from 'react-router-dom'
import {
  Container,
  Layout,
  Text,
  FlexExpander,
  Utils,
  Button,
  ButtonVariation,
  ButtonSize,
  useToaster
} from '@harnessio/uicore'
import { Icon } from '@harnessio/icons'
import { Color, FontVariation } from '@harnessio/design-system'
import { LogViewer, TermRefs } from 'components/LogViewer/LogViewer'
import { ButtonRoleProps, getErrorMessage, PullRequestCheckType, PullRequestSection, timeDistance } from 'utils/Utils'
import type { GitInfoProps } from 'utils/GitUtils'
import { useAppContext } from 'AppContext'
import { useQueryParams } from 'hooks/useQueryParams'
import { useStrings } from 'framework/strings'
import { Split } from 'components/Split/Split'
import { MarkdownViewer } from 'components/MarkdownViewer/MarkdownViewer'
import type { PRChecksDecisionResult } from 'hooks/usePRChecksDecision'
import type { LivelogLine, TypesCheck, TypesExecution, TypesStage, TypesStep } from 'services/code'
import { ExecutionState, ExecutionStatus } from 'components/ExecutionStatus/ExecutionStatus'
import { useShowRequestError } from 'hooks/useShowRequestError'
// import drone from './drone.svg'
import css from './Checks.module.scss'

interface ChecksProps extends Pick<GitInfoProps, 'repoMetadata' | 'pullRequestMetadata'> {
  prChecksDecisionResult?: PRChecksDecisionResult
}

export const Checks: React.FC<ChecksProps> = props => {
  const { getString } = useStrings()
  const history = useHistory()
  const { routes } = useAppContext()
  const termRefs = useRef<TermRefs>()
  const onSplitPaneResized = useCallback(() => termRefs.current?.fitAddon?.fit(), [])
  const [selectedItemData, setSelectedItemData] = useState<TypesCheck>()
  const [selectedStage, setSelectedStage] = useState<TypesStage | null>(null)
  const isCheckDataMarkdown = useMemo(
    () => selectedItemData?.payload?.kind === PullRequestCheckType.MARKDOWN,
    [selectedItemData?.payload?.kind]
  )
  const logContent = useMemo(
    () => get(selectedItemData, 'payload.data.details', selectedItemData?.summary || ''),
    [selectedItemData]
  )
  const executionLink = useMemo(() => {
    if (selectedStage) {
      return routes.toCODEExecution({
        repoPath: props.repoMetadata?.path as string,
        pipeline: selectedItemData?.uid as string,
        execution: get(selectedItemData, 'payload.data.execution_number', '')
      })
    } else {
      return selectedItemData?.link
    }
  }, [props.repoMetadata?.path, routes, selectedItemData, selectedStage])

  if (!props.prChecksDecisionResult) {
    return null
  }

  return (
    <Container className={css.main}>
      <Match expr={props.prChecksDecisionResult?.overallStatus}>
        <Truthy>
          <Split
            split="vertical"
            size="calc(100% - 400px)"
            minSize={800}
            maxSize="calc(100% - 900px)"
            onDragFinished={onSplitPaneResized}
            primary="second">
            <ChecksMenu
              {...props}
              onDataItemChanged={data => {
                setTimeout(() => setSelectedItemData(data), 0)
              }}
              setSelectedStage={setSelectedStage}
            />
            <Container
              className={cx(css.content, {
                [css.markdown]: isCheckDataMarkdown,
                [css.terminal]: !isCheckDataMarkdown
              })}>
              <Render when={selectedItemData}>
                <Container className={css.header}>
                  <Layout.Horizontal className={css.headerLayout} spacing="small">
                    <ExecutionStatus
                      className={cx(css.status, {
                        [css.invert]: selectedItemData?.status === ExecutionState.PENDING
                      })}
                      status={selectedItemData?.status as ExecutionState}
                      iconSize={20}
                      noBackground
                      iconOnly
                    />
                    <Text font={{ variation: FontVariation.BODY1 }} color={Color.WHITE}>
                      {selectedItemData?.uid}
                      {selectedStage ? ` / ${selectedStage.name}` : ''}
                    </Text>
                    <FlexExpander />
                    <Render when={executionLink}>
                      <Button
                        className={css.noShrink}
                        text={getString('prChecks.viewExternal')}
                        rightIcon="chevron-right"
                        variation={ButtonVariation.SECONDARY}
                        size={ButtonSize.SMALL}
                        onClick={() => {
                          if (selectedStage) {
                            history.push(executionLink as string)
                          } else {
                            window.open(executionLink, '_blank')
                          }
                        }}
                      />
                    </Render>
                  </Layout.Horizontal>
                </Container>
              </Render>
              <Match expr={isCheckDataMarkdown}>
                <Truthy>
                  <Container className={css.markdownContainer}>
                    <MarkdownViewer darkMode source={logContent} />
                  </Container>
                </Truthy>
                <Falsy>
                  <Container className={css.terminalContainer}>
                    <Match expr={selectedStage}>
                      <Truthy>
                        <PipelineSteps
                          itemData={selectedItemData as TypesCheck}
                          repoMetadata={props.repoMetadata}
                          stage={selectedStage as TypesStage}
                        />
                      </Truthy>
                      <Falsy>
                        <LogViewer termRefs={termRefs} content={logContent} />
                      </Falsy>
                    </Match>
                  </Container>
                </Falsy>
              </Match>
            </Container>
          </Split>
        </Truthy>
        <Falsy>
          <Container flex={{ align: 'center-center' }} height="90%">
            <Text font={{ variation: FontVariation.BODY1 }}>{getString('prChecks.notFound')}</Text>
          </Container>
        </Falsy>
      </Match>
    </Container>
  )
}

interface ChecksMenuProps extends ChecksProps {
  onDataItemChanged: (itemData: TypesCheck) => void
  setSelectedStage: (stage: TypesStage | null) => void
}

const ChecksMenu: React.FC<ChecksMenuProps> = ({
  repoMetadata,
  pullRequestMetadata,
  prChecksDecisionResult,
  onDataItemChanged,
  setSelectedStage: setSelectedStageFromProps
}) => {
  const { routes } = useAppContext()
  const history = useHistory()
  const { uid } = useQueryParams<{ uid: string }>()
  const [selectedUID, setSelectedUID] = React.useState<string | undefined>()
  const [selectedStage, setSelectedStage] = useState<TypesStage | null>(null)
  useMemo(() => {
    if (selectedUID) {
      const selectedDataItem = prChecksDecisionResult?.data?.find(item => item.uid === selectedUID)
      if (selectedDataItem) {
        onDataItemChanged(selectedDataItem)
      }
    }
  }, [selectedUID, prChecksDecisionResult?.data, onDataItemChanged])

  useEffect(() => {
    if (uid) {
      if (uid !== selectedUID && prChecksDecisionResult?.data?.find(item => item.uid === uid)) {
        setSelectedUID(uid)
      }
    } else {
      // Find and set a default selected item. Order: Error, Failure, Running, Success, Pending
      const defaultSelectedItem =
        prChecksDecisionResult?.data?.find(({ status }) => status === ExecutionState.ERROR) ||
        prChecksDecisionResult?.data?.find(({ status }) => status === ExecutionState.FAILURE) ||
        prChecksDecisionResult?.data?.find(({ status }) => status === ExecutionState.RUNNING) ||
        prChecksDecisionResult?.data?.find(({ status }) => status === ExecutionState.SUCCESS) ||
        prChecksDecisionResult?.data?.find(({ status }) => status === ExecutionState.PENDING) ||
        prChecksDecisionResult?.data?.[0]

      if (defaultSelectedItem) {
        onDataItemChanged(defaultSelectedItem)
        setSelectedUID(defaultSelectedItem.uid)
        history.replace(
          routes.toCODEPullRequest({
            repoPath: repoMetadata.path as string,
            pullRequestId: String(pullRequestMetadata.number),
            pullRequestSection: PullRequestSection.CHECKS
          }) + `?uid=${defaultSelectedItem.uid}${selectedStage ? `&stageId=${selectedStage.name}` : ''}`
        )
      }
    }
  }, [
    uid,
    prChecksDecisionResult?.data,
    selectedUID,
    history,
    routes,
    repoMetadata.path,
    pullRequestMetadata.number,
    onDataItemChanged,
    selectedStage
  ])

  return (
    <Container className={css.menu}>
      {prChecksDecisionResult?.data?.map(itemData => (
        <CheckMenuItem
          repoMetadata={repoMetadata}
          pullRequestMetadata={pullRequestMetadata}
          prChecksDecisionResult={prChecksDecisionResult}
          key={itemData.uid}
          itemData={itemData}
          isPipeline={itemData.payload?.kind === PullRequestCheckType.PIPELINE}
          isSelected={itemData.uid === selectedUID}
          onClick={stage => {
            setSelectedUID(itemData.uid)
            setSelectedStage(stage || null)
            setSelectedStageFromProps(stage || null)

            history.replace(
              routes.toCODEPullRequest({
                repoPath: repoMetadata.path as string,
                pullRequestId: String(pullRequestMetadata.number),
                pullRequestSection: PullRequestSection.CHECKS
              }) + `?uid=${itemData.uid}${stage ? `&stageId=${stage.name}` : ''}`
            )
          }}
        />
      ))}
    </Container>
  )
}

interface CheckMenuItemProps extends ChecksProps {
  isPipeline?: boolean
  isSelected?: boolean
  itemData: TypesCheck
  onClick: (stage?: TypesStage) => void
}

const CheckMenuItem: React.FC<CheckMenuItemProps> = ({
  isPipeline,
  isSelected = false,
  itemData,
  onClick,
  repoMetadata
}) => {
  const [expanded, setExpanded] = useState(isSelected)

  useEffect(() => {
    if (isSelected) {
      setExpanded(isSelected)
    }
  }, [isSelected])

  return (
    <Container className={css.menuItem}>
      <Layout.Horizontal
        spacing="small"
        className={cx(css.layout, { [css.expanded]: expanded, [css.selected]: isSelected })}
        {...ButtonRoleProps}
        onClick={() => {
          if (isPipeline) {
            setExpanded(!expanded)
          } else {
            onClick()
          }
        }}>
        <Match expr={isPipeline}>
          <Truthy>
            <Icon name="pipeline" size={20} />
            {/* <img src={drone} width={20} height={20} /> */}
          </Truthy>
          <Falsy>
            <CheckCircle color={Utils.getRealCSSColor(Color.GREY_500)} className={css.noShrink} />
          </Falsy>
        </Match>

        <Text className={css.uid} lineClamp={1}>
          {itemData.uid}
        </Text>

        <FlexExpander />

        <Text color={Color.GREY_300} font={{ variation: FontVariation.SMALL }} className={css.noShrink}>
          {timeDistance(itemData.updated, itemData.created)}
        </Text>

        <ExecutionStatus
          className={cx(css.status, css.noShrink)}
          status={itemData.status as ExecutionState}
          iconSize={16}
          noBackground
          iconOnly
        />

        <Render when={isPipeline}>
          <NavArrowRight
            color={Utils.getRealCSSColor(Color.GREY_500)}
            className={cx(css.noShrink, css.chevron)}
            strokeWidth="1.5"
          />
        </Render>
      </Layout.Horizontal>

      <Render when={isPipeline}>
        <PipelineStages itemData={itemData} expanded={expanded} repoMetadata={repoMetadata} onClick={onClick} />
      </Render>
    </Container>
  )
}

const PipelineStages: React.FC<
  Pick<CheckMenuItemProps, 'repoMetadata' | 'itemData' | 'onClick'> & { expanded: boolean }
> = ({ itemData, expanded, repoMetadata, onClick }) => {
  const {
    data: execution,
    error,
    loading,
    refetch
  } = useGet<TypesExecution>({
    path: `/api/v1/repos/${repoMetadata?.path}/+/pipelines/${itemData.uid}/executions/${get(
      itemData,
      'payload.data.execution_number'
    )}`,
    lazy: true
  })
  const { uid, stageId } = useQueryParams<{ uid: string; stageId: string }>()
  const stages = useMemo(() => execution?.stages, [execution])
  const [selectedStageName, setSelectedStageName] = useState<string>('')

  useShowRequestError(error)

  useEffect(() => {
    let timeoutId = 0

    if (repoMetadata && expanded) {
      if (!execution) {
        refetch()
      } else {
        if (
          !error &&
          stages?.find(({ status }) => status === ExecutionState.PENDING || status === ExecutionState.RUNNING)
        ) {
          timeoutId = window.setTimeout(refetch, PIPELINE_STAGE_POLLING_INTERVAL)
        }
      }
    }

    return () => {
      window.clearTimeout(timeoutId)
    }
  }, [repoMetadata, expanded, execution, refetch, error, stages])

  useEffect(() => {
    if (stages?.length) {
      if (uid === itemData.uid && !selectedStageName) {
        if (!stageId) {
          setSelectedStageName(stages[0].name as string)
          onClick(stages[0])
        } else {
          const _stage = stages.find(({ name }) => name === stageId)
          if (_stage) {
            setSelectedStageName(_stage.name as string)
            onClick(_stage)
          }
        }
      } else if (uid !== itemData.uid && selectedStageName) {
        setSelectedStageName('')
      }
    }
  }, [stages, selectedStageName, setSelectedStageName, onClick, stageId, uid, itemData.uid])

  useEffect(() => {
    if (stageId && selectedStageName && selectedStageName !== stageId) {
      setSelectedStageName('')
    }
  }, [stageId, selectedStageName])

  return (
    <Container className={cx(css.pipelineStages, { [css.hidden]: !expanded })}>
      <Match expr={loading && !execution}>
        <Truthy>
          <Container className={css.spinner}>
            <Icon name="steps-spinner" size={16} />
          </Container>
        </Truthy>
        <Falsy>
          <>
            {stages?.map(stage => (
              <Layout.Horizontal
                spacing="small"
                key={stage.name}
                className={cx(css.subMenu, { [css.selected]: stage.name === selectedStageName })}
                {...ButtonRoleProps}
                onClick={() => {
                  onClick(stage)
                  setSelectedStageName(stage.name as string)
                }}>
                <ExecutionStatus
                  className={cx(css.status, css.noShrink)}
                  status={stage.status as ExecutionState}
                  iconSize={16}
                  noBackground
                  iconOnly
                />
                <Text color={Color.GREY_800} className={css.text}>
                  {stage.name}
                </Text>
              </Layout.Horizontal>
            ))}
          </>
        </Falsy>
      </Match>
    </Container>
  )
}

const PipelineSteps: React.FC<Pick<CheckMenuItemProps, 'repoMetadata' | 'itemData'> & { stage: TypesStage }> = ({
  itemData,
  stage,
  repoMetadata
}) => {
  return (
    <Container className={cx(css.pipelineSteps)}>
      {stage.steps?.map(step => (
        <PipelineStep
          key={(itemData.uid + ((stage.name as string) + step.name)) as string}
          itemData={itemData}
          repoMetadata={repoMetadata}
          stage={stage}
          step={step}
        />
      ))}
    </Container>
  )
}

const PipelineStep: React.FC<
  Pick<CheckMenuItemProps, 'repoMetadata' | 'itemData'> & { stage: TypesStage; step: TypesStep }
> = ({ itemData, stage, repoMetadata, step }) => {
  const { showError } = useToaster()
  const eventSourceRef = useRef<EventSource | null>(null)
  const [streamingLogs, setStreamingLogs] = useState<LivelogLine[]>([])
  const isRunning = useMemo(() => step.status === ExecutionState.RUNNING, [step])
  const [expanded, setExpanded] = useState(
    isRunning || step.status === ExecutionState.ERROR || step.status === ExecutionState.FAILURE
  )
  const stepLogPath = useMemo(
    () =>
      `/api/v1/repos/${repoMetadata?.path}/+/pipelines/${itemData.uid}/executions/${get(
        itemData,
        'payload.data.execution_number'
      )}/logs/${stage.number}/${step.number}`,
    [itemData, repoMetadata?.path, stage.number, step.number]
  )
  const lazy =
    !expanded || isRunning || step.status === ExecutionState.PENDING || step.status === ExecutionState.SKIPPED
  const {
    data: logs,
    error,
    loading
  } = useGet<LivelogLine[]>({
    path: stepLogPath,
    lazy
  })
  const logContent = useMemo(
    () => ((isRunning ? streamingLogs : logs) || []).map(log => log.out || '').join(''),
    [streamingLogs, logs, isRunning]
  )

  useEffect(() => {
    if (isRunning) {
      if (eventSourceRef.current) {
        eventSourceRef.current.close()
      }

      setStreamingLogs([])
      eventSourceRef.current = new EventSource(`${stepLogPath}/stream`)

      eventSourceRef.current.onmessage = event => {
        try {
          setStreamingLogs(existingLogs => [...existingLogs, JSON.parse(event.data)])
        } catch (exception) {
          showError(getErrorMessage(exception))
          eventSourceRef.current?.close()
          eventSourceRef.current = null
        }
      }
    }

    return () => {
      setStreamingLogs([])
      eventSourceRef.current?.close()
    }
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  useShowRequestError(error)

  return (
    <Container key={step.number}>
      <Layout.Horizontal
        spacing="small"
        className={cx(css.stepHeader, { [css.expanded]: expanded, [css.selected]: expanded })}
        {...ButtonRoleProps}
        onClick={() => {
          setExpanded(!expanded)
        }}>
        <NavArrowRight
          color={Utils.getRealCSSColor(Color.GREY_500)}
          className={cx(css.noShrink, css.chevron)}
          strokeWidth="1.5"
        />

        <ExecutionStatus
          className={cx(css.status, css.noShrink)}
          status={step.status as ExecutionState}
          iconSize={16}
          noBackground
          iconOnly
        />

        <Text className={css.name} lineClamp={1}>
          {step.name}
        </Text>

        <FlexExpander />

        <Render when={loading}>
          <Icon name="steps-spinner" size={16} />
        </Render>

        <Render when={step.started && step.stopped}>
          <Text color={Color.GREY_300} font={{ variation: FontVariation.SMALL }} className={css.noShrink}>
            {timeDistance(step.started, step.stopped)}
          </Text>
        </Render>
      </Layout.Horizontal>
      <Render when={expanded}>
        <Container className={css.stepLogViewer}>
          <Match expr={isRunning}>
            <Truthy>
              {/* Streaming puts too much pressure on xtermjs and cause incorrect row calculation. Using key to force React to create new instance every time there is new data */}
              {[streamingLogs.length].map(len => (
                <LogViewer key={len} content={logContent} autoHeight />
              ))}
            </Truthy>
            <Falsy>
              <LogViewer content={logContent} autoHeight />
            </Falsy>
          </Match>
        </Container>
      </Render>
    </Container>
  )
}

const PIPELINE_STAGE_POLLING_INTERVAL = 5000
