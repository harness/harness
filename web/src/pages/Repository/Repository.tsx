import React from 'react'
import { Container, PageBody } from '@harness/uicore'
import { useGetResourceContent } from 'hooks/useGetResourceContent'
import { voidFn, getErrorMessage } from 'utils/Utils'
import { useAppContext } from 'AppContext'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { useStrings } from 'framework/strings'
import { MarkdownViewer } from 'components/SourceCodeViewer/SourceCodeViewer'
import type { GitInfoProps } from 'utils/GitUtils'
import { useDisableCodeMainLinks } from 'hooks/useDisableCodeMainLinks'
import { RepositoryContent } from './RepositoryContent/RepositoryContent'
import { RepositoryHeader } from './RepositoryHeader/RepositoryHeader'
import css from './Repository.module.scss'

export default function Repository() {
  const { gitRef, resourcePath, repoMetadata, error, loading, refetch } = useGetRepositoryMetadata()
  const {
    data: resourceContent,
    error: resourceError,
    loading: resourceLoading,
    isRepositoryEmpty
  } = useGetResourceContent({ repoMetadata, gitRef, resourcePath, includeCommit: true })

  return (
    <Container className={css.main}>
      <PageBody
        loading={loading || resourceLoading}
        error={getErrorMessage(error || resourceError)}
        retryOnError={voidFn(refetch)}>
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

            {isRepositoryEmpty && <EmptyRepositoryInfo repoMetadata={repoMetadata} />}
          </>
        )}
      </PageBody>
    </Container>
  )
}

const EmptyRepositoryInfo: React.FC<Pick<GitInfoProps, 'repoMetadata'>> = ({ repoMetadata }) => {
  const { routes } = useAppContext()
  const { getString } = useStrings()
  const { currentUserProfileURL } = useAppContext()
  const newFileURL = routes.toCODEFileEdit({
    repoPath: repoMetadata.path as string,
    gitRef: repoMetadata.default_branch as string,
    resourcePath: ''
  })

  useDisableCodeMainLinks(true)

  return (
    <Container className={css.repoEmpty}>
      <Container className={css.layout}>
        <MarkdownViewer
          source={getString('repoEmptyMarkdown')
            .replace(/NEW_FILE_URL/g, newFileURL)
            .replace(/REPO_URL/g, repoMetadata.git_url || '')
            .replace(/REPO_NAME/g, repoMetadata.uid || '')
            .replace(/CREATE_API_TOKEN_URL/g, currentUserProfileURL || '')}
        />
      </Container>
    </Container>
  )
}
