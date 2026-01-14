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

import React, { useMemo } from 'react'
import { Avatar, Container, Layout, StringSubstitute, Text } from '@harnessio/uicore'
import { Icon, IconName } from '@harnessio/icons'
import { Color, FontVariation } from '@harnessio/design-system'
import { defaultTo } from 'lodash-es'
import { Case, Falsy, Match, Truthy } from 'react-jsx-match'
import { CodeIcon, GitInfoProps, MergeStrategy, getMergeMethodDisplay } from 'utils/GitUtils'
import { useStrings } from 'framework/strings'
import type { TypesPullReqActivity } from 'services/code'
import type { CommentItem } from 'components/CommentBox/CommentBox'
import { PullRequestSection } from 'utils/Utils'
import { LabelActivity, ReviewerAddActivity } from 'components/DiffViewer/DiffViewerUtils'
import { useAppContext } from 'AppContext'
import { CommitActions } from 'components/CommitActions/CommitActions'
import { PipeSeparator } from 'components/PipeSeparator/PipeSeparator'
import { TimePopoverWithLocal } from 'utils/timePopoverLocal/TimePopoverWithLocal'
import { Label } from 'components/Label/Label'
import { GitRefLink } from 'components/GitRefLink/GitRefLink'
import { ActivityLabel, CommentType } from '../PullRequestUtils'
import css from './Conversation.module.scss'

interface SystemCommentProps extends Pick<GitInfoProps, 'pullReqMetadata'> {
  commentItems: CommentItem<TypesPullReqActivity>[]
  repoMetadataPath?: string
}

interface MergePayload {
  merge_sha: string
  merge_method: string
  rules_bypassed: boolean
}

interface ReviewerAddActivityPayload {
  reviewer_type: ReviewerAddActivity
}

const formatListWithAndFragment = (names: string[]): React.ReactNode => {
  switch (names.length) {
    case 0:
      return null
    case 1:
      return <strong>{names[0]}</strong>
    case 2:
      return (
        <>
          <strong>{names[0]}</strong> and <strong>{names[1]}</strong>
        </>
      )
    default:
      return (
        <>
          {names.slice(0, -1).map((name, index) => (
            <React.Fragment key={index}>
              <strong>{name}</strong>
              {index < names.length - 2 ? ', ' : ''}
            </React.Fragment>
          ))}{' '}
          and <strong>{names[names.length - 1]}</strong>
        </>
      )
  }
}

const RenderScopedLabel = ({ labelObj }: { labelObj: ActivityLabel }) => {
  return (
    <Label
      name={labelObj.label}
      label_color={labelObj.label_color}
      label_value={{
        name: labelObj.value,
        color: labelObj.value_color
      }}
      scope={labelObj.label_scope}
    />
  )
}

const formatLabelListWithAndFragment = (labels: ActivityLabel[]): JSX.Element => {
  if (!labels || !Array.isArray(labels)) {
    return <></>
  }

  switch (labels.length) {
    case 0:
      return <></>
    case 1:
      return <RenderScopedLabel labelObj={labels[0]} />
    case 2:
      return (
        <>
          <RenderScopedLabel labelObj={labels[0]} /> and <RenderScopedLabel labelObj={labels[1]} />
        </>
      )
    default:
      return (
        <>
          {labels.slice(0, -1).map((labelObj, index) => (
            <React.Fragment key={index}>
              <RenderScopedLabel labelObj={labelObj} />
              {index < labels.length - 2 ? ', ' : ''}
            </React.Fragment>
          ))}{' '}
          and <RenderScopedLabel labelObj={labels[labels.length - 1]} />
        </>
      )
  }
}

