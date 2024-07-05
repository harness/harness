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

import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import {
  Button,
  ButtonSize,
  ButtonVariation,
  Container,
  Layout,
  PageBody,
  stringSubstitute,
  Text,
  useToaster
} from '@harnessio/uicore'
import cx from 'classnames'
import { lineNumbers, ViewUpdate } from '@codemirror/view'
import { Breadcrumbs, IBreadcrumbProps } from '@blueprintjs/core'
import { Link, useHistory, useLocation } from 'react-router-dom'
import { EditorView } from '@codemirror/view'
import { Match, Truthy, Falsy } from 'react-jsx-match'
import { Icon } from '@harnessio/icons'
import { useMutate } from 'restful-react'
import { Editor } from 'components/Editor/Editor'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { useStrings } from 'framework/strings'
import { RepositoryPageHeader } from 'components/RepositoryPageHeader/RepositoryPageHeader'
import { Split } from 'components/Split/Split'
import { CodeIcon, normalizeGitRef, decodeGitContent } from 'utils/GitUtils'
import { LoadingSpinner } from 'components/LoadingSpinner/LoadingSpinner'
import { useQueryParams } from 'hooks/useQueryParams'
import { useAppContext } from 'AppContext'
import { SearchInputWithSpinner } from 'components/SearchInputWithSpinner/SearchInputWithSpinner'
import { voidFn, getErrorMessage, ButtonRoleProps } from 'utils/Utils'
import type { RepoFileContent, RepoRepositoryOutput } from 'services/code'
import { useShowRequestError } from 'hooks/useShowRequestError'
import { NoResultCard } from 'components/NoResultCard/NoResultCard'
import { useGetResourceContent } from 'hooks/useGetResourceContent'
import { addClassToLinesExtension } from 'utils/codemirror/addClassToLinesExtension'
import css from './Search.module.scss'

