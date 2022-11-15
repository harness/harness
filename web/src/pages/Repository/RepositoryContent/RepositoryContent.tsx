import React from 'react'
import { Container } from '@harness/uicore'
import { GitInfoProps, isDir, isFile } from 'utils/GitUtils'
import { useGetResourceContent } from 'hooks/useGetResourceContent'
import { ContentHeader } from './ContentHeader/ContentHeader'
import { FolderContent } from './FolderContent/FolderContent'
import { FileContent } from './FileContent/FileContent'
import css from './RepositoryContent.module.scss'

export function RepositoryContent({
  repoMetadata,
  gitRef,
  resourcePath
}: Pick<GitInfoProps, 'repoMetadata' | 'gitRef' | 'resourcePath'>) {
  const { data /*error, loading, refetch, response */ } = useGetResourceContent({
    repoMetadata,
    gitRef,
    resourcePath,
    includeCommit: true
  })

  /* TODO: Handle loading and error */

  return (
    <Container padding="xlarge" className={css.resourceContent}>
      {!!data && (
        <>
          <ContentHeader
            repoMetadata={repoMetadata}
            gitRef={gitRef}
            resourcePath={resourcePath}
            resourceContent={data}
          />
          {(isDir(data) && (
            <FolderContent
              resourceContent={data}
              repoMetadata={repoMetadata}
              gitRef={gitRef || (repoMetadata.defaultBranch as string)}
            />
          )) || (
            <FileContent
              repoMetadata={repoMetadata}
              gitRef={gitRef}
              resourcePath={resourcePath}
              resourceContent={data}
            />
          )}
        </>
      )}
    </Container>
  )
}
