import React, { useCallback, useRef } from 'react'
import { Render } from 'react-jsx-match'
import SplitPane from 'react-split-pane'
import cx from 'classnames'
import { Container, Layout, Text, Color, FlexExpander, Icon, useToggle } from '@harness/uicore'
import { LogViewer, TermRefs } from 'components/LogViewer/LogViewer'
import { ButtonRoleProps } from 'utils/Utils'
import css from './Checks.module.scss'

// interface ChecksProps {}

export function Checks() {
  const termRefs = useRef<TermRefs>()
  const onSplitPaneResized = useCallback(() => termRefs.current?.fitAddon?.fit(), [])

  return (
    <Container className={css.main}>
      <SplitPane
        split="vertical"
        size="calc(100% - 400px)"
        minSize={800}
        maxSize="calc(100% - 900px)"
        onDragFinished={onSplitPaneResized}
        primary="second">
        <StagesContainer />
        <Container className={css.terminalContainer}>
          <LogViewer termRefs={termRefs} content={`...`} />
        </Container>
      </SplitPane>
    </Container>
  )
}

const StagesContainer: React.FC = () => {
  return (
    <Container className={css.stagesContainer}>
      <Container>
        <Layout.Horizontal className={css.stagesHeader}>
          <Container>
            <Layout.Horizontal spacing="xsmall">
              <span data-stage-state="success" />
              <span data-stage-state="failed" />
              <span data-stage-state="success" />
              <span data-stage-state="running" />
              <span data-stage-state="pending" />
            </Layout.Horizontal>
          </Container>
          <FlexExpander />
          <Text color={Color.GREY_400}>5 stages</Text>
        </Layout.Horizontal>
      </Container>
      <Container>
        {['Mandatory PreRequisite', 'JiraCreateApprove', 'ui', 'JiraClosure'].map(title => (
          <StageSection key={title} title={title} isExpanded={title === 'ui' || title === 'Mandatory PreRequisite'} />
        ))}
      </Container>
    </Container>
  )
}

interface StageSectionProps {
  title: string
  isExpanded?: boolean
}

const StageSection: React.FC<StageSectionProps> = ({ title, isExpanded = false }) => {
  const [expanded, toogleExpanded] = useToggle(isExpanded)

  return (
    <Container key={title} className={cx(css.stageSection, { [css.expanded]: expanded })}>
      <Layout.Horizontal spacing="small" className={css.sectionName} {...ButtonRoleProps} onClick={toogleExpanded}>
        <Icon className={css.chevron} name="chevron-right" size={16} color={Color.GREY_800} />
        <Icon name="tick-circle" size={16} color={Color.GREEN_500} />
        <Text color={Color.GREY_800}>
          <strong>{title}</strong>
        </Text>
        <FlexExpander />
        <Text color={Color.GREY_400}>1:20</Text>
      </Layout.Horizontal>
      <Render when={expanded && title === 'Mandatory PreRequisite'}>
        {['Skip UAT for Ingress Upgrade', 'Change List', 'Manager Approval'].map(name => (
          <Layout.Horizontal spacing="small" key={name} className={css.stageSectionDetails} {...ButtonRoleProps}>
            <Icon name="tick-circle" size={16} color={Color.GREEN_500} />
            <Text color={Color.GREY_800} className={css.text}>
              {name}
            </Text>
          </Layout.Horizontal>
        ))}
      </Render>
      <Render when={expanded && title === 'ui'}>
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
          <Layout.Horizontal
            spacing="small"
            key={name}
            className={cx(css.stageSectionDetails, { [css.active]: name === 'Failover' })}
            {...ButtonRoleProps}>
            <Icon name="tick-circle" size={16} color={Color.GREEN_500} />
            <Text color={Color.GREY_800} className={css.text}>
              {name}
            </Text>
          </Layout.Horizontal>
        ))}
      </Render>
    </Container>
  )
}
