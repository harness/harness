import React, { ChangeEvent, useCallback, useEffect, useMemo, useRef, useState } from 'react'
import {
  Button,
  ButtonVariation,
  Color,
  Container,
  FlexExpander,
  Icon,
  Layout,
  Text,
  TextInput,
  VisualYamlSelectedView,
  VisualYamlToggle
} from '@harness/uicore'
import { Link, useHistory } from 'react-router-dom'
import ReactJoin from 'react-join'
import cx from 'classnames'
import { SourceCodeEditor } from 'components/SourceCodeEditor/SourceCodeEditor'
import type { RepoFileContent } from 'services/code'
import { useAppContext } from 'AppContext'
import { decodeGitContent, GitCommitAction, GitContentType, GitInfoProps, isDir, makeDiffRefs } from 'utils/GitUtils'
import { useStrings } from 'framework/strings'
import { filenameToLanguage, FILE_SEPERATOR } from 'utils/Utils'
import { useGetResourceContent } from 'hooks/useGetResourceContent'
import { CommitModalButton } from 'components/CommitModalButton/CommitModalButton'
import { DiffEditor } from 'components/SourceCodeEditor/MonacoSourceCodeEditor'
import { NavigationCheck } from 'components/NavigationCheck/NavigationCheck'
import css from './FileEditor.module.scss'

interface EditorProps extends Pick<GitInfoProps, 'repoMetadata' | 'gitRef' | 'resourcePath'> {
  resourceContent: GitInfoProps['resourceContent'] | null
  isRepositoryEmpty: boolean
}

