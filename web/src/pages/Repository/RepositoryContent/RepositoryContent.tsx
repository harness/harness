import React, { useEffect } from 'react'
import { Container } from '@harness/uicore'
import { GitInfoProps, isDir } from 'utils/GitUtils'
import { ContentHeader } from './ContentHeader/ContentHeader'
import { FolderContent } from './FolderContent/FolderContent'
import { FileContent } from './FileContent/FileContent'
import css from './RepositoryContent.module.scss'

export function RepositoryContent({
  repoMetadata,
  gitRef,
  resourcePath,
  resourceContent,
  commitRef
}: Pick<GitInfoProps, 'repoMetadata' | 'gitRef' | 'resourcePath' | 'resourceContent' | 'commitRef'>) {
  useEffect(() => {
    window.scroll({ top: 0 })
  }, [gitRef, resourcePath])

  return (
    <Container className={css.resourceContent}>
      <ContentHeader
        repoMetadata={repoMetadata}
        gitRef={gitRef}
        resourcePath={resourcePath}
        resourceContent={resourceContent}
      />
      {(isDir(resourceContent) && (
        <FolderContent
          resourceContent={resourceContent}
          repoMetadata={repoMetadata}
          gitRef={gitRef || (repoMetadata.default_branch as string)}
        />
      )) || (
        <FileContent
          repoMetadata={repoMetadata}
          gitRef={gitRef}
          resourcePath={resourcePath}
          resourceContent={resourceContent}
          commitRef={commitRef}
        />
      )}
    </Container>
  )
}
