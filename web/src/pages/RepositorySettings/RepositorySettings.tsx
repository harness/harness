import React from 'react'
import { Container, PageBody } from '@harness/uicore'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import css from './RepositorySettings.module.scss'

export default function RepositoryBranches() {
  const { repoMetadata, error, loading } = useGetRepositoryMetadata()

  return (
    <Container className={css.main}>
      <PageBody loading={loading} error={error}>
        {repoMetadata ? <>{/* To be implemented... */}</> : null}
      </PageBody>
    </Container>
  )
}
