import React, { useMemo, useState } from 'react'
import { Container, Layout, FlexExpander, DropDown, Button, ButtonVariation, TextInput } from '@harness/uicore'
import { useStrings } from 'framework/strings'
import type { TypesRepository } from 'services/scm'
import { GitBranchType } from 'utils/GitUtils'
import css from './BranchesContentHeader.module.scss'

interface BranchesContentHeaderProps {
  activeBranchType?: GitBranchType
  repoMetadata: TypesRepository
  onBranchTypeSwitched: (branchType: GitBranchType) => void
  onSearchTermChanged: (searchTerm: string) => void
}

export function BranchesContentHeader({
  repoMetadata,
  onBranchTypeSwitched,
  onSearchTermChanged,
  activeBranchType = GitBranchType.ACTIVE
}: BranchesContentHeaderProps) {
  const { getString } = useStrings()
  const [branchType, setBranchType] = useState(activeBranchType)
  const [searchTerm, setSearchTerm] = useState('')
  const items = useMemo(
    () => [
      { label: getString('activeBranches'), value: GitBranchType.ACTIVE },
      { label: getString('inactiveBranches'), value: GitBranchType.INACTIVE },
      { label: getString('yourBranches'), value: GitBranchType.YOURS },
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
        <Button disabled text={getString('createBranch')} variation={ButtonVariation.PRIMARY} />
      </Layout.Horizontal>
    </Container>
  )
}
