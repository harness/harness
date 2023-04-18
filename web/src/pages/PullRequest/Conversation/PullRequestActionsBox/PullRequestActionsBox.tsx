import React, { useState } from 'react'
import {
  Button,
  ButtonVariation,
  Color,
  Container,
  FlexExpander,
  Icon,
  Layout,
  SplitButton,
  StringSubstitute,
  Text,
  useToaster
} from '@harness/uicore'
import { useMutate } from 'restful-react'
import { Case, Else, Match, Render, Truthy } from 'react-jsx-match'
import { Menu, PopoverPosition, Icon as BIcon } from '@blueprintjs/core'
import cx from 'classnames'
import ReactTimeago from 'react-timeago'
import type {
  EnumMergeMethod,
  EnumPullReqState,
  OpenapiMergePullReq,
  OpenapiStatePullReqRequest,
  TypesPullReq
} from 'services/code'
import { useStrings } from 'framework/strings'
import { CodeIcon, GitInfoProps, PullRequestFilterOption, PullRequestState } from 'utils/GitUtils'
import { getErrorMessage } from 'utils/Utils'
import ReviewSplitButton from 'components/Changes/ReviewSplitButton/ReviewSplitButton'
import css from './PullRequestActionsBox.module.scss'

interface PullRequestActionsBoxProps extends Pick<GitInfoProps, 'repoMetadata' | 'pullRequestMetadata'> {
  onPRStateChanged: () => void
}

interface PRMergeOption {
  method: EnumMergeMethod | 'close'
  title: string
  desc: string
  disabled?: boolean
}

export const PullRequestActionsBox: React.FC<PullRequestActionsBoxProps> = ({
  repoMetadata,
  pullRequestMetadata,
  onPRStateChanged
}) => {
  const { getString } = useStrings()
  const { showError } = useToaster()
  const { mutate: mergePR, loading } = useMutate({
    verb: 'POST',
    path: `/api/v1/repos/${repoMetadata.path}/+/pullreq/${pullRequestMetadata.number}/merge`
  })
  const { mutate: updatePRState, loading: loadingState } = useMutate({
    verb: 'POST',
    path: `/api/v1/repos/${repoMetadata.path}/+/pullreq/${pullRequestMetadata.number}/state`
  })
  const mergeable = pullRequestMetadata.merge_check_status === 'mergeable'
  const isDraft = pullRequestMetadata.is_draft
  const mergeOptions: PRMergeOption[] = [
    {
      method: 'squash',
      title: getString('pr.mergeOptions.squashAndMerge'),
      desc: getString('pr.mergeOptions.squashAndMergeDesc'),
      disabled: true
    },
    {
      method: 'merge',
      title: getString('pr.mergeOptions.createMergeCommit'),
      desc: getString('pr.mergeOptions.createMergeCommitDesc')
    },
    {
      method: 'rebase',
      title: getString('pr.mergeOptions.rebaseAndMerge'),
      desc: getString('pr.mergeOptions.rebaseAndMergeDesc'),
      disabled: true
    },
    {
      method: 'close',
      title: getString('pr.mergeOptions.close'),
      desc: getString('pr.mergeOptions.closeDesc')
    }
  ]
  const [mergeOption, setMergeOption] = useState<PRMergeOption>(mergeOptions[1])
  if (pullRequestMetadata.state === PullRequestFilterOption.MERGED) {
    return <MergeInfo pullRequestMetadata={pullRequestMetadata} />
  }

  return (
    <Container
      className={cx(css.main, {
        [css.error]: mergeable === false
      })}>
      <Layout.Vertical spacing="xlarge">
        <Container>
          <Layout.Horizontal spacing="small" flex={{ alignItems: 'center' }} className={css.layout}>
            <Icon
              name={isDraft ? CodeIcon.Draft : mergeable === false ? 'warning-sign' : 'tick-circle'}
              size={20}
              color={isDraft ? Color.ORANGE_900 : mergeable === false ? Color.RED_500 : Color.GREEN_700}
            />
            <Text className={css.sub}>
              {getString(
                isDraft ? 'prState.draftHeading' : mergeable === false ? 'pr.cantBeMerged' : 'pr.branchHasNoConflicts'
              )}
            </Text>

            <FlexExpander />
            <Render when={loading || loadingState}>
              <Icon name={CodeIcon.InputSpinner} size={16} margin={{ right: 'xsmall' }} />
            </Render>
            <Match expr={isDraft}>
              <Truthy>
                <Button
                  className={css.secondaryButton}
                  text={getString('pr.readyForReview')}
                  variation={ButtonVariation.TERTIARY}
                  onClick={() => {
                    const payload: OpenapiStatePullReqRequest = { is_draft: false, state: 'open' }

                    updatePRState(payload)
                      .then(onPRStateChanged)
                      .catch(exception => showError(getErrorMessage(exception)))
                  }}
                />
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
                        <ReviewSplitButton
                          shouldHide={(pullRequestMetadata?.state as EnumPullReqState) === 'merged'}
                          repoMetadata={repoMetadata}
                          pullRequestMetadata={pullRequestMetadata}
                          refreshPr={onPRStateChanged}
                        />
                        <Container
                          inline
                          padding={{ left: 'medium' }}
                          className={cx({
                            [css.btnWrapper]: mergeOption.method !== 'close',
                            [css.hasError]: mergeable === false
                          })}>
                          <SplitButton
                            text={mergeOption.title}
                            disabled={loading}
                            className={cx({
                              [css.secondaryButton]: mergeOption.method === 'close' || mergeable === false
                            })}
                            variation={
                              mergeOption.method === 'close' || mergeable === false
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
                            onClick={() => {
                              if (mergeOption.method !== 'close') {
                                const payload: OpenapiMergePullReq = { method: mergeOption.method }

                                mergePR(payload)
                                  .then(onPRStateChanged)
                                  .catch(exception => showError(getErrorMessage(exception)))
                              } else {
                                const payload: OpenapiStatePullReqRequest = { state: 'closed' }

                                updatePRState(payload)
                                  .then(onPRStateChanged)
                                  .catch(exception => showError(getErrorMessage(exception)))
                              }
                            }}>
                            {/* TODO: These two items are used for creating a PR
                          <Menu.Item
                        className={css.menuItem}
                        text={
                          <>
                            <BIcon icon="blank" />
                            <strong>Create pull request</strong>
                            <p>Open a pull request that is ready for review</p>
                            <p>Automatically request reviews from code owners</p>
                          </>
                        }
                      />
                      <Menu.Item
                        className={css.menuItem}
                        text={
                          <>
                            <BIcon icon="blank" />
                            <strong>Create draft pull request</strong>
                            <p>Does not request code reviews and cannot be merged</p>
                            <p>Cannot be merged until marked ready for review</p>
                          </>
                        }
                      /> */}
                            {mergeOptions.map(option => {
                              return (
                                <Menu.Item
                                  key={option.method}
                                  className={css.menuItem}
                                  disabled={option.disabled}
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
        <Container>
          {/* <Text className={css.heading}>{getString('pr.prMerged')}</Text> */}
          <Text className={css.sub}>
            <StringSubstitute
              str={getString('pr.prMergedInfo')}
              vars={{
                user: <strong>{pullRequestMetadata.merger?.display_name}</strong>,
                source: <strong>{pullRequestMetadata.source_branch}</strong>,
                target: <strong>{pullRequestMetadata.target_branch} </strong>,
                time: <ReactTimeago date={pullRequestMetadata.merged as number} />
              }}
            />
          </Text>
        </Container>
        <FlexExpander />
      </Layout.Horizontal>
    </Container>
  )
}