//ToDo : update all comment options with the correct payload type and remove Unknown
export const SystemComment: React.FC<SystemCommentProps> = ({ pullReqMetadata, commentItems, repoMetadataPath }) => {
  const { getString } = useStrings()
  const payload = commentItems[0].payload
  const type = payload?.type
  const { routes } = useAppContext()

  const displayNameList = useMemo(() => {
    const checkList = payload?.metadata?.mentions?.ids ?? []
    const uniqueList = [...new Set(checkList)]
    const mentionsMap = payload?.mentions ?? {}
    return uniqueList.map(id => mentionsMap[id]?.display_name ?? '')
  }, [payload?.metadata?.mentions?.ids, payload?.mentions])

  const userGroupsDisplayNameList = useMemo(() => {
    const ids = payload?.metadata?.mentions?.user_group_ids ?? []
    const uniqueList = [...new Set(ids)]
    const mentionsMap = payload?.user_group_mentions ?? {}
    return uniqueList.map(id => mentionsMap[id]?.name ?? '')
  }, [payload?.metadata?.mentions?.user_group_ids, payload?.user_group_mentions])

  const principalNameList = useMemo(() => {
    const checkList = (payload?.payload as any)?.principal_ids ?? []
    const uniqueList = [...new Set(checkList)]
    const mentionsMap = payload?.mentions ?? {}
    return uniqueList.map(id => mentionsMap[id as number]?.display_name ?? '')
  }, [(payload?.payload as any)?.principal_ids, payload?.mentions])

  switch (type) {
    case CommentType.MERGE: {
      const areRulesBypassed = (payload?.payload as MergePayload)?.rules_bypassed
      return (
        <Container>
          <Layout.Horizontal spacing="small" style={{ alignItems: 'center' }} className={css.mergedBox}>
            <Container margin={{ left: 'xsmall' }} width={24} height={24} className={css.mergeContainer}>
              <Icon name={CodeIcon.Merged} size={16} color={Color.PURPLE_700} />
            </Container>

            <Avatar name={pullReqMetadata.merger?.display_name} size="small" hoverCard={false} />
            <Text flex tag="div" style={{ whiteSpace: 'nowrap' }}>
              <StringSubstitute
                str={areRulesBypassed ? getString('pr.prMergedInfoWithBypassingRules') : getString('pr.prMergedInfo')}
                vars={{
                  user: <strong className={css.rightTextPadding}>{pullReqMetadata.merger?.display_name}</strong>,
                  mergeMethod: (
                    <Text
                      lineClamp={1}
                      margin={{
                        left: areRulesBypassed ? 'xsmall' : '',
                        right: 'xsmall'
                      }}>
                      {getMergeMethodDisplay(pullReqMetadata?.merge_method as MergeStrategy)}
                    </Text>
                  ),
                  source: (
                    <Text lineClamp={1}>
                      <strong className={css.textPadding}>{pullReqMetadata.source_branch}</strong>
                    </Text>
                  ),
                  target: (
                    <Text lineClamp={1}>
                      <strong className={css.textPadding}>{pullReqMetadata.target_branch}</strong>
                    </Text>
                  ),
                  mergeSha: (
                    <Container className={css.commitContainer} padding={{ left: 'small', right: 'xsmall' }}>
                      <CommitActions
                        enableCopy
                        sha={(payload?.payload as MergePayload)?.merge_sha}
                        href={routes.toCODEPullRequest({
                          repoPath: repoMetadataPath as string,
                          pullRequestSection: PullRequestSection.FILES_CHANGED,
                          pullRequestId: String(pullReqMetadata.number),
                          commitSHA: (payload?.payload as MergePayload)?.merge_sha as string
                        })}
                      />
                    </Container>
                  ),
                  time: (
                    <Text inline margin={{ left: 'xsmall' }}>
                      <PipeSeparator height={9} />
                      <TimePopoverWithLocal
                        time={defaultTo(pullReqMetadata.merged as number, 0)}
                        inline={false}
                        className={css.timeText}
                        font={{ variation: FontVariation.SMALL }}
                        color={Color.GREY_400}
                      />
                    </Text>
                  )
                }}
              />
            </Text>
          </Layout.Horizontal>
        </Container>
      )
    }

    case CommentType.REVIEW_SUBMIT: {
      return (
        <Container>
          <Layout.Horizontal spacing="small" style={{ alignItems: 'center' }} className={css.mergedBox}>
            <Icon
              margin={{ left: 'small' }}
              padding={{ right: 'small' }}
              {...generateReviewDecisionIcon((payload?.payload as Unknown)?.decision)}
            />

            <Avatar name={payload?.author?.display_name as string} size="small" hoverCard={false} />
            <Text color={Color.GREY_500}>
              <StringSubstitute
                str={getString('pr.prReviewSubmit')}
                vars={{
                  user: <strong>{payload?.author?.display_name}</strong>,
                  state: (payload?.payload as Unknown)?.decision,
                  time: (
                    <Text inline margin={{ left: 'xsmall' }}>
                      <PipeSeparator height={9} />
                      <TimePopoverWithLocal
                        time={defaultTo(payload?.created as number, 0)}
                        inline={false}
                        className={css.timeText}
                        font={{ variation: FontVariation.SMALL }}
                        color={Color.GREY_400}
                      />
                    </Text>
                  )
                }}
              />
            </Text>
          </Layout.Horizontal>
        </Container>
      )
    }

    case CommentType.BRANCH_UPDATE: {
      return (
        <Container>
          <Layout.Horizontal spacing="small" style={{ alignItems: 'center' }} className={css.mergedBox}>
            <Avatar name={payload?.author?.display_name} size="small" hoverCard={false} />
            <Text flex tag="div">
              {(payload?.payload as Unknown)?.forced ? (
                <StringSubstitute
                  str={getString('pr.prBranchForcePushInfo')}
                  vars={{
                    user: (
                      <Text padding={{ right: 'small' }} inline>
                        <strong>{payload?.author?.display_name}</strong>
                      </Text>
                    ),
                    oldCommit: (
                      <Container className={css.commitContainer} padding={{ left: 'small', right: 'small' }}>
                        <CommitActions
                          enableCopy
                          sha={(payload?.payload as Unknown)?.old}
                          href={routes.toCODEPullRequest({
                            repoPath: repoMetadataPath as string,
                            pullRequestSection: PullRequestSection.FILES_CHANGED,
                            pullRequestId: String(pullReqMetadata.number),
                            commitSHA: (payload?.payload as Unknown)?.old as string
                          })}
                        />
                      </Container>
                    ),
                    newCommit: (
                      <Container className={css.commitContainer} padding={{ left: 'small' }}>
                        <CommitActions
                          enableCopy
                          sha={(payload?.payload as Unknown)?.new}
                          href={routes.toCODEPullRequest({
                            repoPath: repoMetadataPath as string,
                            pullRequestSection: PullRequestSection.FILES_CHANGED,
                            pullRequestId: String(pullReqMetadata.number),
                            commitSHA: (payload?.payload as Unknown)?.new as string
                          })}
                        />
                      </Container>
                    )
                  }}
                />
              ) : (
                <StringSubstitute
                  str={getString('pr.prBranchPushInfo')}
                  vars={{
                    user: (
                      <Text padding={{ right: 'small' }} inline>
                        <strong>{payload?.author?.display_name}</strong>
                      </Text>
                    ),
                    commit: (
                      <Container className={css.commitContainer} padding={{ left: 'small' }}>
                        <CommitActions
                          enableCopy
                          sha={(payload?.payload as Unknown)?.new}
                          href={routes.toCODEPullRequest({
                            repoPath: repoMetadataPath as string,
                            pullRequestSection: PullRequestSection.FILES_CHANGED,
                            pullRequestId: String(pullReqMetadata.number),
                            commitSHA: (payload?.payload as Unknown)?.new as string
                          })}
                        />
                      </Container>
                    )
                  }}
                />
              )}
            </Text>
            <PipeSeparator height={9} />
            <TimePopoverWithLocal
              time={defaultTo(payload?.created as number, 0)}
              inline={true}
              width={100}
              font={{ variation: FontVariation.SMALL }}
              color={Color.GREY_400}
            />
          </Layout.Horizontal>
        </Container>
      )
    }

    case CommentType.BRANCH_DELETE:
    case CommentType.BRANCH_RESTORE: {
      const isSourceBranchDeleted = type === CommentType.BRANCH_DELETE
      return (
        <Container>
          <Layout.Horizontal spacing="small" style={{ alignItems: 'center' }} className={css.mergedBox}>
            <Avatar name={payload?.author?.display_name} size="small" hoverCard={false} />
            <Text flex tag="div">
              <StringSubstitute
                str={isSourceBranchDeleted ? getString('pr.prBranchDeleteInfo') : getString('pr.prBranchRestoreInfo')}
                vars={{
                  user: (
                    <Text padding={{ right: 'small' }} inline>
                      <strong>{payload?.author?.display_name}</strong>
                    </Text>
                  ),
                  commit: (
                    <Container className={css.commitContainer} padding={{ left: 'small' }}>
                      <CommitActions
                        enableCopy
                        sha={(payload?.payload as Unknown)?.sha}
                        href={routes.toCODEPullRequest({
                          repoPath: repoMetadataPath as string,
                          pullRequestSection: PullRequestSection.FILES_CHANGED,
                          pullRequestId: String(pullReqMetadata.number),
                          commitSHA: (payload?.payload as Unknown)?.sha as string
                        })}
                      />
                    </Container>
                  )
                }}
              />
            </Text>
            <PipeSeparator height={9} />
            <TimePopoverWithLocal
              time={defaultTo(payload?.created as number, 0)}
              width={100}
              inline={true}
              font={{ variation: FontVariation.SMALL }}
              color={Color.GREY_400}
            />
          </Layout.Horizontal>
        </Container>
      )
    }

    case CommentType.STATE_CHANGE: {
      const openFromDraft =
        (payload?.payload as Unknown)?.old_draft === true && (payload?.payload as Unknown)?.new_draft === false
      const changedToDraft =
        (payload?.payload as Unknown)?.old_draft === false && (payload?.payload as Unknown)?.new_draft === true
      return (
        <Container>
          <Layout.Horizontal spacing="small" style={{ alignItems: 'center' }} className={css.mergedBox}>
            <Avatar name={payload?.author?.display_name} size="small" hoverCard={false} />
            <Text>
              <StringSubstitute
                str={getString(openFromDraft || changedToDraft ? 'pr.prStateChangedDraft' : 'pr.prStateChanged')}
                vars={{
                  changedToDraft,
                  user: <strong>{payload?.author?.display_name}</strong>,
                  old: <strong>{(payload?.payload as Unknown)?.old}</strong>,
                  new: <strong>{(payload?.payload as Unknown)?.new}</strong>
                }}
              />
            </Text>
            <PipeSeparator height={9} />
            <TimePopoverWithLocal
              time={defaultTo(payload?.created as number, 0)}
              inline={true}
              width={100}
              font={{ variation: FontVariation.SMALL }}
              color={Color.GREY_400}
            />
          </Layout.Horizontal>
        </Container>
      )
    }

    case CommentType.TITLE_CHANGE: {
      return (
        <Container className={css.mergedBox}>
          <Layout.Horizontal spacing="small" style={{ alignItems: 'center' }}>
            <Avatar name={payload?.author?.display_name} size="small" hoverCard={false} />
            <Text tag="div">
              <StringSubstitute
                str={getString('pr.titleChanged')}
                vars={{
                  user: <strong>{payload?.author?.display_name}</strong>,
                  old: (
                    <strong>
                      <s>{(payload?.payload as Unknown)?.old}</s>
                    </strong>
                  ),
                  new: <strong>{(payload?.payload as Unknown)?.new}</strong>
                }}
              />
            </Text>
            <PipeSeparator height={9} />

            <TimePopoverWithLocal
              time={defaultTo(payload?.created as number, 0)}
              inline={true}
              width={100}
              font={{ variation: FontVariation.SMALL }}
              color={Color.GREY_400}
            />
          </Layout.Horizontal>
        </Container>
      )
    }

    case CommentType.LABEL_MODIFY: {
      const labelList = (payload?.payload as any)?.labels || []
      const labelsListElement = formatLabelListWithAndFragment(labelList)

      return (
        <Container className={css.mergedBox}>
          <Layout.Horizontal spacing="small" style={{ alignItems: 'center' }}>
            <Avatar name={payload?.author?.display_name} size="small" hoverCard={false} />
            <Text tag="div">
              <Match expr={(payload?.payload as Unknown)?.type}>
                <Case val={LabelActivity.ASSIGN}>
                  <strong>{payload?.author?.display_name}</strong> {getString('labels.applied')}
                  {Array.isArray(labelList) && labelList.length > 0 ? (
                    <>{labelsListElement}</>
                  ) : (
                    <RenderScopedLabel labelObj={payload?.payload as ActivityLabel} />
                  )}{' '}
                  <StringSubstitute
                    str={getString('prReview.labelsAssigned')}
                    vars={{
                      count: labelList?.length || 0
                    }}
                  />
                </Case>
                <Case val={LabelActivity.RE_ASSIGN}>
                  <strong>{payload?.author?.display_name}</strong> <span>{getString('labels.updated')}</span>
                  <Label
                    name={(payload?.payload as Unknown).label}
                    label_color={(payload?.payload as Unknown).label_color}
                    label_value={{
                      name: (payload?.payload as Unknown).old_value,
                      color: (payload?.payload as Unknown).old_value_color
                    }}
                    scope={(payload?.payload as Unknown).label_scope}
                  />
                  <span>{getString('labels.labelTo')}</span>
                  <RenderScopedLabel labelObj={payload?.payload as ActivityLabel} />
                </Case>
                <Case val={LabelActivity.UN_ASSIGN}>
                  <strong>{payload?.author?.display_name}</strong> <span>{getString('labels.removed')}</span>
                  <RenderScopedLabel labelObj={payload?.payload as ActivityLabel} />
                  <span>{getString('labels.label')}</span>
                </Case>
              </Match>
            </Text>
            <PipeSeparator height={9} />

            <TimePopoverWithLocal
              time={defaultTo(payload?.created as number, 0)}
              inline={true}
              width={100}
              font={{ variation: FontVariation.SMALL }}
              color={Color.GREY_400}
            />
          </Layout.Horizontal>
        </Container>
      )
    }

    case CommentType.REVIEWER_ADD: {
      const activityMentions = formatListWithAndFragment(displayNameList)
      const principalMentions = formatListWithAndFragment(principalNameList)

      return (
        <Container className={css.mergedBox}>
          <Layout.Horizontal spacing="small" style={{ alignItems: 'center' }}>
            <Avatar name={payload?.author?.display_name} size="small" hoverCard={false} />
            <Text tag="div">
              <Match expr={(payload?.payload as ReviewerAddActivityPayload).reviewer_type}>
                <Case val={ReviewerAddActivity.ASSIGNED}>
                  <StringSubstitute
                    str={getString('prReview.assigned')}
                    vars={{
                      author: <strong>{payload?.author?.display_name}</strong>,
                      reviewer: activityMentions
                    }}
                  />
                </Case>
                <Case val={ReviewerAddActivity.REQUESTED}>
                  <StringSubstitute
                    str={getString('prReview.requested')}
                    vars={{
                      author: <strong>{payload?.author?.display_name}</strong>,
                      reviewer: activityMentions
                    }}
                  />
                </Case>
                <Case val={ReviewerAddActivity.SELF_ASSIGNED}>
                  <StringSubstitute
                    str={getString('prReview.selfAssigned')}
                    vars={{
                      reviewer: activityMentions
                    }}
                  />
                </Case>
                <Case val={ReviewerAddActivity.CODEOWNERS}>
                  <StringSubstitute
                    str={getString('prReview.codeowners')}
                    vars={{
                      author: <strong>{payload?.author?.display_name}</strong>,
                      codeowners: principalMentions,
                      count: principalNameList.length
                    }}
                  />
                </Case>
                <Case val={ReviewerAddActivity.DEFAULT}>
                  <StringSubstitute
                    str={getString('prReview.defaultReviewers')}
                    vars={{
                      author: <strong>{payload?.author?.display_name}</strong>,
                      reviewers: principalMentions,
                      count: principalNameList.length
                    }}
                  />
                </Case>
              </Match>
            </Text>
            <PipeSeparator height={9} />

            <TimePopoverWithLocal
              time={defaultTo(payload?.created as number, 0)}
              inline={true}
              width={100}
              font={{ variation: FontVariation.SMALL }}
              color={Color.GREY_400}
            />
          </Layout.Horizontal>
        </Container>
      )
    }

    case CommentType.USER_GROUP_REVIEWER_ADD: {
      const userGroupReviewers = formatListWithAndFragment(userGroupsDisplayNameList)

      return (
        <Container className={css.mergedBox}>
          <Layout.Horizontal spacing="small" style={{ alignItems: 'center' }}>
            <Avatar name={payload?.author?.display_name} size="small" hoverCard={false} />
            <Text tag="div">
              <Match expr={(payload?.payload as ReviewerAddActivityPayload).reviewer_type}>
                <Case val={ReviewerAddActivity.REQUESTED}>
                  <StringSubstitute
                    str={getString('prReview.requested')}
                    vars={{
                      author: <strong>{payload?.author?.display_name}</strong>,
                      reviewer: userGroupReviewers
                    }}
                  />
                </Case>
                <Case val={ReviewerAddActivity.CODEOWNERS}>
                  <StringSubstitute
                    str={getString('prReview.ugCodeowners')}
                    vars={{
                      author: <strong>{payload?.author?.display_name}</strong>,
                      codeowners: userGroupReviewers,
                      count: userGroupsDisplayNameList.length
                    }}
                  />
                </Case>
                <Case val={ReviewerAddActivity.DEFAULT}>
                  <StringSubstitute
                    str={getString('prReview.ugDefaultReviewers')}
                    vars={{
                      author: <strong>{payload?.author?.display_name}</strong>,
                      reviewers: userGroupReviewers,
                      count: userGroupsDisplayNameList.length
                    }}
                  />
                </Case>
              </Match>
            </Text>
            <PipeSeparator height={9} />

            <TimePopoverWithLocal
              time={defaultTo(payload?.created as number, 0)}
              inline={true}
              width={100}
              font={{ variation: FontVariation.SMALL }}
              color={Color.GREY_400}
            />
          </Layout.Horizontal>
        </Container>
      )
    }

    case CommentType.REVIEWER_DELETE: {
      const mentionId = payload?.metadata?.mentions?.ids?.[0] ?? 0
      const mentionDisplayName = payload?.mentions?.[mentionId]?.display_name ?? ''
      return (
        <Container className={css.mergedBox}>
          <Layout.Horizontal spacing="small" style={{ alignItems: 'center' }}>
            <Avatar name={payload?.author?.display_name} size="small" hoverCard={false} />
            <Text tag="div">
              <Match expr={payload?.author?.id === mentionId}>
                <Truthy>
                  <StringSubstitute
                    str={getString('prReview.selfRemoved')}
                    vars={{
                      author: <strong>{payload?.author?.display_name}</strong>
                    }}
                  />
                </Truthy>
                <Falsy>
                  <StringSubstitute
                    str={getString('prReview.removed')}
                    vars={{
                      author: <strong>{payload?.author?.display_name}</strong>,
                      reviewer: <strong>{mentionDisplayName}</strong>
                    }}
                  />
                </Falsy>
              </Match>
            </Text>
            <PipeSeparator height={9} />

            <TimePopoverWithLocal
              time={defaultTo(payload?.created as number, 0)}
              inline={true}
              width={100}
              font={{ variation: FontVariation.SMALL }}
              color={Color.GREY_400}
            />
          </Layout.Horizontal>
        </Container>
      )
    }

    case CommentType.USER_GROUP_REVIEWER_DELETE: {
      const userGroupReviewers = formatListWithAndFragment(userGroupsDisplayNameList)

      return (
        <Container className={css.mergedBox}>
          <Layout.Horizontal spacing="small" style={{ alignItems: 'center' }}>
            <Avatar name={payload?.author?.display_name} size="small" hoverCard={false} />
            <Text tag="div">
              <StringSubstitute
                str={getString('prReview.ugRemoved')}
                vars={{
                  author: <strong>{payload?.author?.display_name}</strong>,
                  reviewer: <strong>{userGroupReviewers}</strong>
                }}
              />
            </Text>
            <PipeSeparator height={9} />

            <TimePopoverWithLocal
              time={defaultTo(payload?.created as number, 0)}
              inline={true}
              width={100}
              font={{ variation: FontVariation.SMALL }}
              color={Color.GREY_400}
            />
          </Layout.Horizontal>
        </Container>
      )
    }

    case CommentType.TARGET_BRANCH_CHANGE: {
      const vars = {
        user: <strong>{payload?.author?.display_name}</strong>,
        old: (
          <GitRefLink
            text={(payload?.payload as Unknown).old as string}
            url={routes.toCODERepository({
              repoPath: repoMetadataPath as string,
              gitRef: (payload?.payload as Unknown).old
            })}
            showCopy
          />
        ),
        new: (
          <GitRefLink
            text={(payload?.payload as Unknown).new as string}
            url={routes.toCODERepository({
              repoPath: repoMetadataPath as string,
              gitRef: (payload?.payload as Unknown).new
            })}
            showCopy
          />
        )
      }
      return (
        <Container className={css.mergedBox}>
          <Layout.Horizontal spacing="small" style={{ alignItems: 'center' }}>
            <Avatar name={payload?.author?.display_name} size="small" hoverCard={false} />
            <Text tag="div">
              <StringSubstitute str={getString('prReview.targetBranchChange')} vars={vars} />
            </Text>
            <PipeSeparator height={9} />
            <TimePopoverWithLocal
              time={defaultTo(payload?.created as number, 0)}
              inline
              width={100}
              font={{ variation: FontVariation.SMALL }}
              color={Color.GREY_400}
            />
          </Layout.Horizontal>
        </Container>
      )
    }

    default: {
      // eslint-disable-next-line no-console
      console.warn('Unable to render system type activity', commentItems)
      return (
        <Text className={css.mergedBox}>
          <Icon name={CodeIcon.Commit} padding={{ right: 'small' }} />
          {type}
        </Text>
      )
    }
  }
}

const generateReviewDecisionIcon = (
  reviewDecision: string
): {
  name: IconName
  color: string | undefined
  size: number | undefined
  icon: IconName
  iconProps?: { color?: Color }
} => {
  let icon: IconName = 'dot'
  let color: Color | undefined = undefined
  let size: number | undefined = undefined

  switch (reviewDecision) {
    case 'changereq':
      icon = 'main-issue-filled'
      color = Color.ORANGE_700
      size = 18
      break
    case 'approved':
      icon = 'execution-success'
      size = 18
      color = Color.GREEN_700
      break
  }
  const name = icon
  return { name, color, size, icon, ...(color ? { iconProps: { color } } : undefined) }
}
