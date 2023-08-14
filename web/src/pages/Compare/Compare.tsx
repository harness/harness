import { noop } from 'lodash-es'
import React, { useCallback, useState } from 'react'
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
import { getErrorMessage, showToaster } from 'utils/Utils'
import { Images } from 'images'
import { CodeIcon, isRefATag, makeDiffRefs } from 'utils/GitUtils'
import { Changes } from 'components/Changes/Changes'
import type { OpenapiCreatePullReqRequest, TypesDiffStats, TypesPullReq, TypesRepository } from 'services/code'
import { LoadingSpinner } from 'components/LoadingSpinner/LoadingSpinner'
import { TabTitleWithCount, tabContainerCSS } from 'components/TabTitleWithCount/TabTitleWithCount'
import { MarkdownEditorWithPreview } from 'components/MarkdownEditorWithPreview/MarkdownEditorWithPreview'
import { TabContentWrapper } from 'components/TabContentWrapper/TabContentWrapper'
import { CompareContentHeader, PRCreationType } from './CompareContentHeader/CompareContentHeader'
import { CompareCommits } from './CompareCommits'
import css from './Compare.module.scss'

export default function Compare() {
  const { getString } = useStrings()
  const history = useHistory()
  const { routes } = useAppContext()
  const { repoMetadata, error, loading, diffRefs } = useGetRepositoryMetadata()
  const [sourceGitRef, setSourceGitRef] = useState(diffRefs.sourceGitRef)
  const [targetGitRef, setTargetGitRef] = useState(diffRefs.targetGitRef)
  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const { showError } = useToaster()
  const {
    data,
    error: errorStats
  } = useGet<TypesDiffStats>({
    path: `/api/v1/repos/${repoMetadata?.path}/+/diff-stats/${targetGitRef}...${sourceGitRef}`,
    lazy: !repoMetadata
  })
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

  return (
    <Container className={css.main}>
      <RepositoryPageHeader
        repoMetadata={repoMetadata}
        title={getString('comparingChanges')}
        dataTooltipId="comparingChanges"
      />
      <PageBody error={getErrorMessage(error || errorStats)} className={css.pageBody}>
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
                    </Container>
                  )
                },
                {
                  id: 'commits',
                  title: (
                    <TabTitleWithCount
                      icon={CodeIcon.Commit}
                      title={getString('commits')}
                      count={data?.commits}
                      padding={{ left: 'medium' }}
                    />
                  ),
                  panel:
                    <CompareCommits
                        repoMetadata={repoMetadata as TypesRepository}
                        sourceSha={sourceGitRef}
                        targetSha={targetGitRef}
                        handleRefresh={()=>{}} // TODO: when to refresh
                    />
                },
                {
                  id: 'diff',
                  title: (
                    <TabTitleWithCount
                      icon={CodeIcon.File}
                      title={getString('filesChanged')}
                      count={data?.files_changed || 0}
                      padding={{ left: 'medium' }}
                    />
                  ),
                  panel:
                  <TabContentWrapper loading={loading} error={error} onRetry={()=>{}} className={css.changesContainer}>
                    <Changes
                      readOnly
                      repoMetadata={repoMetadata}
                      targetRef={targetGitRef}
                      sourceRef={sourceGitRef}
                      emptyTitle={getString('noChanges')}
                      emptyMessage={getString('noChangesCompare')}
                      onCommentUpdate={noop}
                    />
                  </TabContentWrapper>
                }
              ]}
            />
          </Container>
        )}
      </PageBody>
    </Container>
  )
}
