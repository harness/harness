/*
 * Copyright 2023 Harness, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { noop } from 'lodash-es'
import React, { useCallback, useState, useEffect, useRef } from 'react'
import {
  Container,
  PageBody,
  NoDataCard,
  Tabs,
  Layout,
  TextInput,
  Text,
  useToaster,
  StringSubstitute
} from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'
import cx from 'classnames'
import { useHistory } from 'react-router-dom'
import { useGet, useMutate } from 'restful-react'
import { useAppContext } from 'AppContext'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { useStrings } from 'framework/strings'
import { RepositoryPageHeader } from 'components/RepositoryPageHeader/RepositoryPageHeader'
import { LIST_FETCHING_LIMIT, getErrorMessage, showToaster } from 'utils/Utils'
import { Images } from 'images'
import { CodeIcon, normalizeGitRef, isRefATag, makeDiffRefs, isGitRev } from 'utils/GitUtils'
import { Changes } from 'components/Changes/Changes'
import type {
  OpenapiCreatePullReqRequest,
  OpenapiGetContentOutput,
  RepoFileContent,
  TypesCommit,
  TypesDiffStats,
  TypesPullReq,
  TypesRepository
} from 'services/code'
import { LoadingSpinner } from 'components/LoadingSpinner/LoadingSpinner'
import { TabTitleWithCount, tabContainerCSS } from 'components/TabTitleWithCount/TabTitleWithCount'
import { MarkdownEditorWithPreview } from 'components/MarkdownEditorWithPreview/MarkdownEditorWithPreview'
import { MAX_TEXT_LINE_SIZE_LIMIT, PULL_REQUEST_DESCRIPTION_SIZE_LIMIT } from 'settings'
import { usePageIndex } from 'hooks/usePageIndex'
import { TabContentWrapper } from 'components/TabContentWrapper/TabContentWrapper'
import { useSetPageContainerWidthVar } from 'hooks/useSetPageContainerWidthVar'
import { CompareContentHeader, PRCreationType } from './CompareContentHeader/CompareContentHeader'
import { CompareCommits } from './CompareCommits'
import css from './Compare.module.scss'

export default function Compare() {
  const { getString } = useStrings()
  const history = useHistory()
  const { routes, standalone, routingId } = useAppContext()
  const { repoMetadata, error, loading, diffRefs } = useGetRepositoryMetadata()
  const [sourceGitRef, setSourceGitRef] = useState(diffRefs.sourceGitRef)
  const [targetGitRef, setTargetGitRef] = useState(diffRefs.targetGitRef)
  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const { showError } = useToaster()
  const [page, setPage] = usePageIndex()
  const { data, error: errorStats } = useGet<TypesDiffStats>({
    path: `/api/v1/repos/${repoMetadata?.path}/+/diff-stats/${normalizeGitRef(targetGitRef)}...${normalizeGitRef(
      sourceGitRef
    )}`,
    lazy: !repoMetadata || sourceGitRef === ''
  })
  const { mutate: createPullRequest, loading: creatingInProgress } = useMutate<TypesPullReq>({
    verb: 'POST',
    path: `/api/v1/repos/${repoMetadata?.path}/+/pullreq`
  })
  const { data: commits } = useGet<{
    commits: TypesCommit[]
  }>({
    path: `/api/v1/repos/${repoMetadata?.path}/+/commits`,
    queryParams: {
      limit: LIST_FETCHING_LIMIT,
      page,
      git_ref: normalizeGitRef(sourceGitRef),
      after: normalizeGitRef(targetGitRef)
    },
    lazy: !repoMetadata || sourceGitRef === ''
  })
  const { data: prTemplateData } = useGet<OpenapiGetContentOutput>({
    path: `/api/v1/repos/${repoMetadata?.path}/+/content/.harness/pull_request_template.md`,
    queryParams: {
      include_commit: false,
      git_ref: normalizeGitRef(targetGitRef)
    },
    lazy: !repoMetadata || sourceGitRef === ''
  })

  const onCreatePullRequest = useCallback(
    (creationType: PRCreationType) => {
      if (!sourceGitRef || !targetGitRef) {
        return showToaster(getString('prMustSelectSourceAndTargetBranches'))
      }

      if (isRefATag(sourceGitRef) || isRefATag(targetGitRef) || isGitRev(targetGitRef) || isGitRev(sourceGitRef)) {
        return showToaster(getString('pullMustBeMadeFromBranches'))
      }

      if (sourceGitRef === targetGitRef) {
        return showToaster(getString('prSourceAndTargetMustBeDifferent'))
      }

      if (!title) {
        return showError(getString('pr.titleIsRequired'))
      }

      if (description?.split('\n').some(line => line.length > MAX_TEXT_LINE_SIZE_LIMIT)) {
        return showError(getString('pr.descHasTooLongLine', { max: MAX_TEXT_LINE_SIZE_LIMIT }), 0)
      }

      if (description?.length > PULL_REQUEST_DESCRIPTION_SIZE_LIMIT) {
        return showError(
          getString('pr.descIsTooLong', { max: PULL_REQUEST_DESCRIPTION_SIZE_LIMIT, len: description?.length }),
          0
        )
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
          .then(_data => {
            history.replace(
              routes.toCODEPullRequest({
                repoPath: repoMetadata?.path as string,
                pullRequestId: String(_data.number)
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

  useEffect(() => {
    if (commits?.commits?.length) {
      setTitle(commits.commits[0].title as string)
    }
  }, [commits?.commits])

  const domRef = useRef<HTMLDivElement>(null)
  useSetPageContainerWidthVar({ domRef })

  return (
    <Container className={css.main} ref={domRef}>
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
                              routingId={routingId}
                              standalone={standalone}
                              repoMetadata={repoMetadata}
                              value={description}
                              templateData={(prTemplateData?.content as RepoFileContent)?.data}
                              onChange={setDescription}
                              hideButtons
                              autoFocusAndPosition={true}
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
                      count={data?.commits || 0}
                      padding={{ left: 'medium' }}
                    />
                  ),
                  panel: (
                    <CompareCommits
                      repoMetadata={repoMetadata as TypesRepository}
                      sourceSha={sourceGitRef}
                      targetSha={targetGitRef}
                    />
                  )
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
                  panel: (
                    <TabContentWrapper loading={loading} error={error} onRetry={noop} className={css.changesContainer}>
                      <Changes
                        readOnly={true}
                        repoMetadata={repoMetadata}
                        targetRef={targetGitRef}
                        sourceRef={sourceGitRef}
                        emptyTitle={getString('noChanges')}
                        emptyMessage={getString('noChangesCompare')}
                        scrollElement={
                          (standalone
                            ? document.querySelector(`.${css.main}`)?.parentElement || window
                            : window) as HTMLElement
                        }
                      />
                    </TabContentWrapper>
                  )
                }
              ]}
            />
          </Container>
        )}
      </PageBody>
    </Container>
  )
}
