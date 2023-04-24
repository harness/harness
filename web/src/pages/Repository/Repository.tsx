import React, { useEffect, useState } from 'react'
import { Link, useHistory } from 'react-router-dom'
import {
  Button,
  ButtonVariation,
  Container,
  FlexExpander,
  FontVariation,
  Layout,
  PageBody,
  StringSubstitute,
  Text
} from '@harness/uicore'
import { Falsy, Match, Truthy } from 'react-jsx-match'
import { useGetResourceContent } from 'hooks/useGetResourceContent'
import { voidFn, getErrorMessage, permissionProps } from 'utils/Utils'
import { useAppContext } from 'AppContext'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { LoadingSpinner } from 'components/LoadingSpinner/LoadingSpinner'
import { useStrings } from 'framework/strings'
import type { OpenapiGetContentOutput, TypesRepository } from 'services/code'
import { MarkdownViewer } from 'components/MarkdownViewer/MarkdownViewer'
import { CodeIcon, GitInfoProps } from 'utils/GitUtils'
import { useDisableCodeMainLinks } from 'hooks/useDisableCodeMainLinks'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { CopyButton } from 'components/CopyButton/CopyButton'
import CloneCredentialDialog from 'components/CloneCredentialDialog/CloneCredentialDialog'
import { Images } from 'images'
import { CopyButton } from 'components/CopyButton/CopyButton'
import CloneCredentialDialog from 'components/CloneCredentialDialog/CloneCredentialDialog'
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
  const [fileNotExist, setFileNotExist] = useState(false)
  const { getString } = useStrings()

  useEffect(() => {
    if (resourceError?.status === 404) {
      setFileNotExist(true)
    } else {
      setFileNotExist(false)
    }
  }, [resourceError])
  return (
    <Container className={css.main}>
      <Match expr={fileNotExist}>
        <Truthy>
          <RepositoryHeader repoMetadata={repoMetadata as TypesRepository} />
          <Layout.Vertical>
            <Container className={css.bannerContainer} padding={{ left: 'xlarge' }}>
              <Text font={'small'} padding={{ left: 'large' }}>
                <StringSubstitute
                  str={getString('branchDoesNotHaveFile')}
                  vars={{
                    repoName: repoMetadata?.uid,
                    fileName: resourcePath,
                    branchName: gitRef
                  }}
                />
              </Text>
            </Container>
            <Container padding={{ left: 'xlarge' }}>
              <ContentHeader
                repoMetadata={repoMetadata as TypesRepository}
                gitRef={gitRef}
                resourcePath={resourcePath}
                resourceContent={resourceContent as OpenapiGetContentOutput}
              />
            </Container>
            <PageBody
              noData={{
                when: () => fileNotExist === true,
                message: getString('error404Text'),
                image: Images.error404
              }}></PageBody>
          </Layout.Vertical>
        </Truthy>
        <Falsy>
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
        </Falsy>
      </Match>
    </Container>
  )
}

const EmptyRepositoryInfo: React.FC<Pick<GitInfoProps, 'repoMetadata' | 'resourceContent'>> = (
  { repoMetadata },
  resourceContent
) => {
  const history = useHistory()
  const { routes } = useAppContext()
  const { getString } = useStrings()
  const { currentUserProfileURL } = useAppContext()
  const newFileURL = routes.toCODEFileEdit({
    repoPath: repoMetadata.path as string,
    gitRef: repoMetadata.default_branch as string,
    resourcePath: ''
  })
  const { standalone } = useAppContext()
  const { hooks } = useAppContext()
  const space = useGetSpaceParam()
  const [flag, setFlag] = useState(false)

  const permPushResult = hooks?.usePermissionTranslate?.(
    {
      resource: {
        resourceType: 'CODE_REPO'
      },
      permissions: ['code_repo_push']
    },
    [space]
  )
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
        padding={{ top: 'xxlarge', bottom: 'xxlarge', left: 'xxlarge', right: 'xxlarge' }}
        className={css.divContainer}>
        <Text font={{ variation: FontVariation.H5 }}>{getString('emptyRepoHeader')}</Text>
        <Layout.Horizontal padding={{ top: 'large' }}>
          <Button
            variation={ButtonVariation.PRIMARY}
            text={getString('addNewFile')}
            onClick={() => history.push(newFileURL)}
            {...permissionProps(permPushResult, standalone)}></Button>

          <Container padding={{ left: 'medium', top: 'small' }}>
            <Text className={css.textContainer}>
              <StringSubstitute
                str={getString('emptyRepoInclude')}
                vars={{
                  README: <Link to={newFileURL + `?name=README.md`}>README</Link>,
                  LICENSE: <Link to={newFileURL + `?name=LICENSE.md`}>LICENSE</Link>,
                  GITIGNORE: <Link to={newFileURL + `?name=.gitignore`}>.gitignore</Link>
                }}
              />
            </Text>
          </Container>
        </Layout.Horizontal>
      </Container>
      <Container
        margin={{ bottom: 'xxlarge' }}
        padding={{ top: 'xxlarge', bottom: 'xxlarge', left: 'xxlarge', right: 'xxlarge' }}
        className={css.divContainer}>
        <Text font={{ variation: FontVariation.H4 }}>{getString('firstTimeTitle')}</Text>
        <Text
          className={css.text}
          padding={{ top: 'medium', bottom: 'small' }}
          font={{ variation: FontVariation.BODY }}>
          {getString('cloneHTTPS')}
        </Text>
        <Layout.Horizontal>
          <Container padding={{ bottom: 'medium' }} width={400} margin={{ right: 'small' }}>
            <Layout.Horizontal className={css.layout}>
              <Text className={css.url}>{repoMetadata.git_url}</Text>
              <FlexExpander />
              <CopyButton
                content={repoMetadata?.git_url as string}
                id={css.cloneCopyButton}
                icon={CodeIcon.Copy}
                iconProps={{ size: 14 }}
              />
            </Layout.Horizontal>
          </Container>
          {standalone ? null : (
            <Button
              onClick={() => {
                setFlag(true)
              }}
              variation={ButtonVariation.SECONDARY}>
              {getString('generateCloneCred')}
            </Button>
          )}
        </Layout.Horizontal>
        <Text font={{ variation: FontVariation.BODY, size: 'small' }}>
          <StringSubstitute
            str={getString('manageCredText')}
            vars={{
              URL: (
                <a
                  onClick={() => {
                    history.push(currentUserProfileURL)
                  }}>
                  here
                </a>
              )
            }}
          />
        </Text>
      </Container>
      <Container
        margin={{ bottom: 'xxlarge' }}
        padding={{ top: 'xxlarge', bottom: 'xxlarge', left: 'xxlarge', right: 'xxlarge' }}
        className={css.divContainer}>
        <MarkdownViewer
          source={getString('repoEmptyMarkdownClonePush').replace(/REPO_NAME/g, repoMetadata.uid || '')}
        />
      </Container>
      <Container
        margin={{ bottom: 'xxlarge' }}
        padding={{ top: 'xxlarge', bottom: 'xxlarge', left: 'xxlarge', right: 'xxlarge' }}
        className={css.divContainer}>
        <MarkdownViewer
          source={getString('repoEmptyMarkdownExisting')
            .replace(/REPO_URL/g, repoMetadata.git_url || '')
            .replace(/REPO_NAME/g, repoMetadata.uid || '')
            .replace(/CREATE_API_TOKEN_URL/g, currentUserProfileURL || '')}
        />
      </Container>
      <CloneCredentialDialog flag={flag} setFlag={setFlag} />
    </Container>
  )
}
