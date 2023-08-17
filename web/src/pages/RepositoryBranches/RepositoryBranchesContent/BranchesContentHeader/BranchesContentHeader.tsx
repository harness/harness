import React, { useState } from 'react'
import { Container, Layout, FlexExpander, ButtonVariation } from '@harness/uicore'
import { useStrings } from 'framework/strings'
import { GitBranchType, CodeIcon, GitInfoProps } from 'utils/GitUtils'
import { SearchInputWithSpinner } from 'components/SearchInputWithSpinner/SearchInputWithSpinner'
import { CreateBranchModalButton } from 'components/CreateBranchModal/CreateBranchModal'
import css from './BranchesContentHeader.module.scss'

interface BranchesContentHeaderProps extends Pick<GitInfoProps, 'repoMetadata'> {
  loading?: boolean
  activeBranchType?: GitBranchType
  onBranchTypeSwitched: (branchType: GitBranchType) => void
  onSearchTermChanged: (searchTerm: string) => void
  onNewBranchCreated: () => void
}

export function BranchesContentHeader({
  // onBranchTypeSwitched,
  onSearchTermChanged,
  // activeBranchType = GitBranchType.ALL,
  repoMetadata,
  onNewBranchCreated,
  loading
}: BranchesContentHeaderProps) {
  const { getString } = useStrings()
  // const [branchType, setBranchType] = useState(activeBranchType)
  const [searchTerm, setSearchTerm] = useState('')
  // const items = useMemo(
  //   () => [
  //     { label: getString('activeBranches'), value: GitBranchType.ACTIVE },
  //     { label: getString('inactiveBranches'), value: GitBranchType.INACTIVE },
  //     // { label: getString('yourBranches'), value: GitBranchType.YOURS },
  //     { label: getString('allBranches'), value: GitBranchType.ALL }
  //   ],
  //   [getString]
  // )

  return (
    <Container className={css.main}>
      <Layout.Horizontal spacing="medium">
        {/* <DropDown
          value={branchType}
          items={items}
          onChange={({ value }) => {
            setBranchType(value as GitBranchType)
            onBranchTypeSwitched(value as GitBranchType)
          }}
          popoverClassName={css.branchDropdown}
        /> */}
        <SearchInputWithSpinner
          loading={loading}
          spinnerPosition="right"
          query={searchTerm}
          setQuery={value => {
            setSearchTerm(value)
            onSearchTermChanged(value)
          }}
        />
        <FlexExpander />
        <CreateBranchModalButton
          text={getString('newBranch')}
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
