import React, { useCallback, useEffect, useMemo, useState } from 'react'
import {
  Container,
  PageBody,
  Text,
  FontVariation,
  Tabs,
  IconName,
  HarnessIcons,
  Layout,
  Button,
  ButtonVariation,
  ButtonSize,
  TextInput,
  useToaster,
  Spacing,
  PaddingProps
} from '@harness/uicore'
import { useGet, useMutate } from 'restful-react'
import { Render, Match, Truthy, Else } from 'react-jsx-match'
import { useHistory } from 'react-router-dom'
import { useAppContext } from 'AppContext'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { useStrings } from 'framework/strings'
import { RepositoryPageHeader } from 'components/RepositoryPageHeader/RepositoryPageHeader'
import { voidFn, getErrorMessage, MergeCheckStatus } from 'utils/Utils'
import { CodeIcon, GitInfoProps } from 'utils/GitUtils'
import type { TypesPullReq, TypesPullReqStats, TypesRepository } from 'services/code'
import { LoadingSpinner } from 'components/LoadingSpinner/LoadingSpinner'
import { PullRequestMetaLine } from './PullRequestMetaLine'
import { Conversation } from './Conversation/Conversation'
import { Checks } from './Checks/Checks'
import { Changes } from '../../components/Changes/Changes'
import { PullRequestCommits } from './PullRequestCommits/PullRequestCommits'
import css from './PullRequest.module.scss'

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
  const path = useMemo(
    () => `/api/v1/repos/${repoMetadata?.path}/+/pullreq/${pullRequestId}`,
    [repoMetadata?.path, pullRequestId]
  )
  const {
    data: pullRequestData,
    error: prError,
    loading: prLoading,
    refetch: refetchPullRequest
  } = useGet<TypesPullReq>({
    path,
    lazy: !repoMetadata
  })
  const [prData, setPrData] = useState<TypesPullReq>()
  const showSpinner = useMemo(() => {
    return loading || (prLoading && !prData)
  }, [loading, prLoading, prData])
  const [stats, setStats] = useState<TypesPullReqStats>()
  const prHasChanged = useMemo(() => {
    if (stats && prData?.stats) {
      if (
        stats.commits !== prData.stats.commits ||
        stats.conversations !== prData.stats.conversations ||
        stats.files_changed !== prData.stats.files_changed
      ) {
        window.setTimeout(() => setStats(prData.stats), 50)
        return true
      }
    }
    return false
  }, [prData?.stats, stats])
  const mergeable = useMemo(() => prData?.merge_check_status === MergeCheckStatus.MERGEABLE, [prData])

  useEffect(
    function setStatsIfNotSet() {
      if (!stats && prData?.stats) {
        setStats(prData.stats)
      }
    },
    [prData?.stats, stats]
  )

  // prData holds the latest good PR data to make sure page is not broken
  // when polling fails
  useEffect(
    function setPrDataIfNotSet() {
      if (pullRequestData) {
        setPrData(pullRequestData)
      }
    },
    [pullRequestData]
  )

  useEffect(() => {
    const fn = () => {
      if (repoMetadata) {
        refetchPullRequest().then(() => {
          interval = window.setTimeout(fn, mergeable ? PR_POLLING_INTERVAL : PR_POLLING_INTERVAL_WHEN_NOT_MERGEABLE)
        })
      }
    }
    let interval = window.setTimeout(fn, mergeable ? PR_POLLING_INTERVAL : PR_POLLING_INTERVAL_WHEN_NOT_MERGEABLE)

    return () => window.clearTimeout(interval)
  }, [repoMetadata, refetchPullRequest, path, mergeable])

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
        title={repoMetadata && prData ? <PullRequestTitle repoMetadata={repoMetadata} {...prData} /> : ''}
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
      <PageBody error={!prData && getErrorMessage(error || prError)} retryOnError={voidFn(refetch)}>
        <LoadingSpinner visible={showSpinner} />

        <Render when={repoMetadata && prData}>
          <>
            <PullRequestMetaLine repoMetadata={repoMetadata as TypesRepository} {...prData} />
            <Container className={css.tabsContainer}>
              <Tabs
                id="prTabs"
                defaultSelectedTabId={activeTab}
                large={false}
                onChange={tabId => {
                  history.replace(
                    routes.toCODEPullRequest({
                      repoPath: repoMetadata?.path as string,
                      pullRequestId,
                      pullRequestSection: tabId !== PullRequestSection.CONVERSATION ? (tabId as string) : undefined
                    })
                  )
                }}
                tabList={[
                  {
                    id: PullRequestSection.CONVERSATION,
                    title: (
                      <TabTitle
                        icon={CodeIcon.Chat}
                        title={getString('conversation')}
                        count={prData?.stats?.conversations || 0}
                      />
                    ),
                    panel: (
                      <Conversation
                        repoMetadata={repoMetadata as TypesRepository}
                        pullRequestMetadata={prData as TypesPullReq}
                        onCommentUpdate={voidFn(refetchPullRequest)}
                        prHasChanged={prHasChanged}
                      />
                    )
                  },
                  {
                    id: PullRequestSection.COMMITS,
                    title: (
                      <TabTitle
                        icon={CodeIcon.Commit}
                        title={getString('commits')}
                        count={prData?.stats?.commits || 0}
                        padding={{ left: 'medium' }}
                      />
                    ),
                    panel: (
                      <PullRequestCommits
                        repoMetadata={repoMetadata as TypesRepository}
                        pullRequestMetadata={prData as TypesPullReq}
                        prHasChanged={prHasChanged}
                        handleRefresh={voidFn(refetchPullRequest)}
                      />
                    )
                  },
                  {
                    id: PullRequestSection.FILES_CHANGED,
                    title: (
                      <TabTitle
                        icon={CodeIcon.File}
                        title={getString('filesChanged')}
                        count={prData?.stats?.files_changed || 0}
                        padding={{ left: 'medium' }}
                      />
                    ),
                    panel: (
                      <Container className={css.changes}>
                        <Changes
                          repoMetadata={repoMetadata as TypesRepository}
                          pullRequestMetadata={prData as TypesPullReq}
                          targetBranch={prData?.target_branch}
                          sourceBranch={prData?.source_branch}
                          emptyTitle={getString('noChanges')}
                          emptyMessage={getString('noChangesPR')}
                          onCommentUpdate={voidFn(refetchPullRequest)}
                          prHasChanged={prHasChanged}
                        />
                      </Container>
                    )
                  },
                  {
                    id: PullRequestSection.CHECKS,
                    disabled: window.location.hostname !== 'localhost', // TODO: Remove when API supports checks
                    title: (
                      <TabTitle
                        icon={CodeIcon.ChecksSuccess}
                        title={getString('checks')}
                        count={0} // TODO: Count for checks when API supports it
                        padding={{ left: 'medium' }}
                      />
                    ),
                    panel: <Checks />
                  }
                ]}
              />
            </Container>
          </>
        </Render>
      </PageBody>
    </Container>
  )
}

