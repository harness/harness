import React, { useState } from 'react'
import { Container, Layout, FlexExpander, ButtonVariation } from '@harness/uicore'
import { useStrings } from 'framework/strings'
import { GitBranchType, CodeIcon, GitInfoProps } from 'utils/GitUtils'
import { SearchInputWithSpinner } from 'components/SearchInputWithSpinner/SearchInputWithSpinner'
import { CreateTagModalButton } from 'components/CreateTagModal/CreateTagModal'
import css from './RepositoryTagsContentHeader.module.scss'

interface RepositoryTagsContentHeaderProps extends Pick<GitInfoProps, 'repoMetadata'> {
  loading?: boolean
  activeBranchType?: GitBranchType
  onBranchTypeSwitched: (branchType: GitBranchType) => void
  onSearchTermChanged: (searchTerm: string) => void
  onNewBranchCreated: () => void
}

export function RepositoryTagsContentHeader({
  onSearchTermChanged,
  repoMetadata,
  onNewBranchCreated,
  loading
}: RepositoryTagsContentHeaderProps) {
  const { getString } = useStrings()
  const [searchTerm, setSearchTerm] = useState('')
  

  return (
    <Container className={css.main}>
      <Layout.Horizontal spacing="medium">
        <SearchInputWithSpinner
          loading={loading}
          query={searchTerm}
          setQuery={value => {
            setSearchTerm(value)
            onSearchTermChanged(value)
          }}
        />
        <FlexExpander />
        <CreateTagModalButton
          text={getString('createTag')}
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
