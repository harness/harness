import React from 'react'
import { Container, PageBody } from '@harness/uicore'
import { useGetResourceContent } from 'hooks/useGetResourceContent'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { RepositoryContent } from './RepositoryContent/RepositoryContent'
import { RepositoryHeader } from './RepositoryHeader/RepositoryHeader'
import css from './Repository.module.scss'

export default function Repository() {
  const { gitRef, resourcePath, repoMetadata, error, loading, refetch } = useGetRepositoryMetadata()
  const {
    data: resourceContent,
    error: resourceError,
    loading: resourceLoading
  } = useGetResourceContent({ repoMetadata, gitRef, resourcePath, includeCommit: true })

  return (
    <Container className={css.main}>
      <PageBody loading={loading || resourceLoading} error={error || resourceError} retryOnError={() => refetch()}>
        {!!repoMetadata && (
          <>
            <RepositoryHeader repoMetadata={repoMetadata} />

            {!!resourceContent && (
              <RepositoryContent
                repoMetadata={repoMetadata}
                gitRef={gitRef}
                resourcePath={resourcePath}
                resourceContent={resourceContent}
              />
            )}
          </>
        )}
      </PageBody>
    </Container>
  )
}
