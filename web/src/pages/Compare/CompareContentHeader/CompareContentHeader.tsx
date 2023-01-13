import React from 'react'
import { Container, Layout, FlexExpander, ButtonVariation, Icon, Text, Color } from '@harness/uicore'
import { useHistory } from 'react-router-dom'
import { useAppContext } from 'AppContext'
import { useStrings } from 'framework/strings'
import { GitInfoProps, isRefATag } from 'utils/GitUtils'
import { BranchTagSelect } from 'components/BranchTagSelect/BranchTagSelect'
import { CreatePullRequestModalButton } from 'components/CreatePullRequestModal/CreatePullRequestModal'
import css from './CompareContentHeader.module.scss'

interface CompareContentHeaderProps extends Pick<GitInfoProps, 'repoMetadata'> {
  targetGitRef: string
  onTargetGitRefChanged: (gitRef: string) => void
  sourceGitRef: string
  onSourceGitRefChanged: (gitRef: string) => void
  mergeable?: boolean
}

export function CompareContentHeader({
  repoMetadata,
  targetGitRef,
  onTargetGitRefChanged,
  sourceGitRef,
  onSourceGitRefChanged
}: CompareContentHeaderProps) {
  const { getString } = useStrings()
  const history = useHistory()
  const { routes } = useAppContext()

  return (
    <Container className={css.main} padding="xlarge">
      <Layout.Horizontal spacing="medium">
        <Icon name="code-branch" size={20} />
        <BranchTagSelect
          repoMetadata={repoMetadata}
          disableBranchCreation
          disableViewAllBranches
          gitRef={targetGitRef}
          onSelect={onTargetGitRefChanged}
          labelPrefix={getString('prefixBase')}
          placeHolder={getString('selectBranchPlaceHolder')}
          style={{ '--background-color': 'var(--white)' } as React.CSSProperties}
        />
        <Icon name="arrow-left" size={14} />
        <BranchTagSelect
          repoMetadata={repoMetadata}
          disableBranchCreation
          disableViewAllBranches
          gitRef={sourceGitRef}
          onSelect={onSourceGitRefChanged}
          labelPrefix={getString('prefixCompare')}
          placeHolder={getString('selectBranchPlaceHolder')}
          style={{ '--background-color': 'var(--white)' } as React.CSSProperties}
        />
        {!!targetGitRef && !!sourceGitRef && <MergeableLabel mergeable />}
        <FlexExpander />
        <CreatePullRequestModalButton
          repoMetadata={repoMetadata}
          targetGitRef={targetGitRef}
          sourceGitRef={sourceGitRef}
          onSuccess={data => {
            history.replace(
              routes.toCODEPullRequest({
                repoPath: repoMetadata.path as string,
                pullRequestId: String(data.number)
              })
            )
          }}
          text={getString('createPullRequest')}
          variation={ButtonVariation.PRIMARY}
          disabled={
            !sourceGitRef ||
            !targetGitRef ||
            sourceGitRef === targetGitRef ||
            isRefATag(sourceGitRef) ||
            isRefATag(targetGitRef)
          }
          tooltip={
            isRefATag(sourceGitRef) || isRefATag(targetGitRef) ? getString('pullMustBeMadeFromBranches') : undefined
          }
          tooltipProps={{ isDark: true }}
        />
      </Layout.Horizontal>
    </Container>
  )
}

const MergeableLabel: React.FC<Pick<CompareContentHeaderProps, 'mergeable'>> = ({ mergeable }) => {
  const color = mergeable ? Color.GREEN_700 : Color.RED_500
  const { getString } = useStrings()

  return (
    <Text
      className={css.mergeText}
      icon={mergeable === true ? 'command-artifact-check' : 'cross'}
      iconProps={{ color }}
      color={color}>
      {getString(mergeable ? 'pr.ableToMerge' : 'pr.cantMerge')}
    </Text>
  )
}
