import React from 'react'
import { Container, PageBody, Text, FontVariation, Tabs, IconName } from '@harness/uicore'
import { useGet } from 'restful-react'
import { useAppContext } from 'AppContext'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { useStrings } from 'framework/strings'
import { RepositoryPageHeader } from 'components/RepositoryPageHeader/RepositoryPageHeader'
import { getErrorMessage } from 'utils/Utils'
import type { PullRequestResponse } from 'utils/types'
import { CodeIcon } from 'utils/GitUtils'
import { PullRequestMetadataInfo } from './PullRequestMetadataInfo'
import { PullRequestConversation } from './PullRequestConversation/PullRequestConversation'
import { PullRequestDiff } from './PullRequestDiff/PullRequestDiff'
import { PullRequestCommits } from './PullRequestCommits/PullRequestCommits'
import css from './PullRequest.module.scss'

export default function PullRequest() {
  const { getString } = useStrings()
  const { routes } = useAppContext()
  const { repoMetadata, error, loading, refetch, pullRequestId } = useGetRepositoryMetadata()
  const {
    data: prData,
    error: prError,
    loading: prLoading
  } = useGet<PullRequestResponse>({
    path: `/api/v1/repos/${repoMetadata?.path}/+/pullreq/${pullRequestId}`,
    lazy: !repoMetadata
  })

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
              <PullRequestMetadataInfo repoMetadata={repoMetadata} {...prData} />
              <Container className={css.tabsContainer}>
                <Tabs
                  id="pullRequestTabs"
                  defaultSelectedTabId={'conversation'}
                  large={false}
                  tabList={[
                    {
                      id: 'conversation',
                      title: <TabTitle icon={CodeIcon.Chat} title={getString('conversation')} count={100} />,
                      panel: <PullRequestConversation repoMetadata={repoMetadata} pullRequestMetadata={prData} />
                    },
                    {
                      id: 'commits',
                      title: <TabTitle icon={CodeIcon.Commit} title={getString('commits')} count={15} />,
                      panel: <PullRequestCommits repoMetadata={repoMetadata} pullRequestMetadata={prData} />
                    },
                    {
                      id: 'diff',
                      title: <TabTitle icon={CodeIcon.Commit} title={getString('diff')} count={20} />,
                      panel: <PullRequestDiff repoMetadata={repoMetadata} pullRequestMetadata={prData} />
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

const PullRequestTitle: React.FC<PullRequestResponse> = ({ title, number }) => (
  <Text tag="h1" font={{ variation: FontVariation.H4 }}>
    {title} <span className={css.prNumber}>#{number}</span>
  </Text>
)

const TabTitle: React.FC<{ icon: IconName; title: string; count?: number }> = ({ icon, title, count }) => (
  <Text icon={icon} className={css.tabTitle}>
    {title}{' '}
    {!!count && (
      <Text inline className={css.count}>
        {count}
      </Text>
    )}
  </Text>
)
