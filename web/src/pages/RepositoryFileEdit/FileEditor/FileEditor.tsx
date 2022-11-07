import React from 'react'
import {
  Button,
  ButtonSize,
  ButtonVariation,
  Color,
  Container,
  FlexExpander,
  Icon,
  Layout,
  Text
} from '@harness/uicore'
import { Link } from 'react-router-dom'
import ReactJoin from 'react-join'
import { SourceCodeEditor } from 'components/SourceCodeEditor/SourceCodeEditor'
import type { OpenapiGetContentOutput, RepoFileContent, TypesRepository } from 'services/scm'
import { useAppContext } from 'AppContext'
import { GitIcon } from 'utils/GitUtils'
import { useStrings } from 'framework/strings'
import { filenameToLanguage } from 'utils/Utils'
import css from './FileEditor.module.scss'

interface FileEditorProps {
  repoMetadata: TypesRepository
  gitRef?: string
  resourcePath?: string
  contentInfo: OpenapiGetContentOutput
}

export function FileEditor({ contentInfo, repoMetadata, gitRef, resourcePath = '' }: FileEditorProps) {
  const { getString } = useStrings()
  const { routes } = useAppContext()

  return (
    <Container className={css.container}>
      <Layout.Horizontal className={css.heading}>
        <Container>
          <Layout.Horizontal spacing="small">
            <Link to={routes.toSCMRepository({ repoPath: repoMetadata.path as string, gitRef })}>
              <Icon name="main-folder" />
            </Link>
            <Text color={Color.GREY_900}>/</Text>
            <ReactJoin separator={<Text color={Color.GREY_900}>/</Text>}>
              {resourcePath.split('/').map((_path, index, paths) => {
                const pathAtIndex = paths.slice(0, index + 1).join('/')

                return (
                  <Link
                    key={_path + index}
                    to={routes.toSCMRepository({
                      repoPath: repoMetadata.path as string,
                      gitRef,
                      resourcePath: pathAtIndex
                    })}>
                    <Text color={Color.GREY_900}>{_path}</Text>
                  </Link>
                )
              })}
            </ReactJoin>
          </Layout.Horizontal>
        </Container>
        <FlexExpander />
        <Button
          text={getString('commit')}
          icon={GitIcon.COMMIT}
          iconProps={{ size: 10 }}
          variation={ButtonVariation.PRIMARY}
          size={ButtonSize.SMALL}
        />
      </Layout.Horizontal>

      {(contentInfo?.content as RepoFileContent)?.data && (
        <Container className={css.content}>
          <SourceCodeEditor
            className={css.editorContainer}
            height="100%"
            language={filenameToLanguage(contentInfo?.name)}
            source={window.atob((contentInfo?.content as RepoFileContent)?.data || '')}
          />
        </Container>
      )}
    </Container>
  )
}
