import React from 'react'
import { Container, Layout, PageBody, Text, Color, Icon, Tabs } from '@harness/uicore'
import { Link } from 'react-router-dom'
import SplitPane from 'react-split-pane'
import { PopoverInteractionKind } from '@blueprintjs/core'
import { useStrings } from 'framework/strings'
import { ButtonRoleProps } from 'utils/Utils'
import { ResourceTree } from './ResourceTree/ResourceTree'
import { ResourceContent } from './ResourceContent/ResourceContent'
import css from './RepoFiles.module.scss'

export default function RepoFiles(): JSX.Element {
  const { getString } = useStrings()

  return (
    <Container className={css.main}>
      <PageBody>
        <Container padding={{ top: 'medium', right: 'xlarge', left: 'xlarge' }} background={Color.WHITE}>
          <Container>
            <Layout.Horizontal spacing="small" className={css.breadcrumb}>
              <Link to="">SCM_Project</Link>
              <Icon name="main-chevron-right" size={10} color={Color.GREY_500} />
              <Link to="">{getString('repositories')}</Link>
              <Icon name="main-chevron-right" size={10} color={Color.GREY_500} />
            </Layout.Horizontal>
            <Container padding={{ top: 'medium', bottom: 'small' }}>
              <Text
                inline
                className={css.repoDropdown}
                icon="git-repo"
                iconProps={{
                  size: 14,
                  color: Color.GREY_500,
                  margin: { right: 'small' }
                }}
                rightIcon="main-chevron-down"
                rightIconProps={{
                  size: 14,
                  color: Color.GREY_500,
                  margin: { left: 'xsmall' }
                }}
                tooltip={<Container padding="xlarge">Enter enter enter enter enter enter enter enter</Container>}
                tooltipProps={{
                  interactionKind: PopoverInteractionKind.CLICK,
                  targetClassName: css.targetClassName,
                  minimal: true
                }}
                {...ButtonRoleProps}>
                Drone CIE
              </Text>
            </Container>
          </Container>
        </Container>
        <Container className={css.tabContainer}>
          <Tabs
            id="repoTabs"
            defaultSelectedTabId={'files'}
            tabList={[
              {
                id: 'files',
                title: getString('files'),
                panel: <Files />,
                iconProps: { name: 'file' },
                disabled: false
              },
              {
                id: 'commits',
                title: getString('commits'),
                panel: <Commits />,
                iconProps: { name: 'git-branch-existing' },
                disabled: true
              },
              {
                id: 'pull-requests',
                title: getString('pullRequests'),
                panel: <PullRequests />,
                iconProps: { name: 'git-pull' },
                disabled: true
              },
              {
                id: 'settings',
                title: getString('settings'),
                panel: <Settings />,
                iconProps: { name: 'cog' },
                disabled: true
              }
            ]}></Tabs>
        </Container>
      </PageBody>
    </Container>
  )
}

const TabContentWrapper: React.FC = ({ children }) => {
  return <Container className={css.tabContent}>{children}</Container>
}

const splitPaneResizerStyle = navigator.userAgent.match(/firefox/i)
  ? { display: 'flow-root list-item' }
  : { display: 'inline-table' }
const splitPaneResizeLocalStorageKey = 'SCM_SplitPaneResizeLocalStorageKey'
const splitPaneDefaultSize = 300
const splitPaneMaxSize = 600

function Files(): JSX.Element {
  const paneDefaultSize = parseInt(localStorage.getItem(splitPaneResizeLocalStorageKey) || '')

  return (
    <TabContentWrapper>
      <SplitPane
        split="vertical"
        minSize={0}
        maxSize={splitPaneMaxSize}
        defaultSize={paneDefaultSize >= 0 ? paneDefaultSize : splitPaneDefaultSize}
        onChange={size => localStorage.setItem(splitPaneResizeLocalStorageKey, String(size))}
        className={css.tabContent}
        resizerStyle={splitPaneResizerStyle}
        pane2Style={{ overflow: 'auto' }}
        allowResize={true}>
        <ResourceTree />
        <ResourceContent />
      </SplitPane>
    </TabContentWrapper>
  )
}

function Commits(): JSX.Element {
  return <TabContentWrapper>TBD</TabContentWrapper>
}

function PullRequests(): JSX.Element {
  return <TabContentWrapper>TBD</TabContentWrapper>
}

function Settings(): JSX.Element {
  return <TabContentWrapper>TBD</TabContentWrapper>
}
