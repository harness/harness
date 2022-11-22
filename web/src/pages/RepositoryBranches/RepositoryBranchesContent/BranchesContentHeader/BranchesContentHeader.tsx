import React, { useMemo, useState } from 'react'
import { Container, Layout, FlexExpander, DropDown, ButtonVariation, TextInput } from '@harness/uicore'
import { useStrings } from 'framework/strings'
import { GitBranchType, CodeIcon, GitInfoProps } from 'utils/GitUtils'
import { CreateBranchModalButton } from 'components/CreateBranchModal/CreateBranchModal'
import css from './BranchesContentHeader.module.scss'

interface BranchesContentHeaderProps extends Pick<GitInfoProps, 'repoMetadata'> {
  activeBranchType?: GitBranchType
  onBranchTypeSwitched: (branchType: GitBranchType) => void
  onSearchTermChanged: (searchTerm: string) => void
  onNewBranchCreated: () => void
}

export function BranchesContentHeader({
  onBranchTypeSwitched,
  onSearchTermChanged,
  activeBranchType = GitBranchType.ALL,
  repoMetadata,
  onNewBranchCreated
}: BranchesContentHeaderProps) {
  const { getString } = useStrings()
  const [branchType, setBranchType] = useState(activeBranchType)
  const [searchTerm, setSearchTerm] = useState('')
  const items = useMemo(
    () => [
      { label: getString('activeBranches'), value: GitBranchType.ACTIVE },
      { label: getString('inactiveBranches'), value: GitBranchType.INACTIVE },
      // { label: getString('yourBranches'), value: GitBranchType.YOURS },
      { label: getString('allBranches'), value: GitBranchType.ALL }
    ],
    [getString]
  )

  return (
    <Container className={css.main}>
      <Layout.Horizontal spacing="medium">
        <DropDown
          value={branchType}
          items={items}
          onChange={({ value }) => {
            setBranchType(value as GitBranchType)
            onBranchTypeSwitched(value as GitBranchType)
          }}
          popoverClassName={css.branchDropdown}
        />
        <FlexExpander />
        <TextInput
          placeholder={getString('searchBranches')}
          autoFocus
          onFocus={event => event.target.select()}
          value={searchTerm}
          onInput={event => {
            const value = event.currentTarget.value
            setSearchTerm(value)
            onSearchTermChanged(value)
          }}
        />
        <CreateBranchModalButton
          text={getString('createBranch')}
          icon={CodeIcon.Add}
          variation={ButtonVariation.PRIMARY}
          repoMetadata={repoMetadata}
          onSuccess={onNewBranchCreated}
          showSuccessMessage
        />
      </Layout.Horizontal>
    </Container>
  )
}
