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

import React, { useEffect, useMemo, useState } from 'react'
import {
  Button,
  ButtonVariation,
  Checkbox,
  Container,
  FlexExpander,
  Layout,
  SplitButton,
  StringSubstitute,
  Text,
  useToaster
} from '@harnessio/uicore'
import { Icon } from '@harnessio/icons'
import { Color } from '@harnessio/design-system'
import { useMutate } from 'restful-react'
import { Case, Else, Match, Render, Truthy } from 'react-jsx-match'
import { Menu, PopoverPosition, Icon as BIcon } from '@blueprintjs/core'
import cx from 'classnames'
import ReactTimeago from 'react-timeago'
import type {
  EnumPullReqState,
  OpenapiMergePullReq,
  OpenapiStatePullReqRequest,
  TypesPullReq,
  TypesRuleViolations
} from 'services/code'
import { useStrings } from 'framework/strings'
import { CodeIcon, PullRequestFilterOption, PullRequestState } from 'utils/GitUtils'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { useAppContext } from 'AppContext'
import { Images } from 'images'
import {
  extractInfoFromRuleViolationArr,
  getErrorMessage,
  MergeCheckStatus,
  permissionProps,
  PRDraftOption,
  PRMergeOption,
  PullRequestActionsBoxProps,
  Violation
} from 'utils/Utils'
import { UserPreference, useUserPreference } from 'hooks/useUserPreference'
import ReviewSplitButton from 'components/Changes/ReviewSplitButton/ReviewSplitButton'
import RuleViolationAlertModal from 'components/RuleViolationAlertModal/RuleViolationAlertModal'
import css from './PullRequestActionsBox.module.scss'

