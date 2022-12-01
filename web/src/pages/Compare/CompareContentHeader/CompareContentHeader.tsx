import React from 'react'
import { Container, Layout, FlexExpander, ButtonVariation, Button, Icon } from '@harness/uicore'
import { useStrings } from 'framework/strings'
import { GitInfoProps, isRefATag } from 'utils/GitUtils'
import { BranchTagSelect } from 'components/BranchTagSelect/BranchTagSelect'
import css from './CompareContentHeader.module.scss'

interface CompareContentHeaderProps extends Pick<GitInfoProps, 'repoMetadata'> {
  baseRef: string
  compareRef: string
  onBaseRefChanged: (ref: string) => void
  onCompareRefChanged: (ref: string) => void
}

export function CompareContentHeader({
  repoMetadata,
  baseRef,
  compareRef,
  onBaseRefChanged,
  onCompareRefChanged
}: CompareContentHeaderProps) {
  const { getString } = useStrings()

  return (
    <Container className={css.main} padding="xlarge">
      <Layout.Horizontal spacing="medium">
        <Icon name="code-branch" size={20} />
        <BranchTagSelect
          repoMetadata={repoMetadata}
          disableBranchCreation
          disableViewAllBranches
          gitRef={baseRef}
          onSelect={onBaseRefChanged}
          labelPrefix={getString('prefixBase')}
          placeHolder={getString('selectBranchPlaceHolder')}
        />
        <Icon name="arrow-left" size={14} />
        <BranchTagSelect
          repoMetadata={repoMetadata}
          disableBranchCreation
          disableViewAllBranches
          gitRef={compareRef}
          onSelect={onCompareRefChanged}
          labelPrefix={getString('prefixCompare')}
          placeHolder={getString('selectBranchPlaceHolder')}
        />
        <FlexExpander />
        <Button
          text={getString('createPullRequest')}
          variation={ButtonVariation.PRIMARY}
          disabled={!baseRef || !compareRef || baseRef === compareRef || isRefATag(baseRef) || isRefATag(compareRef)}
          tooltip={isRefATag(baseRef) || isRefATag(compareRef) ? getString('pullMustBeMadeFromBranches') : undefined}
          tooltipProps={{ isDark: true }}
        />
      </Layout.Horizontal>
    </Container>
  )
}