interface PullRequestTitleProps extends TypesPullReq, Pick<GitInfoProps, 'repoMetadata'> {
  onSaveDone?: (newTitle: string) => Promise<boolean>
}

const PullRequestTitle: React.FC<PullRequestTitleProps> = ({ repoMetadata, title, number, description }) => {
  const [original, setOriginal] = useState(title)
  const [val, setVal] = useState(title)
  const [edit, setEdit] = useState(false)
  const { getString } = useStrings()
  const { showError } = useToaster()
  const { mutate } = useMutate({
    verb: 'PATCH',
    path: `/api/v1/repos/${repoMetadata.path}/+/pullreq/${number}`
  })
  const submitChange = useCallback(() => {
    mutate({
      title: val,
      description
    })
      .then(() => {
        setEdit(false)
        setOriginal(val)
      })
      .catch(exception => showError(getErrorMessage(exception), 0))
  }, [description, val, mutate, showError])

  return (
    <Layout.Horizontal spacing="xsmall" className={css.prTitle}>
      <Match expr={edit}>
        <Truthy>
          <Container>
            <Layout.Horizontal spacing="small">
              <TextInput
                wrapperClassName={css.input}
                value={val}
                onFocus={event => event.target.select()}
                onInput={event => setVal(event.currentTarget.value)}
                autoFocus
                onKeyDown={event => {
                  switch (event.key) {
                    case 'Enter':
                      submitChange()
                      break
                    case 'Escape': // does not work, maybe TextInput cancels ESC?
                      setEdit(false)
                      break
                  }
                }}
              />
              <Button
                variation={ButtonVariation.PRIMARY}
                text={getString('save')}
                size={ButtonSize.MEDIUM}
                disabled={(val || '').trim().length === 0 || title === val}
                onClick={submitChange}
              />
              <Button
                variation={ButtonVariation.TERTIARY}
                text={getString('cancel')}
                size={ButtonSize.MEDIUM}
                onClick={() => setEdit(false)}
              />
            </Layout.Horizontal>
          </Container>
        </Truthy>
        <Else>
          <>
            <Text tag="h1" font={{ variation: FontVariation.H4 }}>
              {original} <span className={css.prNumber}>#{number}</span>
            </Text>
            <Button
              variation={ButtonVariation.ICON}
              tooltip={getString('edit')}
              tooltipProps={{ isDark: true, position: 'right' }}
              size={ButtonSize.SMALL}
              icon="code-edit"
              className={css.btn}
              onClick={() => setEdit(true)}
            />
          </>
        </Else>
      </Match>
    </Layout.Horizontal>
  )
}

const TabTitle: React.FC<{ icon: IconName; title: string; count?: number; padding?: Spacing | PaddingProps }> = ({
  icon,
  title,
  count,
  padding
}) => {
  // Icon inside a tab got overriden-and-looked-bad styles from UICore
  // on hover. Use icon directly instead
  const TabIcon: React.ElementType = HarnessIcons[icon]

  return (
    <Text className={css.tabTitle} padding={padding}>
      <TabIcon width={16} height={16} />
      {title}
      <Render when={count}>
        <Text inline className={css.count}>
          {count}
        </Text>
      </Render>
    </Text>
  )
}

enum PullRequestSection {
  CONVERSATION = 'conversation',
  COMMITS = 'commits',
  FILES_CHANGED = 'changes',
  CHECKS = 'checks'
}

const PR_POLLING_INTERVAL = 15000
const PR_POLLING_INTERVAL_WHEN_NOT_MERGEABLE = 5000