const POLLING_INTERVAL = 60000
export const PullRequestActionsBox: React.FC<PullRequestActionsBoxProps> = ({
  repoMetadata,
  pullRequestMetadata,
  onPRStateChanged,
  refetchReviewers
}) => {
  const [isActionBoxOpen, setActionBoxOpen] = useState(false)
  const { getString } = useStrings()
  const { showError } = useToaster()
  const { currentUser } = useAppContext()
  const { hooks, standalone } = useAppContext()
  const space = useGetSpaceParam()
  const { mutate: mergePR, loading } = useMutate({
    verb: 'POST',
    path: `/api/v1/repos/${repoMetadata.path}/+/pullreq/${pullRequestMetadata.number}/merge`
  })
  const [ruleViolation, setRuleViolation] = useState(false)
  const [ruleViolationArr, setRuleViolationArr] = useState<{ data: { rule_violations: TypesRuleViolations[] } }>()
  const [length, setLength] = useState(0)
  const [notBypassable, setNotBypassable] = useState(false)
  const [finalRulesArr, setFinalRulesArr] = useState<Violation[]>()
  const [bypass, setBypass] = useState(false)
  const { mutate: updatePRState, loading: loadingState } = useMutate({
    verb: 'POST',
    path: `/api/v1/repos/${repoMetadata.path}/+/pullreq/${pullRequestMetadata.number}/state`
  })
  const mergeable = useMemo(
    () => pullRequestMetadata.merge_check_status === MergeCheckStatus.MERGEABLE,
    [pullRequestMetadata]
  )
  const isClosed = pullRequestMetadata.state === PullRequestState.CLOSED
  const isOpen = pullRequestMetadata.state === PullRequestState.OPEN
  const isConflict = pullRequestMetadata.merge_check_status === MergeCheckStatus.CONFLICT
  const unchecked = useMemo(
    () => pullRequestMetadata.merge_check_status === MergeCheckStatus.UNCHECKED && !isClosed,
    [pullRequestMetadata, isClosed]
  )

  useEffect(() => {
    if (ruleViolationArr && !isDraft && ruleViolationArr.data.rule_violations) {
      const { checkIfBypassAllowed, violationArr, uniqueViolations } = extractInfoFromRuleViolationArr(
        ruleViolationArr.data.rule_violations
      )
      setNotBypassable(checkIfBypassAllowed)
      setFinalRulesArr(violationArr)
      setLength(uniqueViolations.size)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [ruleViolationArr])
  const dryMerge = () => {
    if (!isClosed && pullRequestMetadata.state !== PullRequestState.MERGED) {
      mergePR({ bypass_rules: true, dry_run: true, source_sha: pullRequestMetadata?.source_sha })
        .then(res => {
          if (res?.rule_violations?.length > 0) {
            setRuleViolation(true)
            setRuleViolationArr({ data: { rule_violations: res?.rule_violations } })
            setAllowedStrats(res.allowed_methods)
          } else {
            setRuleViolation(false)
            setAllowedStrats(res.allowed_methods)
          }
        })
        .catch(err => {
          if (err.status === 422) {
            setRuleViolation(true)
            setRuleViolationArr(err)
            setAllowedStrats(err.allowed_methods)
          } else {
            showError(getErrorMessage(err))
          }
        })
    }
  }
  useEffect(() => {
    dryMerge() // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])
  useEffect(() => {
    // dryMerge()
    const intervalId = setInterval(async () => {
      dryMerge()
    }, POLLING_INTERVAL) // Poll every 20 seconds
    // Cleanup interval on component unmount
    return () => {
      clearInterval(intervalId)
    } // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [onPRStateChanged])
  const isDraft = pullRequestMetadata.is_draft
  const mergeOptions: PRMergeOption[] = [
    {
      method: 'squash',
      title: getString('pr.mergeOptions.squashAndMerge'),
      desc: getString('pr.mergeOptions.squashAndMergeDesc'),
      disabled: mergeable === false
    },
    {
      method: 'merge',
      title: getString('pr.mergeOptions.createMergeCommit'),
      desc: getString('pr.mergeOptions.createMergeCommitDesc'),
      disabled: mergeable === false
    },
    {
      method: 'rebase',
      title: getString('pr.mergeOptions.rebaseAndMerge'),
      desc: getString('pr.mergeOptions.rebaseAndMergeDesc'),
      disabled: mergeable === false
    },
    {
      method: 'close',
      title: getString('pr.mergeOptions.close'),
      desc: getString('pr.mergeOptions.closeDesc')
    }
  ]
  const [allowedStrats, setAllowedStrats] = useState<string[]>([
    mergeOptions[0].method,
    mergeOptions[1].method,
    mergeOptions[2].method,
    mergeOptions[3].method
  ])
  const draftOptions: PRDraftOption[] = [
    {
      method: 'open',
      title: getString('pr.draftOpenForReview.title'),
      desc: getString('pr.draftOpenForReview.desc')
    },
    {
      method: 'close',
      title: getString('pr.mergeOptions.close'),
      desc: getString('pr.mergeOptions.closeDesc')
    }
  ]

  const [mergeOption, setMergeOption, resetMergeOption] = useUserPreference<PRMergeOption>(
    UserPreference.PULL_REQUEST_MERGE_STRATEGY,
    mergeOptions[0],
    option => option.method !== 'close'
  )
  useEffect(() => {
    if (allowedStrats) {
      const matchingMethods = mergeOptions.filter(option => allowedStrats.includes(option.method))
      if (matchingMethods.length > 0) {
        setMergeOption(matchingMethods[0])
      }
    } else {
      setMergeOption(mergeOptions[3])
    } // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [allowedStrats])
  const [draftOption, setDraftOption] = useState<PRDraftOption>(draftOptions[0])
  const permPushResult = hooks?.usePermissionTranslate?.(
    {
      resource: {
        resourceType: 'CODE_REPOSITORY'
      },
      permissions: ['code_repo_push']
    },
    [space]
  )
  const isActiveUserPROwner = useMemo(() => {
    return (
      !!currentUser?.uid && !!pullRequestMetadata?.author?.uid && currentUser?.uid === pullRequestMetadata?.author?.uid
    )
  }, [currentUser, pullRequestMetadata])

  if (pullRequestMetadata.state === PullRequestFilterOption.MERGED) {
    return <MergeInfo pullRequestMetadata={pullRequestMetadata} />
  }
  return (
    <Container
      className={cx(css.main, {
        [css.error]: mergeable === false && !unchecked && !isClosed && !isDraft,
        [css.unchecked]: unchecked,
        [css.closed]: isClosed,
        [css.draft]: isDraft,
        [css.ruleViolation]: ruleViolation
      })}>
      <Layout.Vertical spacing="xlarge">
        <Container>
          <Layout.Horizontal spacing="small" flex={{ alignItems: 'center' }} className={css.layout}>
            {(unchecked && <img src={Images.PrUnchecked} width={20} height={20} />) || (
              <Icon
                name={
                  isDraft
                    ? CodeIcon.Draft
                    : isClosed
                    ? 'issue'
                    : mergeable === false
                    ? 'warning-sign'
                    : ruleViolation
                    ? 'warning-sign'
                    : 'tick-circle'
                }
                size={20}
                color={
                  isDraft
                    ? Color.ORANGE_900
                    : isClosed
                    ? Color.GREY_500
                    : mergeable === false
                    ? Color.RED_500
                    : ruleViolation
                    ? Color.RED_500
                    : Color.GREEN_700
                }
              />
            )}
            <Text
              className={cx(css.sub, {
                [css.unchecked]: unchecked,
                [css.draft]: isDraft,
                [css.closed]: isClosed,
                [css.unmergeable]: mergeable === false && isOpen,
                [css.ruleViolate]: ruleViolation && !isClosed
              })}>
              {getString(
                isDraft
                  ? 'prState.draftHeading'
                  : isClosed
                  ? 'pr.prClosed'
                  : unchecked
                  ? 'pr.checkingToMerge'
                  : mergeable === false && isOpen
                  ? 'pr.cantBeMerged'
                  : ruleViolation
                  ? 'branchProtection.prFailedText'
                  : 'pr.branchHasNoConflicts',
                ruleViolation ? { ruleCount: length } : { ruleCount: 0 }
              )}
              {ruleViolation && mergeable && !isDraft ? (
                <Button
                  className={css.viewDetailsBtn}
                  rightIcon={'chevron-right'}
                  variation={ButtonVariation.LINK}
                  text={getString('prChecks.viewExternal')}
                  onClick={() => {
                    setActionBoxOpen(true)
                  }}
                />
              ) : null}
              <RuleViolationAlertModal
                setOpen={setActionBoxOpen}
                open={isActionBoxOpen}
                title={getString('branchProtection.mergePrAlertTitle')}
                text={getString('branchProtection.mergePrAlertText', { ruleCount: length })}
                rules={finalRulesArr}
              />
            </Text>
            <FlexExpander />
            <Render when={loading || loadingState}>
              <Icon name={CodeIcon.InputSpinner} size={16} margin={{ right: 'xsmall' }} />
            </Render>
            <Match expr={isDraft}>
              <Truthy>
                <SplitButton
                  text={draftOption.title}
                  disabled={loading}
                  className={css.secondaryButton}
                  variation={ButtonVariation.TERTIARY}
                  popoverProps={{
                    interactionKind: 'click',
                    usePortal: true,
                    popoverClassName: css.popover,
                    position: PopoverPosition.BOTTOM_RIGHT,
                    transitionDuration: 1000
                  }}
                  {...permissionProps(permPushResult, standalone)}
                  onClick={async () => {
                    if (draftOption.method === 'open') {
                      updatePRState({ is_draft: false, state: 'open' })
                        .then(onPRStateChanged)
                        .catch(exception => showError(getErrorMessage(exception)))
                    } else {
                      updatePRState({ state: 'closed' })
                        .then(onPRStateChanged)
                        .catch(exception => showError(getErrorMessage(exception)))
                    }
                  }}>
                  {draftOptions.map(option => {
                    return (
                      <Menu.Item
                        key={option.method}
                        className={css.menuItem}
                        disabled={option.disabled}
                        text={
                          <>
                            <BIcon icon={draftOption.method === option.method ? 'tick' : 'blank'} />
                            <strong>{option.title}</strong>
                            <p>{option.desc}</p>
                          </>
                        }
                        onClick={() => setDraftOption(option)}
                      />
                    )
                  })}
                </SplitButton>
              </Truthy>
              <Else>
                <Container>
                  <Match expr={pullRequestMetadata.state}>
                    <Case val={PullRequestState.CLOSED}>
                      <Button
                        className={css.secondaryButton}
                        text={getString('pr.openForReview')}
                        variation={ButtonVariation.TERTIARY}
                        onClick={() => {
                          const payload: OpenapiStatePullReqRequest = { state: 'open' }
                          updatePRState(payload)
                            .then(onPRStateChanged)
                            .catch(exception => showError(getErrorMessage(exception)))
                        }}
                      />
                    </Case>
                    <Case val={PullRequestState.OPEN}>
                      <Layout.Horizontal>
                        {!notBypassable && mergeable && !isDraft && ruleViolation ? (
                          <Checkbox
                            className={css.checkbox}
                            checked={bypass}
                            label={getString('branchProtection.mergeCheckboxAlert')}
                            onChange={event => {
                              setBypass(event.currentTarget.checked)
                            }}
                          />
                        ) : null}
                        <ReviewSplitButton
                          shouldHide={(pullRequestMetadata?.state as EnumPullReqState) === 'merged'}
                          repoMetadata={repoMetadata}
                          pullRequestMetadata={pullRequestMetadata}
                          refreshPr={onPRStateChanged}
                          disabled={isActiveUserPROwner}
                          refetchReviewers={refetchReviewers}
                        />
                        <Container
                          inline
                          padding={{ left: 'medium' }}
                          className={cx({
                            [css.btnWrapper]: mergeOption.method !== 'close',
                            [css.hasError]: mergeable === false,
                            [css.hasRuleViolated]: ruleViolation
                          })}>
                          <SplitButton
                            text={mergeOption.title}
                            disabled={
                              loading ||
                              unchecked ||
                              (isConflict && mergeOption.method !== 'close') ||
                              (ruleViolation && !bypass && mergeOption.method !== 'close')
                            }
                            className={cx({
                              [css.secondaryButton]: mergeOption.method === 'close' || mergeable === false
                            })}
                            variation={
                              mergeOption.method === 'close' || mergeable === false || ruleViolation
                                ? ButtonVariation.TERTIARY
                                : ButtonVariation.PRIMARY
                            }
                            popoverProps={{
                              interactionKind: 'click',
                              usePortal: true,
                              popoverClassName: css.popover,
                              position: PopoverPosition.BOTTOM_RIGHT,
                              transitionDuration: 1000
                            }}
                            {...permissionProps(permPushResult, standalone)}
                            onClick={async () => {
                              if (mergeOption.method !== 'close') {
                                const payload: OpenapiMergePullReq = {
                                  method: mergeOption.method,
                                  source_sha: pullRequestMetadata?.source_sha,
                                  bypass_rules: bypass,
                                  dry_run: false
                                }
                                mergePR(payload)
                                  .then(() => {
                                    onPRStateChanged()
                                    setRuleViolationArr(undefined)
                                  })
                                  .catch(exception => showError(getErrorMessage(exception)))
                              } else {
                                updatePRState({ state: 'closed' })
                                  .then(() => {
                                    resetMergeOption()
                                    onPRStateChanged()
                                    setRuleViolationArr(undefined)
                                  })
                                  .catch(exception => showError(getErrorMessage(exception)))
                              }
                            }}>
                            {mergeOptions.map(option => {
                              const mergeCheck = allowedStrats !== undefined && allowedStrats.includes(option.method)
                              return (
                                <Menu.Item
                                  key={option.method}
                                  className={css.menuItem}
                                  disabled={option.method !== 'close' ? !mergeCheck : option.disabled}
                                  text={
                                    <>
                                      <BIcon icon={mergeOption.method === option.method ? 'tick' : 'blank'} />
                                      <strong>{option.title}</strong>
                                      <p>{option.desc}</p>
                                    </>
                                  }
                                  onClick={() => setMergeOption(option)}
                                />
                              )
                            })}
                          </SplitButton>
                        </Container>
                      </Layout.Horizontal>
                    </Case>
                  </Match>
                </Container>
              </Else>
            </Match>
          </Layout.Horizontal>
        </Container>
      </Layout.Vertical>
    </Container>
  )
}

const MergeInfo: React.FC<{ pullRequestMetadata: TypesPullReq }> = ({ pullRequestMetadata }) => {
  const { getString } = useStrings()

  return (
    <Container className={cx(css.main, css.merged)}>
      <Layout.Horizontal spacing="medium" flex={{ alignItems: 'center' }} className={css.layout}>
        <Container width={24} height={24} className={css.mergeContainer}>
          <Icon name={CodeIcon.Merged} size={20} color={Color.PURPLE_700} />
        </Container>
        <Text className={cx(css.sub, css.merged)}>
          <StringSubstitute
            str={getString('pr.prMergedBannerInfo')}
            vars={{
              user: <strong>{pullRequestMetadata.merger?.display_name}</strong>,
              source: <strong>{pullRequestMetadata.source_branch}</strong>,
              target: <strong>{pullRequestMetadata.target_branch} </strong>,
              time: <ReactTimeago date={pullRequestMetadata.merged as number} />
            }}
          />
        </Text>
        <FlexExpander />
      </Layout.Horizontal>
    </Container>
  )
}
