import React from 'react'
import { Container, PageBody } from '@harness/uicore'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { RepositoryFileEditContent } from './RepositoryFileEditContent/RepositoryFileEditContent'
import { RepositoryFileEditHeader } from './RepositoryFileEditHeader/RepositoryFileEditHeader'
import css from './RepositoryFileEdit.module.scss'

export default function RepositoryFileEdit() {
  const { gitRef = '', resourcePath = '', repoMetadata, error, loading } = useGetRepositoryMetadata()

  return (
    <Container className={css.main}>
      <PageBody loading={loading} error={error}>
        {repoMetadata ? (
          <>
            <RepositoryFileEditHeader repoMetadata={repoMetadata} resourcePath={resourcePath} />
            <RepositoryFileEditContent repoMetadata={repoMetadata} gitRef={gitRef} resourcePath={resourcePath} />
          </>
        ) : null}
      </PageBody>
    </Container>
  )
}
