import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { Falsy, Match, Render, Truthy } from 'react-jsx-match'
import { CheckCircle, NavArrowRight } from 'iconoir-react'
import { get, sortBy } from 'lodash-es'
import cx from 'classnames'
import { useHistory } from 'react-router-dom'
import { Container, Layout, Text, FlexExpander, Utils, Button, ButtonVariation, ButtonSize } from '@harnessio/uicore'
import { Icon } from '@harnessio/icons'
import { Color, FontVariation } from '@harnessio/design-system'
import { LogViewer, TermRefs } from 'components/LogViewer/LogViewer'
import { ButtonRoleProps, PullRequestCheckType, PullRequestSection, timeDistance } from 'utils/Utils'
import type { GitInfoProps } from 'utils/GitUtils'
import { useAppContext } from 'AppContext'
import { useQueryParams } from 'hooks/useQueryParams'
import { useStrings } from 'framework/strings'
import { Split } from 'components/Split/Split'
import { MarkdownViewer } from 'components/MarkdownViewer/MarkdownViewer'
import type { PRChecksDecisionResult } from 'hooks/usePRChecksDecision'
import type { TypesCheck, TypesStage } from 'services/code'
import { ExecutionState, ExecutionStatus } from 'components/ExecutionStatus/ExecutionStatus'
import { CheckPipelineStages } from './CheckPipelineStages'
import { findDefaultExecution } from './ChecksUtils'
import { CheckPipelineSteps } from './CheckPipelineSteps'
import css from './Checks.module.scss'

interface ChecksProps extends Pick<GitInfoProps, 'repoMetadata' | 'pullRequestMetadata'> {
  prChecksDecisionResult?: PRChecksDecisionResult
}

export const Checks: React.FC<ChecksProps> = ({ repoMetadata, pullRequestMetadata, prChecksDecisionResult }) => {
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
        repoPath: repoMetadata?.path as string,
        pipeline: selectedItemData?.uid as string,
        execution: get(selectedItemData, 'payload.data.execution_number', '')
      })
    } else {
      return selectedItemData?.link
    }
  }, [repoMetadata?.path, routes, selectedItemData, selectedStage])

  if (!prChecksDecisionResult) {
    return null
  }

  return (
    <Container className={css.main}>
      <Match expr={prChecksDecisionResult?.overallStatus}>
        <Truthy>
          <Split
            split="vertical"
            size={'calc(100% - 400px)'}
            minSize={'calc(100% - 300px)'}
            maxSize={'calc(100% - 600px)'}
            onDragFinished={onSplitPaneResized}
            primary="second">
            <ChecksMenu
              repoMetadata={repoMetadata}
              pullRequestMetadata={pullRequestMetadata}
              prChecksDecisionResult={prChecksDecisionResult}
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
                        <CheckPipelineSteps
                          repoMetadata={repoMetadata}
                          pullRequestMetadata={pullRequestMetadata}
                          pipelineName={selectedItemData?.uid as string}
                          stage={selectedStage as TypesStage}
                          executionNumber={get(selectedItemData, 'payload.data.execution_number', '')}
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
      const defaultSelectedItem = findDefaultExecution(prChecksDecisionResult?.data)

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
      {sortBy(prChecksDecisionResult?.data || [], ['uid'])?.map(itemData => (
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
          setSelectedStage={stage => {
            setSelectedStage(stage)
            setSelectedStageFromProps(stage)
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
  setSelectedStage: (stage: TypesStage | null) => void
}

const CheckMenuItem: React.FC<CheckMenuItemProps> = ({
  isPipeline,
  isSelected = false,
  itemData,
  onClick,
  repoMetadata,
  pullRequestMetadata,
  setSelectedStage
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
        <CheckPipelineStages
          pipelineName={itemData.uid as string}
          executionNumber={get(itemData, 'payload.data.execution_number', '')}
          expanded={expanded}
          repoMetadata={repoMetadata}
          pullRequestMetadata={pullRequestMetadata}
          onSelectStage={setSelectedStage}
        />
      </Render>
    </Container>
  )
}
