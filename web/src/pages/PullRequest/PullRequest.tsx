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
  useToaster
} from '@harness/uicore'
import { useGet, useMutate } from 'restful-react'
import { Render, Match, Truthy, Else } from 'react-jsx-match'
import { useHistory } from 'react-router-dom'
import { useAppContext } from 'AppContext'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { useStrings } from 'framework/strings'
import { RepositoryPageHeader } from 'components/RepositoryPageHeader/RepositoryPageHeader'
import { voidFn, getErrorMessage, PR_POLLING_LIMIT } from 'utils/Utils'
import { CodeIcon, GitInfoProps } from 'utils/GitUtils'
import type { TypesPullReq, TypesRepository } from 'services/code'
import { LoadingSpinner } from 'components/LoadingSpinner/LoadingSpinner'
import { PullRequestMetaLine } from './PullRequestMetaLine'
import { Conversation } from './Conversation/Conversation'
import { Checks } from './Checks/Checks'
import { Changes } from '../../components/Changes/Changes'
import { PullRequestCommits } from './PullRequestCommits/PullRequestCommits'
import css from './PullRequest.module.scss'

enum PullRequestSection {
  CONVERSATION = 'conversation',
  COMMITS = 'commits',
  FILES_CHANGED = 'changes',
  CHECKS = 'checks'
}

export default function PullRequest() {
  const history = useHistory()
  const { getString } = useStrings()
  const { routes } = useAppContext()
  const [prHasChanged, setPrHasChanged] = useState(false)
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
  const {
    data: pollPrData,
    error: pollPrError,
    refetch: refetchPollPullRequest
  } = useGet<TypesPullReq>({
    path: `/api/v1/repos/${repoMetadata?.path}/+/pullreq/${pullRequestId}`,
    lazy: !repoMetadata
  })
  const [newPrData, setNewPrData] = useState<TypesPullReq>()

  const handleRefresh = () => {
    refetchPullRequest()
    setNewPrData(prData as TypesPullReq)
    setPrHasChanged(false)
  }

  useEffect(() => {
    const interval = window.setTimeout(() => {
      refetchPollPullRequest()
      setNewPrData(pollPrData as TypesPullReq)
    }, PR_POLLING_LIMIT)
    return () => window.clearTimeout(interval)
  }, [pollPrData, refetchPollPullRequest])

  useEffect(() => {
    if (prData?.stats && newPrData?.stats) {
      const prStatsChanged =
        prData.stats.commits !== newPrData.stats.commits || prData.stats.files_changed !== newPrData.stats.files_changed
      if (prStatsChanged) {
        setPrHasChanged(prStatsChanged)
      }
    }
  }, [newPrData, refetchPollPullRequest, prData?.stats])

  const activeTab = useMemo(
    () =>
      Object.values(PullRequestSection).find(value => value === pullRequestSection)
        ? pullRequestSection
        : PullRequestSection.CONVERSATION,
    [pullRequestSection]
  )
  // /repos/${paramsInPath.repo_ref}/pullreq/${paramsInPath.pullreq_number}/metadata

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
      <PageBody error={getErrorMessage(error || prError || pollPrError)} retryOnError={voidFn(refetch)}>
        <LoadingSpinner visible={loading || prLoading} withBorder={!!prData && prLoading} />

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
                        handleRefresh={voidFn(handleRefresh)}
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
                      />
                    ),
                    panel: (
                      <PullRequestCommits
                        repoMetadata={repoMetadata as TypesRepository}
                        pullRequestMetadata={prData as TypesPullReq}
                        prHasChanged={prHasChanged}
                        handleRefresh={voidFn(handleRefresh)}
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
                          handleRefresh={voidFn(handleRefresh)}
                        />
                      </Container>
                    )
                  },
                  {
                    id: PullRequestSection.CHECKS,
                    disabled: true,
                    title: (
                      <TabTitle
                        icon={CodeIcon.ChecksSuccess}
                        title={getString('checks')}
                        count={0} // TODO: Count for checks when API supports it
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

const TabTitle: React.FC<{ icon: IconName; title: string; count?: number }> = ({ icon, title, count }) => {
  // Icon inside a tab got overriden-and-looked-bad styles from UICore
  // on hover. Use icon directly instead
  const TabIcon: React.ElementType = HarnessIcons[icon]

  return (
    <Text className={css.tabTitle}>
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
