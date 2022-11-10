import React, { ChangeEvent, useCallback, useMemo, useState } from 'react'
import { Button, ButtonVariation, Color, Container, FlexExpander, Icon, Layout, Text, TextInput } from '@harness/uicore'
import { Link, useHistory } from 'react-router-dom'
import ReactJoin from 'react-join'
import cx from 'classnames'
import { SourceCodeEditor } from 'components/SourceCodeEditor/SourceCodeEditor'
import type { OpenapiGetContentOutput, RepoFileContent, TypesRepository } from 'services/scm'
import { useAppContext } from 'AppContext'
import { isDir } from 'utils/GitUtils'
import { useStrings } from 'framework/strings'
import { filenameToLanguage, FILE_SEPERATOR } from 'utils/Utils'
import { CommitModalButton } from 'components/CommitModalButton/CommitModalButton'
import css from './FileEditor.module.scss'

interface FileEditorProps {
  repoMetadata: TypesRepository
  gitRef: string
  resourcePath: string
  contentInfo: OpenapiGetContentOutput
}

const PathSeparator = () => <Text color={Color.GREY_900}>/</Text>

function Editor({ contentInfo, repoMetadata, gitRef, resourcePath }: FileEditorProps) {
  const history = useHistory()
  const isNew = useMemo(() => isDir(contentInfo), [contentInfo])
  const [fileName, setFileName] = useState(isNew ? '' : (contentInfo.name as string))
  const [parentPath, setParentPath] = useState(
    isNew ? resourcePath : resourcePath.split(FILE_SEPERATOR).slice(0, -1).join(FILE_SEPERATOR)
  )
  const { getString } = useStrings()
  const { routes } = useAppContext()
  const language = filenameToLanguage(contentInfo?.name)
  const rebuildPaths = useCallback(() => {
    const _tokens = fileName.split(FILE_SEPERATOR).filter(part => !!part.trim())
    const _fileName = _tokens.pop() as string
    const _parentPath = parentPath
      .split(FILE_SEPERATOR)
      .concat(_tokens)
      .map(p => p.trim())
      .filter(part => !!part.trim())
      .join(FILE_SEPERATOR)

    if (_fileName && _fileName !== fileName) {
      setFileName(_fileName.trim())
    }

    setParentPath(_parentPath)
  }, [fileName, setFileName, parentPath, setParentPath])
  const fileResourcePath = useMemo(
    () => [parentPath, fileName].filter(p => !!p.trim()).join(FILE_SEPERATOR),
    [parentPath, fileName]
  )

  return (
    <Container className={css.container}>
      <Layout.Horizontal className={css.heading}>
        <Container>
          <Layout.Horizontal spacing="small" className={css.path}>
            <Link to={routes.toSCMRepository({ repoPath: repoMetadata.path as string, gitRef })}>
              <Icon name="main-folder" padding={{ right: 'small' }} />
              <Text color={Color.GREY_900} inline>
                {repoMetadata.uid}
              </Text>
            </Link>
            <PathSeparator />
            {parentPath && (
              <>
                <ReactJoin separator={<PathSeparator />}>
                  {parentPath.split(FILE_SEPERATOR).map((_path, index, paths) => {
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
                <PathSeparator />
              </>
            )}
            <TextInput
              autoFocus
              value={fileName}
              wrapperClassName={css.inputContainer}
              placeholder={getString('nameYourFile')}
              onInput={(event: ChangeEvent<HTMLInputElement>) => {
                setFileName(event.currentTarget.value)
              }}
              onBlur={rebuildPaths}
              onFocus={event => {
                const value = (parentPath ? parentPath + FILE_SEPERATOR : '') + fileName
                setFileName(value)
                setParentPath('')
                setTimeout(() => event.target.setSelectionRange(value.length, value.length), 0)
              }}
            />
            <Text color={Color.GREY_900}>{getString('in')}</Text>
            <Link
              to={routes.toSCMRepository({
                repoPath: repoMetadata.path as string,
                gitRef
              })}
              className={css.refLink}>
              {gitRef}
            </Link>
          </Layout.Horizontal>
        </Container>
        <FlexExpander />
        <Container>
          <Layout.Horizontal spacing="small">
            <CommitModalButton
              text={getString('commitChanges')}
              variation={ButtonVariation.PRIMARY}
              commitMessagePlaceHolder={'Update file...'}
              gitRef={gitRef}
              resourcePath={fileResourcePath}
              onSubmit={data => console.log({ data })}
            />
            <Button
              text={getString('cancelChanges')}
              variation={ButtonVariation.TERTIARY}
              onClick={() => {
                history.push(
                  routes.toSCMRepository({
                    repoPath: repoMetadata.path as string,
                    gitRef,
                    resourcePath
                  })
                )
              }}
            />
          </Layout.Horizontal>
        </Container>
      </Layout.Horizontal>

      <Container className={cx(css.content, language)}>
        <SourceCodeEditor
          className={css.editorContainer}
          height="100%"
          language={language}
          source={window.atob((contentInfo?.content as RepoFileContent)?.data || '')}
        />
      </Container>
    </Container>
  )
}

export const FileEditor = React.memo(Editor)
