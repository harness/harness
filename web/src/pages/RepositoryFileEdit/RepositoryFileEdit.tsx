import React from 'react'
import { Container, PageBody } from '@harness/uicore'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { useGetResourceContent } from 'hooks/useGetResourceContent'
import { useDisableCodeMainLinks } from 'hooks/useDisableCodeMainLinks'
import { voidFn, getErrorMessage } from 'utils/Utils'
import { RepositoryFileEditHeader } from './RepositoryFileEditHeader/RepositoryFileEditHeader'
import { FileEditor } from './FileEditor/FileEditor'
import css from './RepositoryFileEdit.module.scss'

export default function RepositoryFileEdit() {
  const { gitRef, resourcePath, repoMetadata, error, loading, refetch } = useGetRepositoryMetadata()
  const {
    data: resourceContent,
    error: resourceError,
    loading: resourceLoading,
    isRepositoryEmpty
  } = useGetResourceContent({ repoMetadata, gitRef, resourcePath })

  useDisableCodeMainLinks(!!isRepositoryEmpty)

  return (
    <Container className={css.main}>
      <PageBody
        loading={loading || resourceLoading}
        error={getErrorMessage(error || resourceError)}
        retryOnError={voidFn(refetch)}>
        {repoMetadata ? (
          <>
            <RepositoryFileEditHeader repoMetadata={repoMetadata} resourceContent={resourceContent} />
            <Container className={css.resourceContent}>
              {(resourceContent || isRepositoryEmpty) && (
                <FileEditor
                  repoMetadata={repoMetadata}
                  gitRef={gitRef}
                  resourcePath={resourcePath}
                  resourceContent={resourceContent}
                  isRepositoryEmpty={isRepositoryEmpty}
                />
              )}
            </Container>
          </>
        ) : null}
      </PageBody>
    </Container>
  )
}
