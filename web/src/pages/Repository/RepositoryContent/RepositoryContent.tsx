import React from 'react'
import { Container } from '@harness/uicore'
import { useGet } from 'restful-react'
import type { RepositoryDTO } from 'types/SCMTypes'
import type { OpenapiGetContentOutput } from 'services/scm'
import { isDir, isFile } from 'utils/GitUtils'
import { ContentHeader } from './ContentHeader/ContentHeader'
import { FolderContent } from './FolderContent/FolderContent'
import { FileContent } from './FileContent/FileContent'
import css from './RepositoryContent.module.scss'

interface RepositoryContentProps {
  gitRef?: string
  resourcePath?: string
  repoMetadata: RepositoryDTO
}

export function RepositoryContent({ repoMetadata, gitRef, resourcePath }: RepositoryContentProps): JSX.Element {
  const { data /*error, loading, refetch, response */ } = useGet<OpenapiGetContentOutput>({
    path: `/api/v1/repos/${repoMetadata.path}/+/content${resourcePath ? '/' + resourcePath : ''}?include_commit=true${
      gitRef ? `&git_ref=${gitRef}` : ''
    }`
  })

  /* TODO: Handle loading and error */

  return (
    <Container padding="xlarge" className={css.resourceContent}>
      <ContentHeader
        repoMetadata={repoMetadata}
        gitRef={gitRef || repoMetadata.defaultBranch}
        resourcePath={resourcePath}
      />
      {data && isDir(data) && <FolderContent contentInfo={data} repoMetadata={repoMetadata} gitRef={gitRef} />}
      {data && isFile(data) && <FileContent contentInfo={data} />}
    </Container>
  )
}
