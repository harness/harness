import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { Falsy, Match, Render, Truthy } from 'react-jsx-match'
import { /*CheckCircle,*/ NavArrowRight } from 'iconoir-react'
import { get } from 'lodash-es'
import cx from 'classnames'
import { useHistory } from 'react-router-dom'
import {
  Container,
  Layout,
  Text,
  FlexExpander,
  useToggle,
  Utils,
  Button,
  ButtonVariation,
  ButtonSize
} from '@harnessio/uicore'
// import { Icon } from '@harnessio/icons'
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
import type { TypesCheck } from 'services/code'
import { ExecutionState, ExecutionStatus } from 'components/ExecutionStatus/ExecutionStatus'
import css from './Checks.module.scss'

interface ChecksProps extends Pick<GitInfoProps, 'repoMetadata' | 'pullRequestMetadata'> {
  prChecksDecisionResult?: PRChecksDecisionResult
}

export const Checks: React.FC<ChecksProps> = props => {
  const { getString } = useStrings()
  const termRefs = useRef<TermRefs>()
  const onSplitPaneResized = useCallback(() => termRefs.current?.fitAddon?.fit(), [])
  const [selectedItemData, setSelectedItemData] = useState<TypesCheck>()
  const isCheckDataMarkdown = useMemo(
    () => selectedItemData?.payload?.kind === PullRequestCheckType.MARKDOWN,
    [selectedItemData?.payload?.kind]
  )
  const logContent = useMemo(
    () => get(selectedItemData, 'payload.data.details', selectedItemData?.summary || ''),
    [selectedItemData]
  )

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
                    </Text>
                    <FlexExpander />
                    <Render when={selectedItemData?.link}>
                      <Button
                        className={css.noShrink}
                        text={getString('prChecks.viewExternal')}
                        rightIcon="chevron-right"
                        variation={ButtonVariation.SECONDARY}
                        size={ButtonSize.SMALL}
                        onClick={() => {
                          window.open(selectedItemData?.link, '_blank')
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
                    <LogViewer termRefs={termRefs} content={logContent} />
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
}

const ChecksMenu: React.FC<ChecksMenuProps> = ({
  repoMetadata,
  pullRequestMetadata,
  prChecksDecisionResult,
  onDataItemChanged
}) => {
  const { routes } = useAppContext()
  const history = useHistory()
  const { uid } = useQueryParams<{ uid: string }>()
  const [selectedUID, setSelectedUID] = React.useState<string | undefined>()

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
          }) + `?uid=${defaultSelectedItem.uid}`
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
    onDataItemChanged
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
          expandable={false}
          isSelected={itemData.uid === selectedUID}
          onClick={() => {
            history.replace(
              routes.toCODEPullRequest({
                repoPath: repoMetadata.path as string,
                pullRequestId: String(pullRequestMetadata.number),
                pullRequestSection: PullRequestSection.CHECKS
              }) + `?uid=${itemData.uid}`
            )
            setSelectedUID(itemData.uid)
          }}
        />
      ))}
    </Container>
  )
}

interface CheckMenuItemProps extends ChecksProps {
  expandable?: boolean
  isSelected?: boolean
  itemData: TypesCheck
  onClick: () => void
}

const CheckMenuItem: React.FC<CheckMenuItemProps> = ({ expandable, isSelected = false, itemData, onClick }) => {
  const [expanded, toogleExpanded] = useToggle(isSelected)

  return (
    <Container className={css.menuItem}>
      <Layout.Horizontal
        spacing="small"
        className={cx(css.layout, { [css.expanded]: expanded, [css.selected]: isSelected })}
        {...ButtonRoleProps}
        onClick={expandable ? toogleExpanded : onClick}>
        <Render when={expandable}>
          <NavArrowRight color={Utils.getRealCSSColor(Color.GREY_500)} className={cx(css.noShrink, css.chevron)} />
        </Render>

        {/*
        TODO: This is reserved for future Pipeline implementation reference. Needs
        a couple of things:
          - persist selected pipeline stage in URL
          - onClick back to the Menu
          - Custom rendering for pipeline stages
       */}
        {/* <Match expr={expandable}>
          <Truthy>
            <Icon name="ci-main" size={16} />
          </Truthy>
          <Falsy>
            <CheckCircle color={Utils.getRealCSSColor(Color.GREY_500)} className={css.noShrink} />
          </Falsy>
        </Match>
 */}
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
      </Layout.Horizontal>

      {/*
        TODO: This is reserved for future Pipeline implementation reference. Needs
        a couple of things:
          - persist selected pipeline stage in URL
          - onClick back to the Menu
          - Custom rendering for pipeline stages
       */}
      {/* <Render when={expanded && itemData.payload?.kind === PullRequestCheckType.PIPELINE}>
        {[
          'Service',
          'Infrastructure',
          'Resource Constraint',
          'Get_Deployed_Ver_Swarmia',
          'Rollout Deployment',
          'Post Version to Swarmia',
          'Slack_Notify',
          'Failover'
        ].map(name => (
          <Layout.Horizontal spacing="small" key={name} className={css.subMenu} {...ButtonRoleProps}>
            <Icon name="tick-circle" size={16} color={Color.GREEN_500} />
            <Text color={Color.GREY_800} className={css.text}>
              {name}
            </Text>
          </Layout.Horizontal>
        ))}
      </Render> */}
    </Container>
  )
}
