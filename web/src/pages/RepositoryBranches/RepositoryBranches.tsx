import React from 'react'
import { Container, PageBody } from '@harnessio/uicore'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { useStrings } from 'framework/strings'
import { RepositoryPageHeader } from 'components/RepositoryPageHeader/RepositoryPageHeader'
import { LoadingSpinner } from 'components/LoadingSpinner/LoadingSpinner'
import { voidFn, getErrorMessage } from 'utils/Utils'
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
      <PageBody error={getErrorMessage(error)} retryOnError={voidFn(refetch)}>
        <LoadingSpinner visible={loading} />

        {repoMetadata ? <RepositoryBranchesContent repoMetadata={repoMetadata} /> : null}
      </PageBody>
    </Container>
  )
}
