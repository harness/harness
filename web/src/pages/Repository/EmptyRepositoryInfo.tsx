import React, { useState } from 'react'
import { Link, useHistory } from 'react-router-dom'
import {
  Button,
  ButtonVariation,
  Container,
  FlexExpander,
  FontVariation,
  Layout,
  StringSubstitute,
  Text
} from '@harness/uicore'
import { permissionProps } from 'utils/Utils'
import { useAppContext } from 'AppContext'
import { useStrings } from 'framework/strings'
import { MarkdownViewer } from 'components/MarkdownViewer/MarkdownViewer'
import { CodeIcon, GitInfoProps } from 'utils/GitUtils'
import { useDisableCodeMainLinks } from 'hooks/useDisableCodeMainLinks'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { CopyButton } from 'components/CopyButton/CopyButton'
import CloneCredentialDialog from 'components/CloneCredentialDialog/CloneCredentialDialog'
import css from './EmptyRepositoryInfo.module.scss'

export const EmptyRepositoryInfo: React.FC<Pick<GitInfoProps, 'repoMetadata'>> = ({ repoMetadata }) => {
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
        resourceType: 'CODE_REPOSITORY'
      },
      permissions: ['code_repo_push']
    },
    [space]
  )
  useDisableCodeMainLinks(true)
  return (
    <Container className={css.emptyRepo}>
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
                    history.push(standalone ? routes.toCODEUserProfile() : currentUserProfileURL)
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
          source={getString('repoEmptyMarkdownClonePush')
            .replace(/REPO_NAME/g, repoMetadata.uid || '')
            .replace(/DEFAULT_BRANCH/g, repoMetadata.default_branch || '')}
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
            .replace(/CREATE_API_TOKEN_URL/g, standalone ? routes.toCODEUserProfile() : currentUserProfileURL || '')
            .replace(/DEFAULT_BRANCH/g, repoMetadata.default_branch || '')}
        />
      </Container>
      <CloneCredentialDialog flag={flag} setFlag={setFlag} />
    </Container>
  )
}