export default function Search() {
  const { showError } = useToaster()
  const history = useHistory()
  const location = useLocation()
  const highlightedLines = useRef<number[]>([])
  const [highlightlLineNumbersExtension, updateHighlightlLineNumbers] = useMemo(
    () => addClassToLinesExtension([], css.highlightLineNumber),
    []
  )
  const extensions = useMemo(() => {
    return [
      lineNumbers({
        formatNumber: (lineNo: number) => lineNo.toString()
      }),
      highlightlLineNumbersExtension
    ]
  }, [highlightlLineNumbersExtension])
  const viewRef = useRef<EditorView>()
  const { getString } = useStrings()
  const { routes, standalone } = useAppContext()
  const { q } = useQueryParams<{ q: string }>()
  const [searchTerm, setSearchTerm] = useState(q || '')
  const { repoMetadata, error, loading, refetch } = useGetRepositoryMetadata()
  const [resourcePath, setResourcePath] = useState('')
  const [filename, setFileName] = useState('')
  const gitRef = useMemo(() => repoMetadata?.default_branch || '', [repoMetadata])
  const breadcrumbs = useMemo(() => {
    return repoMetadata?.path
      ? resourcePath.split('/').map((_path, index, paths) => {
          const pathAtIndex = paths.slice(0, index + 1).join('/')
          const href = routes.toCODERepository({
            repoPath: repoMetadata.path as string,
            gitRef,
            resourcePath: pathAtIndex
          })

          return { href, text: _path }
        })
      : []
  }, [resourcePath, repoMetadata, gitRef, routes])
  const onSelectResult = useCallback(
    (fileName: string, filePath: string, _content: string, _highlightedLines: number[]) => {
      updateHighlightlLineNumbers(_highlightedLines, viewRef.current)
      highlightedLines.current = _highlightedLines
      setFileName(fileName)
      setResourcePath(filePath)
    },
    [updateHighlightlLineNumbers]
  )
  const {
    data: resourceContent,
    error: resourceError = null,
    loading: resourceLoading
  } = useGetResourceContent({
    repoMetadata,
    gitRef: normalizeGitRef(gitRef) as string,
    resourcePath,
    includeCommit: false,
    lazy: !resourcePath
  })
  const fileContent: string = useMemo(
    () =>
      resourceContent?.path === resourcePath
        ? decodeGitContent((resourceContent?.content as RepoFileContent)?.data)
        : resourceError
        ? getString('failedToFetchFileContent')
        : '',

    [resourceContent?.content, resourceContent?.path, resourcePath, resourceError, getString]
  )

  // eslint-disable-next-line react-hooks/exhaustive-deps
  const onViewUpdate = useCallback(({ view, docChanged }: ViewUpdate) => {
    const firstLine = (highlightedLines.current || [])[0]

    if (docChanged && firstLine > 0 && view.state.doc.lines >= firstLine) {
      view.dispatch({
        effects: EditorView.scrollIntoView(view.state.doc.line(firstLine).from, { y: 'start', yMargin: 18 * 2 })
      })
    }
  }, [])
  const [loadingSearch, setLoadingSearch] = useState(false)
  const { mutate: sendSearch } = useMutate<SearchResultType[]>({
    verb: 'POST',
    path: `/api/v1/repos/${repoMetadata?.path}/+/semantic/search`
  })
  const [searchResult, setSearchResult] = useState<SearchResultType[]>([])
  const performSearch = useCallback(() => {
    setLoadingSearch(true)
    history.replace({ pathname: location.pathname, search: `q=${searchTerm}` })

    sendSearch({ query: searchTerm })
      .then(response => {
        setSearchResult(response)
      })
      .catch(exception => {
        showError(getErrorMessage(exception), 0)
      })
      .finally(() => {
        setLoadingSearch(false)
      })
  }, [searchTerm, history, location, sendSearch, showError])

  useEffect(() => {
    if (q && repoMetadata?.path) {
      performSearch()
    }
  }, [repoMetadata?.path]) // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    if (fileContent && fileContent !== viewRef?.current?.state.doc.toString()) {
      viewRef?.current?.dispatch({
        changes: { from: 0, to: viewRef?.current?.state.doc.length, insert: fileContent }
      })
    }
  }, [fileContent])

  useShowRequestError(resourceError)
  const hideContent = false
  return (
    <Container className={css.main}>
      <RepositoryPageHeader
        repoMetadata={repoMetadata}
        title={getString('search')}
        dataTooltipId="semanticSearch"
        content={
          hideContent ? (
            <Container className={css.searchBox}>
              <SearchInputWithSpinner
                spinnerPosition="right"
                query={searchTerm}
                setQuery={setSearchTerm}
                onSearch={performSearch}
              />
              <Button
                variation={ButtonVariation.PRIMARY}
                text={getString('search')}
                disabled={!searchTerm.trim()}
                onClick={performSearch}
                loading={loadingSearch}
              />
            </Container>
          ) : null
        }
        className={css.pageHeader}
      />
      <PageBody error={getErrorMessage(error)} retryOnError={voidFn(refetch)}>
        <LoadingSpinner visible={loading || resourceLoading || loadingSearch} withBorder />

        <Match expr={q}>
          <Truthy>
            <Split split="vertical" className={css.split} size={450} minSize={300} maxSize={700} primary="first">
              <SearchResults
                standalone={standalone}
                repoMetadata={repoMetadata}
                onSelect={onSelectResult}
                data={searchResult}
              />

              <Layout.Vertical className={cx(css.preview, { [css.noResult]: !searchResult?.length })}>
                <Container className={css.filePath}>
                  <Container>
                    <Layout.Horizontal spacing="small">
                      <Link
                        className={css.pathText}
                        to={routes.toCODERepository({
                          repoPath: (repoMetadata?.path as string) || '',
                          gitRef
                        })}>
                        <Icon name={CodeIcon.Folder} />
                      </Link>
                      <Text inline className={css.pathText}>
                        /
                      </Text>
                      <Breadcrumbs
                        items={breadcrumbs}
                        breadcrumbRenderer={({ text, href }: IBreadcrumbProps) => {
                          return (
                            <Link to={href as string}>
                              <Text inline className={css.pathText}>
                                {text}
                              </Text>
                            </Link>
                          )
                        }}
                      />
                    </Layout.Horizontal>
                  </Container>
                  <Button
                    variation={ButtonVariation.SECONDARY}
                    text={getString('viewFile')}
                    size={ButtonSize.SMALL}
                    rightIcon="chevron-right"
                    onClick={() => {
                      history.push(
                        routes.toCODERepository({
                          repoPath: (repoMetadata?.path as string) || '',
                          gitRef,
                          resourcePath
                        })
                      )
                    }}
                  />
                </Container>
                <Container className={css.fileContent}>
                  <Editor
                    standalone={standalone}
                    repoMetadata={repoMetadata}
                    viewRef={viewRef}
                    filename={filename}
                    content={fileContent}
                    readonly={true}
                    onViewUpdate={onViewUpdate}
                    extensions={extensions}
                    maxHeight="auto"
                  />
                </Container>
              </Layout.Vertical>
            </Split>
          </Truthy>
          <Falsy>
            <Container className={css.noResultContainer}>
              <NoResultCard
                showWhen={() => true}
                forSearch={true}
                title={getString('comingSoon')}
                emptySearchMessage={getString('poweredByAI')}
              />
            </Container>
          </Falsy>
        </Match>
      </PageBody>
    </Container>
  )
}

