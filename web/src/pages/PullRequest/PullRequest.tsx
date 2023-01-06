import React, { useMemo } from 'react'
import { Container, PageBody, Text, FontVariation, Tabs, IconName, HarnessIcons } from '@harness/uicore'
import { useGet } from 'restful-react'
import { useHistory } from 'react-router-dom'
import { useAppContext } from 'AppContext'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { useStrings } from 'framework/strings'
import { RepositoryPageHeader } from 'components/RepositoryPageHeader/RepositoryPageHeader'
import { getErrorMessage } from 'utils/Utils'
import { CodeIcon } from 'utils/GitUtils'
import type { TypesPullReq } from 'services/code'
import { PullRequestMetaLine } from './PullRequestMetaLine'
import { Conversation } from './Conversation/Conversation'
import { Changes } from './Changes/Changes'
import { PullRequestCommits } from './PullRequestCommits/PullRequestCommits'
import css from './PullRequest.module.scss'

enum PullRequestSection {
  CONVERSATION = 'conversation',
  COMMITS = 'commits',
  FILES_CHANGED = 'changes'
}

export default function PullRequest() {
  const history = useHistory()
  const { getString } = useStrings()
  const { routes } = useAppContext()
  const {
    repoMetadata,
    error,
    loading,
    refetch,
    pullRequestId,
    pullRequestSection = PullRequestSection.CONVERSATION
  } = useGetRepositoryMetadata()
  const {
    data: prData,
    error: prError,
    loading: prLoading,
    refetch: refetchPullRequest
  } = useGet<TypesPullReq>({
    path: `/api/v1/repos/${repoMetadata?.path}/+/pullreq/${pullRequestId}`,
    lazy: !repoMetadata
  })
  const activeTab = useMemo(
    () =>
      Object.values(PullRequestSection).find(value => value === pullRequestSection)
        ? pullRequestSection
        : PullRequestSection.CONVERSATION,
    [pullRequestSection]
  )

  return (
    <Container className={css.main}>
      <RepositoryPageHeader
        repoMetadata={repoMetadata}
        title={prData ? <PullRequestTitle {...prData} /> : ''}
        dataTooltipId="repositoryPullRequests"
        extraBreadcrumbLinks={
          repoMetadata && [
            {
              label: getString('pullRequests'),
              url: routes.toCODEPullRequests({ repoPath: repoMetadata.path as string })
            }
          ]
        }
      />
      <PageBody loading={loading || prLoading} error={getErrorMessage(error || prError)} retryOnError={() => refetch()}>
        {repoMetadata ? (
          prData ? (
            <>
              <PullRequestMetaLine repoMetadata={repoMetadata} {...prData} />
              <Container className={css.tabsContainer}>
                <Tabs
                  id="prTabs"
                  defaultSelectedTabId={activeTab}
                  large={false}
                  onChange={tabId => {
                    // PR metadata can be changed from conversation tab, refetch to get latest
                    // when the tab is activated
                    if (tabId === PullRequestSection.CONVERSATION) {
                      refetchPullRequest()
                    }
                    history.replace(
                      routes.toCODEPullRequest({
                        repoPath: repoMetadata.path as string,
                        pullRequestId,
                        pullRequestSection: tabId !== PullRequestSection.CONVERSATION ? (tabId as string) : undefined
                      })
                    )
                  }}
                  tabList={[
                    {
                      id: PullRequestSection.CONVERSATION,
                      title: <TabTitle icon={CodeIcon.Chat} title={getString('conversation')} count={0} />,
                      panel: <Conversation repoMetadata={repoMetadata} pullRequestMetadata={prData} />
                    },
                    {
                      id: PullRequestSection.COMMITS,
                      title: <TabTitle icon={CodeIcon.Commit} title={getString('commits')} count={0} />,
                      panel: <PullRequestCommits repoMetadata={repoMetadata} pullRequestMetadata={prData} />
                    },
                    {
                      id: PullRequestSection.FILES_CHANGED,
                      title: <TabTitle icon={CodeIcon.File} title={getString('filesChanged')} count={0} />,
                      panel: <Changes repoMetadata={repoMetadata} pullRequestMetadata={prData} />
                    }
                  ]}
                />
              </Container>
            </>
          ) : null
        ) : null}
      </PageBody>
    </Container>
  )
}

const PullRequestTitle: React.FC<TypesPullReq> = ({ title, number }) => (
  <Text tag="h1" font={{ variation: FontVariation.H4 }}>
    {title} <span className={css.prNumber}>#{number}</span>
  </Text>
)

const TabTitle: React.FC<{ icon: IconName; title: string; count?: number }> = ({ icon, title, count }) => {
  // Icon inside a tab got overriden-looked-bad styles from UICore
  // on hover. Use icon directly instead
  const TabIcon: React.ElementType = HarnessIcons[icon]

  return (
    <Text className={css.tabTitle}>
      <TabIcon width={16} height={16} />
      {title}
      {!!count && (
        <Text inline className={css.count}>
          {count}
        </Text>
      )}
    </Text>
  )
}
