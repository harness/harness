import { noop } from 'lodash-es'
import React, { useCallback, useEffect, useMemo, useState } from 'react'
import {
  Container,
  PageBody,
  NoDataCard,
  Tabs,
  Layout,
  TextInput,
  Text,
  FontVariation,
  useToaster,
  Color,
  StringSubstitute
} from '@harness/uicore'
import cx from 'classnames'
import { useHistory } from 'react-router-dom'
import { useGet, useMutate } from 'restful-react'
import { useAppContext } from 'AppContext'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { useStrings } from 'framework/strings'
import { RepositoryPageHeader } from 'components/RepositoryPageHeader/RepositoryPageHeader'
import { voidFn, getErrorMessage, LIST_FETCHING_LIMIT, showToaster } from 'utils/Utils'
import { Images } from 'images'
import { CodeIcon, isRefATag, makeDiffRefs } from 'utils/GitUtils'
import { CommitsView } from 'components/CommitsView/CommitsView'
import { Changes } from 'components/Changes/Changes'
import type { OpenapiCreatePullReqRequest, TypesCommit, TypesPullReq } from 'services/code'
import { LoadingSpinner } from 'components/LoadingSpinner/LoadingSpinner'
import { usePageIndex } from 'hooks/usePageIndex'
import { ResourceListingPagination } from 'components/ResourceListingPagination/ResourceListingPagination'
import { TabTitleWithCount, tabContainerCSS } from 'components/TabTitleWithCount/TabTitleWithCount'
import type { DiffFileEntry } from 'utils/types'
import { MarkdownEditorWithPreview } from 'components/MarkdownEditorWithPreview/MarkdownEditorWithPreview'
import { CompareContentHeader, PRCreationType } from './CompareContentHeader/CompareContentHeader'
import css from './Compare.module.scss'

