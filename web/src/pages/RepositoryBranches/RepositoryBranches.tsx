import React from 'react'
import { Container, PageBody } from '@harness/uicore'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { useStrings } from 'framework/strings'
import { RepositoryPageHeader } from 'components/RepositoryPageHeader/RepositoryPageHeader'
import { getErrorMessage } from 'utils/Utils'
import { RepositoryBranchesContent } from './RepositoryBranchesContent/RepositoryBranchesContent'
import css from './RepositoryBranches.module.scss'

export default function RepositoryBranches() {
  const { getString } = useStrings()
  const { repoMetadata, error, loading, refetch } = useGetRepositoryMetadata()

  return (
    <Container className={css.main}>
      <RepositoryPageHeader
        repoMetadata={repoMetadata}
        title={getString('branches')}
        dataTooltipId="repositoryBranches"
      />
      <PageBody loading={loading} error={getErrorMessage(error)} retryOnError={() => refetch()}>
        {repoMetadata ? <RepositoryBranchesContent repoMetadata={repoMetadata} /> : null}
      </PageBody>
    </Container>
  )
}
