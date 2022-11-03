import React from 'react'
import { Container, PageBody } from '@harness/uicore'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { RepositoryContent } from './RepositoryContent/RepositoryContent'
import { RepositoryHeader } from './RepositoryHeader/RepositoryHeader'
import css from './Repository.module.scss'

export default function Repository() {
  const { gitRef = '', resourcePath = '', repoMetadata, error, loading } = useGetRepositoryMetadata()

  return (
    <Container className={css.main}>
      <PageBody loading={loading} error={error}>
        {repoMetadata ? (
          <>
            <RepositoryHeader repoMetadata={repoMetadata} />
            <RepositoryContent repoMetadata={repoMetadata} gitRef={gitRef} resourcePath={resourcePath} />
          </>
        ) : null}
      </PageBody>
    </Container>
  )
}