interface SearchResultsProps {
  data: SearchResultType[]
  onSelect: (fileName: string, filePath: string, content: string, highlightedLines: number[]) => void
  repoMetadata: RepoRepositoryOutput | undefined
  standalone: boolean
}

const SearchResults: React.FC<SearchResultsProps> = ({ data, onSelect, repoMetadata, standalone }) => {
  const { getString } = useStrings()
  const [selected, setSelected] = useState(data?.[0]?.file_path || '')
  const count = useMemo(() => data?.length || 0, [data])

  useEffect(() => {
    if (data?.length) {
      const item = data[0]
      onSelect(
        item.file_name,
        item.file_path,
        (item.lines || []).join('\n').replace(/^\n/g, '').trim(),
        range(item.start_line, item.end_line)
      )
    }
  }, [data, onSelect])

  return (
    <Container className={css.searchResult}>
      <Layout.Vertical spacing="medium">
        <Text className={css.resultTitle}>
          {stringSubstitute(getString('searchResult'), { count: count - 4 ? `(${count})` : ' ' })}
        </Text>
        {data.map(item => (
          <Container
            key={item.file_path}
            className={cx(css.result, { [css.selected]: item.file_path === selected })}
            {...ButtonRoleProps}
            onClick={() => {
              setSelected(item.file_path)
              onSelect(
                item.file_name,
                item.file_path,
                (item.lines || []).join('\n').replace(/^\n/g, '').trim(),
                range(item.start_line, item.end_line)
              )
            }}>
            <Layout.Vertical spacing="small">
              <Container>
                <Layout.Horizontal className={css.layout}>
                  <Container className={css.texts}>
                    <Layout.Vertical spacing="xsmall">
                      <Text className={css.filename} lineClamp={1}>
                        {item.file_name}
                      </Text>
                      <Text className={css.path} lineClamp={1}>
                        {item.file_path}
                      </Text>
                    </Layout.Vertical>
                  </Container>
                  <Text inline className={css.aiLabel}>
                    {getString('aiSearch')}
                  </Text>
                </Layout.Horizontal>
              </Container>
              <Editor
                standalone={standalone}
                repoMetadata={repoMetadata}
                filename={item.file_name}
                content={(item.lines || []).join('\n').replace(/^\n/g, '').trim()}
                readonly={true}
                maxHeight="200px"
                darkTheme
              />
            </Layout.Vertical>
          </Container>
        ))}
      </Layout.Vertical>
    </Container>
  )
}

type SearchResultType = {
  commit: string
  file_path: string
  start_line: number
  end_line: number
  file_name: string
  lines: string[]
}

const range = (start: number, stop: number, step = 1) =>
  Array.from({ length: (stop - start) / step + 1 }, (_, i) => start + i * step)
