import React from 'react'
import { Container } from '@harness/uicore'
import type { TypesRepository } from 'services/scm'
import { isFile } from 'utils/GitUtils'
import { useGetResourceContent } from 'hooks/useGetResourceContent'
import { FileEditor } from '../FileEditor/FileEditor'
import css from './RepositoryFileEditContent.module.scss'

interface RepositoryFileEditContentProps {
  gitRef?: string
  resourcePath?: string
  repoMetadata: TypesRepository
}

export function RepositoryFileEditContent({ repoMetadata, gitRef, resourcePath }: RepositoryFileEditContentProps) {
  const { data /*error, loading, refetch, response */ } = useGetResourceContent({ repoMetadata, gitRef, resourcePath })

  // TODO: Handle loading, error, refetch...

  return (
    <Container className={css.resourceContent}>
      {data && isFile(data) && (
        <FileEditor
          repoMetadata={repoMetadata}
          gitRef={gitRef || repoMetadata.defaultBranch}
          resourcePath={resourcePath}
          contentInfo={data}
        />
      )}
    </Container>
  )
}