function Editor({ resourceContent, repoMetadata, gitRef, resourcePath, isRepositoryEmpty }: EditorProps) {
  const history = useHistory()
  const name = new URLSearchParams(window.location.href.split('?')?.[1]).get('name')
  const inputRef = useRef<HTMLInputElement | null>()
  const isNew = useMemo(() => !resourceContent || isDir(resourceContent), [resourceContent])
  const [fileName, setFileName] = useState(isNew ? '' : resourceContent?.name || '')
  const [parentPath, setParentPath] = useState(
    isNew ? resourcePath : resourcePath.split(FILE_SEPERATOR).slice(0, -1).join(FILE_SEPERATOR)
  )
  const { getString } = useStrings()
  const { routes } = useAppContext()
  const [language, setLanguage] = useState(() => filenameToLanguage(fileName))
  const [originalContent, setOriginalContent] = useState(
    decodeGitContent((resourceContent?.content as RepoFileContent)?.data)
  )
  const [content, setContent] = useState(originalContent)
  const fileResourcePath = useMemo(
    () => [(parentPath || '').trim(), (fileName || '').trim()].filter(p => !!p.trim()).join(FILE_SEPERATOR),
    [parentPath, fileName]
  )
  const { data: folderContent, refetch: verifyFolder } = useGetResourceContent({
    repoMetadata,
    gitRef,
    resourcePath: fileResourcePath,
    includeCommit: false,
    lazy: true
  })
  const isUpdate = useMemo(() => resourcePath === fileResourcePath, [resourcePath, fileResourcePath])
  const commitAction = useMemo(
    () => (isNew ? GitCommitAction.CREATE : isUpdate ? GitCommitAction.UPDATE : GitCommitAction.MOVE),
    [isNew, isUpdate]
  )
  const [startVerifyFolder, setStartVerifyFolder] = useState(false)
  const rebuildPaths = useCallback(() => {
    const _tokens = fileName.split(FILE_SEPERATOR).filter(part => !!part.trim())
    const _fileName = ((_tokens.pop() as string) || '').trim()
    const _parentPath = parentPath
      .split(FILE_SEPERATOR)
      .concat(_tokens)
      .map(p => p.trim())
      .filter(part => !!part.trim())
      .join(FILE_SEPERATOR)

    if (_fileName) {
      const normalizedFilename = _fileName.trim()
      const newLanguage = filenameToLanguage(normalizedFilename)

      if (normalizedFilename !== fileName) {
        setFileName(normalizedFilename)
      }

      // A workaround to force Monaco update content
      // with new language. Monaco still throws an error
      // textModel.js:178 Uncaught Error: Model is disposed!
      if (language !== newLanguage) {
        setLanguage(newLanguage)
        setOriginalContent(content)
      }
    }

    setParentPath(_parentPath)

    // Make API call to verify if fileResourcePath is an existing folder
    verifyFolder().then(() => setStartVerifyFolder(true))
  }, [fileName, parentPath, language, content, verifyFolder])
  const [selectedView, setSelectedView] = useState(VisualYamlSelectedView.VISUAL)
  const disabled = useMemo(
    () => !fileName || (isUpdate && content === originalContent),
    [fileName, isUpdate, content, originalContent]
  )

  // Calculate file name input field width based on number of characters inside
  useEffect(() => {
    if (inputRef.current) {
      inputRef.current.size = Math.min(Math.max(fileName.length - 2, 20), 50)
    }
  }, [fileName, inputRef])

  // When file name is modified, verify if fileResourcePath is a folder. If it is
  // then rebuild parentPath and fileName (becomes empty)
  useEffect(() => {
    if (startVerifyFolder && folderContent?.type === GitContentType.DIR) {
      setStartVerifyFolder(false)
      setParentPath(fileResourcePath)
      setFileName('')
    }
  }, [startVerifyFolder, folderContent, fileResourcePath])

  useEffect(() => {
    if (isNew && !!name) {
      // setName from click on empty repo page so either readme, license or gitignore
      const nameExists = name
      if (nameExists !== '') {
        const newFilename = name
        setFileName(newFilename)
      }
    }
  }, [isNew, name])

  return (
    <Container className={css.container}>
      <Layout.Horizontal className={css.heading}>
        <Container>
          <Layout.Horizontal spacing="small" className={css.path}>
            <Link to={routes.toCODERepository({ repoPath: repoMetadata.path as string, gitRef })}>
              <Icon name="code-folder" padding={{ right: 'xsmall' }} />
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
                        to={routes.toCODERepository({
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
              autoFocus={isNew}
              value={fileName}
              inputRef={_ref => (inputRef.current = _ref)}
              wrapperClassName={css.inputContainer}
              placeholder={getString('nameYourFile')}
              onInput={(event: ChangeEvent<HTMLInputElement>) => {
                setFileName(event.currentTarget.value)
              }}
              onBlur={rebuildPaths}
              onFocus={({ target }) => {
                const value = (parentPath ? parentPath + FILE_SEPERATOR : '') + fileName
                setFileName(value)
                setParentPath('')
                setTimeout(() => {
                  target.setSelectionRange(value.length, value.length)
                  target.scrollLeft = Number.MAX_SAFE_INTEGER
                }, 0)
              }}
            />
            <Text color={Color.GREY_900}>{getString('in')}</Text>
            <Link
              to={routes.toCODERepository({
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
              disabled={disabled}
              repoMetadata={repoMetadata}
              commitAction={commitAction}
              commitTitlePlaceHolder={getString(isNew ? 'createFile' : isUpdate ? 'updateFile' : 'renameFile')
                .replace('__path__', isUpdate || isNew ? fileResourcePath : resourcePath)
                .replace('__newPath__', fileResourcePath)}
              gitRef={gitRef}
              oldResourcePath={commitAction === GitCommitAction.MOVE ? resourcePath : undefined}
              resourcePath={fileResourcePath}
              payload={content}
              sha={resourceContent?.sha}
              onSuccess={(_data, newBranch) => {
                if (newBranch) {
                  history.replace(
                    routes.toCODECompare({
                      repoPath: repoMetadata.path as string,
                      diffRefs: makeDiffRefs(repoMetadata?.default_branch as string, newBranch)
                    })
                  )
                } else {
                  history.push(
                    routes.toCODERepository({
                      repoPath: repoMetadata.path as string,
                      resourcePath: fileResourcePath,
                      gitRef
                    })
                  )
                }
                setOriginalContent(content)
              }}
              disableBranchCreation={isRepositoryEmpty}
            />
            <Button
              text={getString('cancel')}
              variation={ButtonVariation.TERTIARY}
              onClick={() => {
                history.replace(
                  routes.toCODERepository({
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

      <Container className={css.tabs}>
        <VisualYamlToggle
          onChange={setSelectedView}
          selectedView={selectedView}
          labels={{ visual: getString('contents'), yaml: getString('changes') }}
          className={css.selectedView}
        />
      </Container>

      <Container className={cx(css.editorContainer, language)}>
        {selectedView === VisualYamlSelectedView.VISUAL ? (
          <SourceCodeEditor language={language} source={content} onChange={setContent} />
        ) : (
          <DiffEditor language={language} original={originalContent} source={content} onChange={setContent} />
        )}
      </Container>
      <NavigationCheck when={!disabled} />
    </Container>
  )
}

const PathSeparator = () => <Text color={Color.GREY_900}>/</Text>

export const FileEditor = React.memo(Editor)
