/*
 * Copyright 2023 Harness, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import React, { useEffect, useMemo, useRef, useState } from 'react'
import { useGet } from 'restful-react'
import {
  ButtonSize,
  ButtonVariation,
  Container,
  FlexExpander,
  Heading,
  Layout,
  StringSubstitute,
  Tabs,
  Utils
} from '@harnessio/uicore'
import { Icon } from '@harnessio/icons'
import { Color } from '@harnessio/design-system'
import { Document, Page, pdfjs } from 'react-pdf'
import { Render, Match, Truthy, Falsy, Case, Else } from 'react-jsx-match'
import { Link, useHistory } from 'react-router-dom'
import type { EditorDidMount } from 'react-monaco-editor'
import type { editor } from 'monaco-editor'
import { SourceCodeViewer } from 'components/SourceCodeViewer/SourceCodeViewer'
import type { OpenapiContentInfo, RepoFileContent, TypesCommit } from 'services/code'
import {
  normalizeGitRef,
  decodeGitContent,
  findMarkdownInfo,
  GitCommitAction,
  GitInfoProps,
  isGitRev,
  isRefATag,
  makeDiffRefs
} from 'utils/GitUtils'
import {
  filenameToLanguage,
  permissionProps,
  LIST_FETCHING_LIMIT,
  RenameDetails,
  FileSection,
  PAGE_CONTAINER_WIDTH
} from 'utils/Utils'
import { useAppContext } from 'AppContext'
import { LatestCommitForFile } from 'components/LatestCommit/LatestCommit'
import { useCommitModal } from 'components/CommitModalButton/CommitModalButton'
import { useStrings } from 'framework/strings'
import { getConfig } from 'services/config'
import { OptionsMenuButton } from 'components/OptionsMenuButton/OptionsMenuButton'
import { PlainButton } from 'components/PlainButton/PlainButton'
import { CommitsView } from 'components/CommitsView/CommitsView'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { useQueryParams } from 'hooks/useQueryParams'
import { FileCategory, RepoContentExtended, useFileContentViewerDecision } from 'utils/FileUtils'
import { useDownloadRawFile } from 'hooks/useDownloadRawFile'
import { usePageIndex } from 'hooks/usePageIndex'
import { Readme } from '../FolderContent/Readme'
import { GitBlame } from './GitBlame'
import RenameContentHistory from './RenameContentHistory'
import css from './FileContent.module.scss'

export function FileContent({
  repoMetadata,
  gitRef,
  resourcePath,
  resourceContent,
  commitRef
}: Pick<GitInfoProps, 'repoMetadata' | 'gitRef' | 'resourcePath' | 'resourceContent' | 'commitRef'>) {
  const { routes } = useAppContext()
  const { getString } = useStrings()
  const downloadFile = useDownloadRawFile()
  const { category, isText, isFileTooLarge, isViewable, filename, extension, size, base64Data, rawURL } =
    useFileContentViewerDecision({ repoMetadata, gitRef, resourcePath, resourceContent })
  const history = useHistory()
  const [activeTab, setActiveTab] = React.useState<string>(FileSection.CONTENT)
  const content = useMemo(
    () => decodeGitContent((resourceContent?.content as RepoFileContent)?.data),
    [resourceContent?.content]
  )
  const markdownInfo = useMemo(() => findMarkdownInfo(resourceContent), [resourceContent])
  const [openDeleteFileModal] = useCommitModal({
    repoMetadata,
    gitRef,
    resourcePath,
    commitAction: GitCommitAction.DELETE,
    commitTitlePlaceHolder: getString('deleteFile').replace('__path__', resourcePath),
    onSuccess: (_commitInfo, newBranch) => {
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
            gitRef
          })
        )
      }
    }
  })

  const { standalone } = useAppContext()
  const { hooks } = useAppContext()
  const space = useGetSpaceParam()
  const permPushResult = hooks?.usePermissionTranslate?.(
    {
      resource: {
        resourceType: 'CODE_REPOSITORY',
        resourceIdentifier: repoMetadata?.identifier as string
      },
      permissions: ['code_repo_push']
    },
    [space]
  )
  const permsFinal = useMemo(() => {
    const perms = permissionProps(permPushResult, standalone)
    if (isRefATag(gitRef) && perms) {
      return { tooltip: perms.tooltip, disabled: true }
    }

    if (isRefATag(gitRef)) {
      return { tooltip: getString('editNotAllowed'), disabled: true }
    } else if (perms?.disabled) {
      return { disabled: perms.disabled, tooltip: perms.tooltip }
    }
    return { disabled: isRefATag(gitRef) || false, tooltip: undefined }
  }, [permPushResult, gitRef]) // eslint-disable-line react-hooks/exhaustive-deps
  const [pdfWidth, setPdfWidth] = useState<number>(700)
  const ref = useRef<HTMLDivElement>(null)
  const [numPages, setNumPages] = useState<number>()

  useEffect(() => {
    if (ref.current) {
      const width = Math.min(Math.max(ref.current.clientWidth - 100, 700), 1800)

      if (pdfWidth !== width) {
        setPdfWidth(width)
      }
    }
  }, [pdfWidth, ref.current?.clientWidth])

  const [page] = usePageIndex()
  const { data: commits } = useGet<{ commits: TypesCommit[]; rename_details: RenameDetails[] }>({
    path: `/api/v1/repos/${repoMetadata?.path}/+/commits`,
    queryParams: {
      limit: LIST_FETCHING_LIMIT,
      page,
      git_ref: normalizeGitRef(commitRef || gitRef || repoMetadata?.default_branch),
      path: resourcePath
    },
    lazy: !repoMetadata
  })
  const editButtonDisabled = useMemo(() => permsFinal.disabled || !isText, [permsFinal.disabled, isText])
  const editAsText = useMemo(
    () => editButtonDisabled && !isFileTooLarge && category === FileCategory.OTHER,
    [editButtonDisabled, isFileTooLarge, category]
  )

  const { keyword } = useQueryParams<{ keyword?: string }>()

  const onEditorMount: EditorDidMount = editor => {
    if (!keyword) {
      return
    }

    const editorModel = editor.getModel() as editor.ITextModel
    const keywordMatches: editor.FindMatch[] = editorModel.findMatches(keyword, false, true, false, null, false)

    if (keywordMatches.length > 0) {
      keywordMatches.forEach((match: editor.FindMatch): void => {
        editorModel.deltaDecorations(
          [],
          [
            {
              range: match.range,
              options: {
                isWholeLine: false,
                inlineClassName: 'highlight'
              }
            }
          ]
        )
      })
    }
  }

  return (
    <Container className={css.tabsContainer} ref={ref}>
      <Tabs
        id="fileTabs"
        selectedTabId={activeTab}
        defaultSelectedTabId={FileSection.CONTENT}
        large={false}
        onChange={(id: string) => setActiveTab(id)}
        tabList={[
          {
            id: FileSection.CONTENT,
            title: getString('content'),
            panel: (
              <Container className={css.fileContent}>
                <Layout.Vertical spacing="small" style={{ maxWidth: '100%' }}>
                  <LatestCommitForFile
                    gitRef={gitRef}
                    repoMetadata={repoMetadata}
                    latestCommit={resourceContent.latest_commit}
                    standaloneStyle
                    size={size}
                  />
                  <Container className={css.container} background={Color.WHITE}>
                    <Layout.Horizontal padding="small" className={css.heading}>
                      <Heading level={5} color={Color.BLACK}>
                        {resourceContent.name}
                      </Heading>
                      <FlexExpander />
                      <Layout.Horizontal spacing="xsmall" style={{ alignItems: 'center' }}>
                        <PlainButton
                          withoutCurrentColor
                          size={ButtonSize.SMALL}
                          variation={ButtonVariation.TERTIARY}
                          iconProps={{ size: 16 }}
                          text={getString(editAsText ? 'editAsText' : 'edit')}
                          icon="code-edit"
                          tooltipProps={{ isDark: true }}
                          tooltip={permsFinal.tooltip}
                          disabled={(editButtonDisabled && !editAsText) || isGitRev(gitRef)}
                          onClick={() => {
                            history.push(
                              routes.toCODEFileEdit({
                                repoPath: repoMetadata.path as string,
                                gitRef,
                                resourcePath
                              })
                            )
                          }}
                        />
                        <OptionsMenuButton
                          isDark={true}
                          icon="Options"
                          iconProps={{ size: 14 }}
                          style={{ padding: '5px' }}
                          width="145px"
                          items={[
                            {
                              hasIcon: true,
                              iconName: 'arrow-right',
                              text: getString('viewRaw'),
                              onClick: () => {
                                const url = standalone
                                  ? rawURL.replace(/^\/code/, '')
                                  : getConfig(rawURL).replace('//', '/')
                                window.open(url, '_blank')
                              }
                            },
                            '-',
                            {
                              hasIcon: true,
                              iconName: 'cloud-download',
                              text: getString('download'),
                              onClick: () => downloadFile({ repoMetadata, resourcePath, gitRef, filename })
                            },
                            {
                              hasIcon: true,
                              iconName: 'code-copy',
                              iconSize: 16,
                              text: getString('copy'),
                              onClick: () => Utils.copy(content)
                            },
                            {
                              hasIcon: true,
                              iconName: 'code-delete',
                              iconSize: 16,
                              title: getString(isRefATag(gitRef) ? 'deleteNotAllowed' : 'delete'),
                              disabled: isRefATag(gitRef),
                              text: getString('delete'),
                              onClick: openDeleteFileModal
                            }
                          ]}
                        />
                      </Layout.Horizontal>
                    </Layout.Horizontal>

                    <Render
                      when={
                        (resourceContent?.content as RepoFileContent)?.data ||
                        (resourceContent?.content as RepoContentExtended)?.target ||
                        (resourceContent?.content as RepoContentExtended)?.url
                      }>
                      <Container className={css.content}>
                        <Match expr={isViewable}>
                          <Falsy>
                            <Center>
                              <Link
                                to={rawURL}
                                onClick={e => {
                                  Utils.stopEvent(e)
                                  downloadFile({
                                    repoMetadata,
                                    resourcePath,
                                    gitRef: normalizeGitRef(gitRef) as string,
                                    filename
                                  })
                                }}>
                                <Layout.Horizontal spacing="small">
                                  <Icon name="cloud-download" size={16} />
                                  <span>{getString('download')}</span>
                                </Layout.Horizontal>
                              </Link>
                            </Center>
                          </Falsy>
                          <Truthy>
                            <Match expr={isFileTooLarge}>
                              <Truthy>
                                <Center>
                                  <Match expr={category}>
                                    <Case val={FileCategory.PDF}>
                                      <Document
                                        file={rawURL}
                                        options={{
                                          // TODO: Configure this to use a local worker/webpack loader
                                          cMapUrl: `https://unpkg.com/pdfjs-dist@${pdfjs.version}/cmaps/`,
                                          cMapPacked: true
                                        }}
                                        onLoadSuccess={({ numPages: nextNumPages }) => setNumPages(nextNumPages)}>
                                        {Array.from(new Array(numPages), (_el, index) => (
                                          <Page
                                            loading=""
                                            width={pdfWidth}
                                            key={`page_${index + 1}`}
                                            pageNumber={index + 1}
                                            renderTextLayer={false}
                                            renderAnnotationLayer={false}
                                          />
                                        ))}
                                      </Document>
                                    </Case>
                                    <Case val={FileCategory.AUDIO}>
                                      <audio controls>
                                        <source src={rawURL} />
                                      </audio>
                                    </Case>
                                    <Case val={FileCategory.VIDEO}>
                                      <video controls height={500}>
                                        <source src={rawURL} />
                                      </video>
                                    </Case>
                                    <Else>
                                      <StringSubstitute
                                        str={getString('fileTooLarge')}
                                        vars={{
                                          download: (
                                            <Link
                                              to={rawURL} // TODO: Link component generates wrong copy link
                                              onClick={e => {
                                                Utils.stopEvent(e)
                                                downloadFile({
                                                  repoMetadata,
                                                  resourcePath,
                                                  gitRef: normalizeGitRef(gitRef) as string,
                                                  filename
                                                })
                                              }}>
                                              <Layout.Horizontal spacing="small" padding={{ left: 'small' }}>
                                                <Icon name="cloud-download" size={16} />
                                                <span>{getString('clickHereToDownload')}</span>
                                              </Layout.Horizontal>
                                            </Link>
                                          )
                                        }}
                                      />
                                    </Else>
                                  </Match>
                                </Center>
                              </Truthy>
                              <Falsy>
                                <Match expr={markdownInfo}>
                                  <Truthy>
                                    <Readme
                                      metadata={repoMetadata}
                                      readmeInfo={markdownInfo as OpenapiContentInfo}
                                      contentOnly
                                      maxWidth={`calc(var(${PAGE_CONTAINER_WIDTH}) - 80px)`}
                                      gitRef={gitRef}
                                    />
                                  </Truthy>
                                  <Falsy>
                                    <Center>
                                      <Match expr={category}>
                                        <Case val={FileCategory.SVG}>
                                          <img
                                            src={`data:image/svg+xml;base64,${base64Data}`}
                                            alt={filename}
                                            style={{ maxWidth: '100%', maxHeight: '100%' }}
                                          />
                                        </Case>
                                        <Case val={FileCategory.IMAGE}>
                                          <img
                                            src={`data:image/${extension};base64,${base64Data}`}
                                            alt={filename}
                                            style={{ maxWidth: '100%', maxHeight: '100%' }}
                                          />
                                        </Case>
                                        <Case val={FileCategory.PDF}>
                                          <Document
                                            file={`data:application/pdf;base64,${base64Data}`}
                                            options={{
                                              // TODO: Configure this to use a local worker/webpack loader
                                              cMapUrl: `https://unpkg.com/pdfjs-dist@${pdfjs.version}/cmaps/`,
                                              cMapPacked: true
                                            }}
                                            onLoadSuccess={({ numPages: nextNumPages }) => setNumPages(nextNumPages)}>
                                            {Array.from(new Array(numPages), (_el, index) => (
                                              <Page
                                                loading=""
                                                width={pdfWidth}
                                                key={`page_${index + 1}`}
                                                pageNumber={index + 1}
                                                renderTextLayer={false}
                                                renderAnnotationLayer={false}
                                              />
                                            ))}
                                          </Document>
                                        </Case>
                                        <Case val={FileCategory.AUDIO}>
                                          <audio controls>
                                            <source src={`data:audio/${extension};base64,${base64Data}`} />
                                          </audio>
                                        </Case>
                                        <Case val={FileCategory.VIDEO}>
                                          <video controls height={500}>
                                            <source src={`data:video/${extension};base64,${base64Data}`} />
                                          </video>
                                        </Case>
                                        <Case val={FileCategory.TEXT}>
                                          <SourceCodeViewer
                                            editorDidMount={onEditorMount}
                                            language={filenameToLanguage(filename)}
                                            source={decodeGitContent(base64Data)}
                                          />
                                        </Case>
                                        <Case val={FileCategory.SUBMODULE}>
                                          <SourceCodeViewer
                                            language={filenameToLanguage(filename)}
                                            source={decodeGitContent(base64Data)}
                                          />
                                        </Case>
                                        <Case val={FileCategory.SYMLINK}>
                                          <SourceCodeViewer
                                            language={filenameToLanguage(filename)}
                                            source={decodeGitContent(base64Data)}
                                          />
                                        </Case>
                                      </Match>
                                    </Center>
                                  </Falsy>
                                </Match>
                              </Falsy>
                            </Match>
                          </Truthy>
                        </Match>
                      </Container>
                    </Render>
                  </Container>
                </Layout.Vertical>
              </Container>
            )
          },
          {
            id: FileSection.BLAME,
            title: getString('blame'),
            panel: (
              <Container className={css.gitBlame}>
                {[resourcePath + gitRef].map(key => (
                  <GitBlame
                    standalone={standalone}
                    repoMetadata={repoMetadata}
                    resourcePath={resourcePath}
                    gitRef={gitRef}
                    key={key}
                  />
                ))}
              </Container>
            )
          },
          {
            id: FileSection.HISTORY,
            title: getString('history'),
            panel: (
              <>
                {repoMetadata && !!commits?.commits?.length && (
                  <>
                    <Container className={css.gitCommit}>
                      <CommitsView
                        commits={commits.commits}
                        repoMetadata={repoMetadata}
                        emptyTitle={getString('noCommits')}
                        emptyMessage={getString('noCommitsMessage')}
                        showFileHistoryIcons={true}
                        resourcePath={resourcePath}
                        setActiveTab={setActiveTab}
                      />
                    </Container>
                    <Container className={css.gitHistory}>
                      {commits?.rename_details && repoMetadata ? (
                        <RenameContentHistory
                          rename_details={commits.rename_details}
                          repoMetadata={repoMetadata}
                          setActiveTab={setActiveTab}
                        />
                      ) : null}
                    </Container>
                  </>
                )}
              </>
            )
          }
        ]}
      />
    </Container>
  )
}

const Center: React.FC = ({ children }) => (
  <Container flex={{ align: 'center-center' }} style={{ width: '100%', height: '100%' }} padding={{ right: 'large' }}>
    {children}
  </Container>
)
