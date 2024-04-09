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

import React, { useEffect, useMemo, useRef, useState } from 'react'
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
  useIsMounted,
  useToaster
} from '@harnessio/uicore'
import { Icon } from '@harnessio/icons'
import { Color } from '@harnessio/design-system'
import { useMutate } from 'restful-react'
import { Case, Else, Match, Render, Truthy } from 'react-jsx-match'
import { Menu, PopoverPosition, Icon as BIcon } from '@blueprintjs/core'
import cx from 'classnames'
import ReactTimeago from 'react-timeago'
import type { OpenapiStatePullReqRequest, TypesPullReq, TypesRuleViolations } from 'services/code'
import { useStrings } from 'framework/strings'
import { CodeIcon, PullRequestFilterOption, PullRequestState } from 'utils/GitUtils'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { useAppContext } from 'AppContext'
import {
  dryMerge,
  extractInfoFromRuleViolationArr,
  getErrorMessage,
  MergeCheckStatus,
  permissionProps,
  PRDraftOption,
  PRMergeOption,
  PullRequestActionsBoxProps
} from 'utils/Utils'
import { OptionsMenuButton } from 'components/OptionsMenuButton/OptionsMenuButton'
import { UserPreference, useUserPreference } from 'hooks/useUserPreference'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import MergeSideDialogBox from './MergeSideDialogBox'
import css from './PullRequestActionsBox.module.scss'

