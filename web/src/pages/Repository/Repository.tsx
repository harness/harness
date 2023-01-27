import React from 'react'
import { Button, ButtonVariation, Container, FontVariation, Layout, PageBody, Text } from '@harness/uicore'
import { useGetResourceContent } from 'hooks/useGetResourceContent'
import { voidFn, getErrorMessage } from 'utils/Utils'
import { useAppContext } from 'AppContext'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { LoadingSpinner } from 'components/LoadingSpinner/LoadingSpinner'
import { useStrings } from 'framework/strings'
import type { OpenapiGetContentOutput } from 'services/code'
import { MarkdownViewer } from 'components/SourceCodeViewer/SourceCodeViewer'
import type { GitInfoProps } from 'utils/GitUtils'
import { useDisableCodeMainLinks } from 'hooks/useDisableCodeMainLinks'
import { RepositoryContent } from './RepositoryContent/RepositoryContent'
import { RepositoryHeader } from './RepositoryHeader/RepositoryHeader'
import { ContentHeader } from './RepositoryContent/ContentHeader/ContentHeader'
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
      <PageBody error={getErrorMessage(error || resourceError)} retryOnError={voidFn(refetch)}>
        <LoadingSpinner visible={loading || resourceLoading} withBorder={!!resourceContent && resourceLoading} />

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

            {isRepositoryEmpty && (
              <EmptyRepositoryInfo
                repoMetadata={repoMetadata}
                resourceContent={resourceContent as OpenapiGetContentOutput}
              />
            )}
          </>
        )}
      </PageBody>
    </Container>
  )
}

const EmptyRepositoryInfo: React.FC<Pick<GitInfoProps, 'repoMetadata' | 'resourceContent'>> = (
  { repoMetadata },
  resourceContent
) => {
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
    <Container className={css.emptyRepo}>
      <ContentHeader
        repoMetadata={repoMetadata}
        gitRef={repoMetadata.default_branch as string}
        resourcePath={''}
        resourceContent={resourceContent}
      />
      <Container
        margin={{ bottom: 'xxlarge' }}
        padding={{ top: 'xxlarge', bottom: 'xxlarge', left: 'medium', right: 'medium' }}
        className={css.divContainer}>
        <Text font={{ variation: FontVariation.H5 }}>{getString('emptyRepoHeader')}</Text>
        <Layout.Horizontal padding={{ top: 'large' }}>
          <Button variation={ButtonVariation.PRIMARY} text={getString('addNewFile')} href={newFileURL}></Button>

          <Container padding={{ left: 'medium', top: 'xsmall' }}>
            <MarkdownViewer
              source={getString('emptyRepoInclude')
                .replace(/README_URL/g, newFileURL + `?name=README.md` || '')
                .replace(/LICENSE_URL/g, newFileURL + `?name=LICENSE.md` || '')
                .replace(/GITIGNORE_URL/g, newFileURL + `?name=.gitignore` || '')}
            />
          </Container>
        </Layout.Horizontal>
      </Container>
      <Container
        margin={{ bottom: 'xxlarge' }}
        padding={{ top: 'xxlarge', bottom: 'xxlarge', left: 'medium', right: 'medium' }}
        className={css.divContainer}>
        <MarkdownViewer
          source={getString('repoEmptyMarkdownClone')
            .replace(/REPO_URL/g, repoMetadata.git_url || '')
            .replace(/REPO_NAME/g, repoMetadata.uid || '')}
        />
      </Container>
      <Container
        margin={{ bottom: 'xxlarge' }}
        padding={{ top: 'xxlarge', bottom: 'xxlarge', left: 'medium', right: 'medium' }}
        className={css.divContainer}>
        <MarkdownViewer
          source={getString('repoEmptyMarkdownExisting')
            .replace(/REPO_URL/g, '...')
            .replace(/REPO_NAME/g, repoMetadata.uid || '')
            .replace(/CREATE_API_TOKEN_URL/g, currentUserProfileURL || '')}
        />
      </Container>
    </Container>
  )
}
