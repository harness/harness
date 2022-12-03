import React from 'react'
import { Container, PageBody, Text, FontVariation, Tabs } from '@harness/uicore'
import { useGet } from 'restful-react'
import { useAppContext } from 'AppContext'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { useStrings } from 'framework/strings'
import { RepositoryPageHeader } from 'components/RepositoryPageHeader/RepositoryPageHeader'
import { getErrorMessage } from 'utils/Utils'
import type { PullRequestResponse } from 'utils/types'
import { PullRequestMetadataInfo } from './PullRequestMetadataInfo'
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
        {!!repoMetadata && !!prData && <PullRequestMetadataInfo repoMetadata={repoMetadata} {...prData} />}

        {!!prData && (
          <Container className={css.tabsContainer}>
            <Tabs
              id="pullRequestTabs"
              defaultSelectedTabId={'conversation'}
              large={false}
              tabList={[
                {
                  id: 'conversation',
                  title: getString('conversation'),
                  panel: <div>conversation</div>
                },
                {
                  id: 'commits',
                  title: getString('commits'),
                  panel: <div>Commit</div>
                },
                {
                  id: 'diff',
                  title: getString('diff'),
                  panel: <div>Diff</div>
                }
              ]}
            />
          </Container>
        )}
        {/* <pre>{JSON.stringify(prData || {}, null, 2)}</pre> */}
      </PageBody>
    </Container>
  )
}

const PullRequestTitle: React.FC<PullRequestResponse> = ({ title, number }) => (
  <Text tag="h1" font={{ variation: FontVariation.H4 }}>
    {title} <span className={css.prNumber}>#{number}</span>
  </Text>
)