export const PullRequestActionsBox: React.FC<PullRequestActionsBoxProps> = ({
  repoMetadata,
  pullReqMetadata,
  onPRStateChanged,
  allowedStrategy
}) => {
  const { getString } = useStrings()
  const { showError } = useToaster()
  const { hooks, standalone } = useAppContext()
  const space = useGetSpaceParam()
  const { pullRequestSection } = useGetRepositoryMetadata()
  const { mutate: mergePR, loading } = useMutate({
    verb: 'POST',
    path: `/api/v1/repos/${repoMetadata.path}/+/pullreq/${pullReqMetadata.number}/merge`
  })
  const [ruleViolation, setRuleViolation] = useState(false)
  const [ruleViolationArr, setRuleViolationArr] = useState<{ data: { rule_violations: TypesRuleViolations[] } }>()
  const [notBypassable, setNotBypassable] = useState(false)
  const [bypass, setBypass] = useState(false)
  const { mutate: updatePRState, loading: loadingState } = useMutate({
    verb: 'POST',
    path: `/api/v1/repos/${repoMetadata.path}/+/pullreq/${pullReqMetadata.number}/state`
  })
  const mergeable = useMemo(() => pullReqMetadata.merge_check_status === MergeCheckStatus.MERGEABLE, [pullReqMetadata])
  const isClosed = pullReqMetadata.state === PullRequestState.CLOSED
  const isOpen = pullReqMetadata.state === PullRequestState.OPEN
  const isConflict = pullReqMetadata.merge_check_status === MergeCheckStatus.CONFLICT
  const isMounted = useIsMounted()
  const unchecked = useMemo(
    () => pullReqMetadata.merge_check_status === MergeCheckStatus.UNCHECKED && !isClosed,
    [pullReqMetadata, isClosed]
  )
  const [sideDialogOpen, setSideDialogOpen] = useState(false)
  // Flags to optimize rendering
  const internalFlags = useRef({ dryRun: false })
  useEffect(() => {
    if (ruleViolationArr && !isDraft && ruleViolationArr.data.rule_violations) {
      const { checkIfBypassAllowed } = extractInfoFromRuleViolationArr(ruleViolationArr.data.rule_violations)
      setNotBypassable(checkIfBypassAllowed)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [ruleViolationArr])

  useEffect(() => {
    // recheck PR in case source SHA changed or PR was marked as unchecked
    // TODO: optimize call to handle all causes and avoid double calls by keeping track of SHA
    dryMerge(
      isMounted,
      isClosed,
      pullReqMetadata,
      internalFlags,
      mergePR,
      setRuleViolation,
      setRuleViolationArr,
      setAllowedStrats,
      pullRequestSection,
      showError
    ) // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [unchecked, pullReqMetadata?.source_sha])
  const [prMerged, setPrMerged] = useState(false)

  useEffect(() => {
    // dryMerge()
    const intervalId = setInterval(async () => {
      if (!prMerged) {
        dryMerge(
          isMounted,
          isClosed,
          pullReqMetadata,
          internalFlags,
          mergePR,
          setRuleViolation,
          setRuleViolationArr,
          setAllowedStrats,
          pullRequestSection,
          showError
        )
      }
    }, POLLING_INTERVAL) // Poll every 20 seconds
    // Cleanup interval on component unmount
    return () => {
      clearInterval(intervalId)
    } // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [onPRStateChanged, prMerged])
  const isDraft = pullReqMetadata.is_draft
  const mergeOptions: PRMergeOption[] = [
    {
      method: 'squash',
      title: getString('pr.mergeOptions.squashAndMerge'),
      desc: getString('pr.mergeOptions.squashAndMergeDesc'),
      disabled: mergeable === false,
      label: getString('pr.mergeOptions.squashAndMerge'),
      value: 'squash'
    },
    {
      method: 'merge',
      title: getString('pr.mergeOptions.createMergeCommit'),
      desc: getString('pr.mergeOptions.createMergeCommitDesc'),
      disabled: mergeable === false,
      label: getString('pr.mergeOptions.createMergeCommit'),
      value: 'merge'
    },
    {
      method: 'rebase',
      title: getString('pr.mergeOptions.rebaseAndMerge'),
      desc: getString('pr.mergeOptions.rebaseAndMergeDesc'),
      disabled: mergeable === false,
      label: getString('pr.mergeOptions.rebaseAndMerge'),
      value: 'rebase'
    },

    {
      method: 'close',
      title: getString('pr.mergeOptions.close'),
      desc: getString('pr.mergeOptions.closeDesc'),
      label: getString('pr.mergeOptions.close'),
      value: 'close'
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
  }, [allowedStrats, allowedStrategy])

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

  if (pullReqMetadata.state === PullRequestFilterOption.MERGED) {
    return <MergeInfo pullRequestMetadata={pullReqMetadata} />
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
                  ? 'branchProtection.prFailedText'
                  : ruleViolation
                  ? 'branchProtection.prFailedText'
                  : 'pr.branchHasNoConflicts'
              )}
            </Text>
            <FlexExpander />
            <Render when={loading || loadingState}>
              <Icon name={CodeIcon.InputSpinner} size={16} margin={{ right: 'xsmall' }} />
            </Render>
            <Match expr={isDraft}>
              <Truthy>
                <SplitButton
                  text={draftOption.title}
                  disabled={loading && !internalFlags.current.dryRun}
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
                  <Match expr={pullReqMetadata.state}>
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

                        <Container
                          inline
                          padding={{ left: 'medium' }}
                          className={cx({
                            [css.btnWrapper]: mergeOption.method !== 'close',
                            [css.hasError]: mergeable === false,
                            [css.hasRuleViolated]: ruleViolation,
                            [css.bypass]: bypass
                          })}>
                          <Button
                            text={getString('pr.mergePR')}
                            disabled={
                              (loading && !internalFlags.current.dryRun) ||
                              (unchecked && mergeOption.method !== 'close') ||
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
                            {...permissionProps(permPushResult, standalone)}
                            onClick={async () => {
                              setSideDialogOpen(true)
                            }}></Button>
                        </Container>
                        <OptionsMenuButton
                          className={css.optionMenuButton}
                          {...permissionProps(permPushResult, standalone)}
                          isDark
                          items={[
                            {
                              hasIcon: true,
                              iconName: CodeIcon.Draft,
                              text: getString('markAsDraft'),
                              onClick: () => {
                                updatePRState({ is_draft: true, state: 'open' })
                                  .then(onPRStateChanged)
                                  .catch(exception => showError(getErrorMessage(exception)))
                              }
                            },
                            {
                              hasIcon: true,
                              iconName: 'code-rejected',
                              text: getString('pr.mergeOptions.close'),
                              onClick: async () => {
                                updatePRState({ state: 'closed' })
                                  .then(() => {
                                    resetMergeOption()
                                    onPRStateChanged()
                                    setRuleViolationArr(undefined)
                                  })
                                  .catch(exception => showError(getErrorMessage(exception)))
                              }
                            }
                          ]}
                          tooltipProps={{
                            isDark: true,
                            position: PopoverPosition.RIGHT,
                            popoverClassName: css.overviewPopover
                          }}
                        />
                      </Layout.Horizontal>
                    </Case>
                  </Match>
                </Container>
              </Else>
            </Match>
          </Layout.Horizontal>
        </Container>
      </Layout.Vertical>
      {sideDialogOpen && (
        <MergeSideDialogBox
          mergeOption={mergeOption}
          sideDialogOpen={sideDialogOpen}
          setSideDialogOpen={setSideDialogOpen}
          mergeOptions={mergeOptions}
          allowedStrats={allowedStrats}
          mergeable={mergeable}
          ruleViolation={ruleViolation}
          mergePR={mergePR}
          setPrMerged={setPrMerged}
          pullReqMetadata={pullReqMetadata}
          onPRStateChanged={onPRStateChanged}
          setRuleViolationArr={setRuleViolationArr}
          bypass={bypass}
          setMergeOption={setMergeOption}
        />
      )}
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

const POLLING_INTERVAL = 10000
