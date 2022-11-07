import React from 'react'
import { Container } from '@harness/uicore'
import type { TypesRepository } from 'services/scm'
import { isDir, isFile } from 'utils/GitUtils'
import { useGetResourceContent } from 'hooks/useGetResourceContent'
import { ContentHeader } from './ContentHeader/ContentHeader'
import { FolderContent } from './FolderContent/FolderContent'
import { FileContent } from './FileContent/FileContent'
import css from './RepositoryContent.module.scss'

interface RepositoryContentProps {
  repoMetadata: TypesRepository
  gitRef: string
  resourcePath: string
}

export function RepositoryContent({ repoMetadata, gitRef, resourcePath }: RepositoryContentProps) {
  const { data /*error, loading, refetch, response */ } = useGetResourceContent({
    repoMetadata,
    gitRef,
    resourcePath,
    includeCommit: true
  })

  /* TODO: Handle loading and error */

  return (
    <Container padding="xlarge" className={css.resourceContent}>
      <ContentHeader
        repoMetadata={repoMetadata}
        gitRef={gitRef || repoMetadata.defaultBranch}
        resourcePath={resourcePath}
      />
      {data && isDir(data) && (
        <FolderContent
          contentInfo={data}
          repoMetadata={repoMetadata}
          gitRef={gitRef || (repoMetadata.defaultBranch as string)}
        />
      )}
      {data && isFile(data) && (
        <FileContent repoMetadata={repoMetadata} gitRef={gitRef} resourcePath={resourcePath} contentInfo={data} />
      )}
    </Container>
  )
}