export default function Compare() {
  const { getString } = useStrings()
  const history = useHistory()
  const { routes } = useAppContext()
  const { repoMetadata, error, loading, diffRefs } = useGetRepositoryMetadata()
  const [sourceGitRef, setSourceGitRef] = useState(diffRefs.sourceGitRef)
  const [targetGitRef, setTargetGitRef] = useState(diffRefs.targetGitRef)
  const [page, setPage] = usePageIndex()
  const limit = LIST_FETCHING_LIMIT
  const {
    data: commits,
    error: commitsError,
    refetch,
    response
  } = useGet<{
    commits: TypesCommit[]
  }>({
    path: `/api/v1/repos/${repoMetadata?.path}/+/commits`,
    queryParams: {
      limit,
      page,
      git_ref: sourceGitRef,
      after: targetGitRef
    },
    lazy: !repoMetadata
  })
  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [diffs, setDiffs] = useState<DiffFileEntry[]>([])
  const { showError } = useToaster()
  const { mutate: createPullRequest, loading: creatingInProgress } = useMutate<TypesPullReq>({
    verb: 'POST',
    path: `/api/v1/repos/${repoMetadata?.path}/+/pullreq`
  })
  const onCreatePullRequest = useCallback(
    (creationType: PRCreationType) => {
      if (!sourceGitRef || !targetGitRef) {
        return showToaster(getString('prMustSelectSourceAndTargetBranches'))
      }

      if (isRefATag(sourceGitRef) || isRefATag(targetGitRef)) {
        return showToaster(getString('pullMustBeMadeFromBranches'))
      }

      if (sourceGitRef === targetGitRef) {
        return showToaster(getString('prSourceAndTargetMustBeDifferent'))
      }

      if (!title) {
        return showToaster(getString('pr.titleIsRequired'))
      }

      const pullReqUrl = window.location.href.split('compare')?.[0]
      const payload: OpenapiCreatePullReqRequest = {
        target_branch: targetGitRef,
        source_branch: sourceGitRef,
        title: title,
        description: description || '',
        is_draft: creationType === PRCreationType.DRAFT
      }

      try {
        createPullRequest(payload)
          .then(data => {
            history.replace(
              routes.toCODEPullRequest({
                repoPath: repoMetadata?.path as string,
                pullRequestId: String(data.number)
              })
            )
          })
          .catch(_error => {
            if (_error.status === 409) {
              showError(
                <StringSubstitute
                  str={getString('pullRequestalreadyExists')}
                  vars={{
                    prLink: (
                      <a
                        className={css.hyperlink}
                        color={Color.PRIMARY_7}
                        href={`${pullReqUrl}${_error.data.values.number}`}>
                        {` #${_error.data.values.number} ${_error.data.values.title} `}
                      </a>
                    )
                  }}
                />
              )
            } else {
              showError(getErrorMessage(_error), 0, 'pr.failedToCreate')
            }
          })
      } catch (exception) {
        showError(getErrorMessage(exception), 0, 'pr.failedToCreate')
      }
    },
    [
      createPullRequest,
      description,
      showError,
      sourceGitRef,
      getString,
      targetGitRef,
      title,
      history,
      routes,
      repoMetadata
    ]
  )
  const ChangesTab = useMemo(() => {
    if (repoMetadata) {
      return (
        <Container className={css.changesContainer}>
          <Changes
            readOnly
            repoMetadata={repoMetadata}
            targetRef={targetGitRef}
            sourceRef={sourceGitRef}
            emptyTitle={getString('noChanges')}
            emptyMessage={getString('noChangesCompare')}
            onCommentUpdate={noop}
            onDataReady={setDiffs}
          />
        </Container>
      )
    }
  }, [repoMetadata, sourceGitRef, targetGitRef, getString])

  useEffect(() => {
    if (commits?.commits?.length) {
      setTitle(commits.commits[0].title as string)
    }
  }, [commits?.commits])

  return (
    <Container className={css.main}>
      <RepositoryPageHeader
        repoMetadata={repoMetadata}
        title={getString('comparingChanges')}
        dataTooltipId="comparingChanges"
      />
      <PageBody error={getErrorMessage(error || commitsError)} retryOnError={voidFn(refetch)} className={css.pageBody}>
        <LoadingSpinner visible={loading} />

        {repoMetadata && (
          <CompareContentHeader
            loading={creatingInProgress}
            repoMetadata={repoMetadata}
            targetGitRef={targetGitRef}
            onTargetGitRefChanged={gitRef => {
              setTargetGitRef(gitRef)
              history.replace(
                routes.toCODECompare({
                  repoPath: repoMetadata.path as string,
                  diffRefs: makeDiffRefs(gitRef, sourceGitRef)
                })
              )
            }}
            sourceGitRef={sourceGitRef}
            onSourceGitRefChanged={gitRef => {
              setSourceGitRef(gitRef)
              history.replace(
                routes.toCODECompare({
                  repoPath: repoMetadata.path as string,
                  diffRefs: makeDiffRefs(targetGitRef, gitRef)
                })
              )
            }}
            onCreatePullRequestClick={onCreatePullRequest}
          />
        )}

        {(!targetGitRef || !sourceGitRef) && (
          <Container className={css.noDataContainer}>
            <NoDataCard image={Images.EmptyState} message={getString('selectToViewMore')} />
          </Container>
        )}

        {!!repoMetadata && !!targetGitRef && !!sourceGitRef && (
          <Container className={cx(css.tabsContainer, tabContainerCSS.tabsContainer)}>
            <Tabs
              id="prComparing"
              defaultSelectedTabId="general"
              large={false}
              onChange={() => setPage(1)}
              tabList={[
                {
                  id: 'general',
                  title: <TabTitleWithCount icon={CodeIcon.Chat} title={getString('overview')} count={0} />,
                  panel: (
                    <Container className={css.generalTab}>
                      <Layout.Vertical spacing="small" padding="xxlarge">
                        <Container>
                          <Layout.Vertical spacing="small">
                            <Text font={{ variation: FontVariation.SMALL_BOLD }}>{getString('title')} *</Text>
                            <TextInput
                              defaultValue={title}
                              onInput={e => {
                                setTitle((e.currentTarget.value || '').trim())
                              }}
                              placeholder={getString('pr.titlePlaceHolder')}
                            />
                          </Layout.Vertical>
                        </Container>
                        <Container className={css.markdownContainer}>
                          <Layout.Vertical spacing="small">
                            <Text font={{ variation: FontVariation.SMALL_BOLD }}>{getString('description')}</Text>
                            <MarkdownEditorWithPreview
                              value={description}
                              onChange={setDescription}
                              hideButtons
                              autoFocusAndPositioning={true}
                              i18n={{
                                placeHolder: getString('pr.descriptionPlaceHolder'),
                                tabEdit: getString('write'),
                                tabPreview: getString('preview'),
                                save: getString('save'),
                                cancel: getString('cancel')
                              }}
                              editorHeight="100%"
                            />
                          </Layout.Vertical>
                        </Container>
                      </Layout.Vertical>
                      {/** Fake rendering Changes Tab to get changes count - no API has it now */}
                      <Container style={{ display: 'none' }}>{ChangesTab}</Container>
                    </Container>
                  )
                },
                {
                  id: 'commits',
                  title: (
                    <TabTitleWithCount
                      icon={CodeIcon.Commit}
                      title={getString('commits')}
                      count={commits?.commits?.length || 0}
                      padding={{ left: 'medium' }}
                    />
                  ),
                  panel: (
                    <Container padding="xlarge">
                      <CommitsView
                        commits={commits?.commits || []}
                        repoMetadata={repoMetadata}
                        emptyTitle={getString('compareEmptyDiffTitle')}
                        emptyMessage={getString('compareEmptyDiffMessage')}
                      />
                      <ResourceListingPagination response={response} page={page} setPage={setPage} />
                    </Container>
                  )
                },
                {
                  id: 'diff',
                  title: (
                    <TabTitleWithCount
                      icon={CodeIcon.File}
                      title={getString('filesChanged')}
                      count={diffs?.length || 0}
                      padding={{ left: 'medium' }}
                    />
                  ),
                  panel: ChangesTab
                }
              ]}
            />
          </Container>
        )}
      </PageBody>
    </Container>
  )
}
