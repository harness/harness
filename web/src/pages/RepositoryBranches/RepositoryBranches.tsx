import React from 'react'
import { Container, PageBody } from '@harness/uicore'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { RepositoryBranchesContent } from './RepositoryBranchesContent/RepositoryBranchesContent'
import { RepositoryBranchesHeader } from './RepositoryBranchesHeader/RepositoryBranchessHeader'
import css from './RepositoryBranches.module.scss'

export default function RepositoryBranches() {
  const { repoMetadata, error, loading, commitRef, refetch } = useGetRepositoryMetadata()

  return (
    <Container className={css.main}>
      <PageBody loading={loading} error={error} retryOnError={() => refetch()}>
        {repoMetadata ? (
          <>
            <RepositoryBranchesHeader repoMetadata={repoMetadata} />
            <RepositoryBranchesContent
              repoMetadata={repoMetadata}
              commitRef={commitRef || (repoMetadata.defaultBranch as string)}
            />
          </>
        ) : null}
      </PageBody>
    </Container>
  )
}
