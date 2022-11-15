import React from 'react'
import { Container, PageBody } from '@harness/uicore'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { useGetResourceContent } from 'hooks/useGetResourceContent'
import type { GitInfoProps } from 'utils/GitUtils'
import { RepositoryFileEditHeader } from './RepositoryFileEditHeader/RepositoryFileEditHeader'
import { FileEditor } from './FileEditor/FileEditor'
import css from './RepositoryFileEdit.module.scss'

export default function RepositoryFileEdit() {
  const { gitRef = '', resourcePath = '', repoMetadata, error, loading, refetch } = useGetRepositoryMetadata()

  return (
    <Container className={css.main}>
      <PageBody loading={loading} error={error} retryOnError={() => refetch()}>
        {repoMetadata ? (
          <ResourceContentFetcher repoMetadata={repoMetadata} gitRef={gitRef} resourcePath={resourcePath} />
        ) : null}
      </PageBody>
    </Container>
  )
}

function ResourceContentFetcher({
  repoMetadata,
  gitRef,
  resourcePath
}: Pick<GitInfoProps, 'repoMetadata' | 'gitRef' | 'resourcePath'>) {
  const { data: resourceContent /*error, loading, refetch */ } = useGetResourceContent({
    repoMetadata,
    gitRef,
    resourcePath
  })

  // TODO: Handle error, loading, etc...

  return resourceContent ? (
    <>
      <RepositoryFileEditHeader repoMetadata={repoMetadata} resourceContent={resourceContent} />
      <Container className={css.resourceContent}>
        <FileEditor
          repoMetadata={repoMetadata}
          gitRef={gitRef}
          resourcePath={resourcePath}
          resourceContent={resourceContent}
        />
      </Container>
    </>
  ) : null
}
