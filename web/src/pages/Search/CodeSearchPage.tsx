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

import React, { useCallback, useEffect, useMemo, useState } from 'react'
import { Container, Layout, SelectOption, Text, useToaster, useToggle } from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'
import { Icon } from '@harnessio/icons'
import { Minus, Plus } from 'iconoir-react'
import {
  Decoration,
  DecorationSet,
  EditorView,
  MatchDecorator,
  ViewPlugin,
  ViewUpdate,
  lineNumbers
} from '@codemirror/view'
import type { Extension } from '@codemirror/state'
import { Link } from 'react-router-dom'
import { useMutate } from 'restful-react'
import { debounce, escapeRegExp, flatten, sortBy, uniq } from 'lodash-es'
import Keywords from 'react-keywords'
import cx from 'classnames'
import { useHistory } from 'react-router-dom'
import { useAppContext } from 'AppContext'
import { useStrings } from 'framework/strings'
import { useQueryParams } from 'hooks/useQueryParams'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { useUpdateQueryParams } from 'hooks/useUpdateQueryParams'
import { ButtonRoleProps, getErrorMessage } from 'utils/Utils'

import { Editor } from 'components/Editor/Editor'
import { NoResultCard } from 'components/NoResultCard/NoResultCard'
import KeywordSearchbar from 'components/KeywordSearchbar/KeywordSearchbar'
import { LoadingSpinner } from 'components/LoadingSpinner/LoadingSpinner'
import { SEARCH_MODE } from 'components/CodeSearch/CodeSearch'
import CodeSearchBar from 'components/CodeSearchBar/CodeSearchBar'
import { defaultUsefulOrNot } from 'components/DefaultUsefulOrNot/UsefulOrNot'
import { AidaClient } from 'utils/types'
import KeywordSearchFilters from './KeywordSearchFilters'
import type { FileMatch, KeywordSearchResponse } from './CodeSearchPage.types'
import css from './Search.module.scss'

type SemanticSearchResultType = {
  commit: string
  file_path: string
  start_line: number
  end_line: number
  file_name: string
  lines: string[]
}

// COMMON
const Search = () => {
  const { getString } = useStrings()
  const space = useGetSpaceParam()
  const { repoName, repoMetadata } = useGetRepositoryMetadata()
  const { updateQueryParams } = useUpdateQueryParams()
  const { showError } = useToaster()
  const repoPath = repoName ? `${space}/${repoName}` : undefined

  const { q, mode, regex } = useQueryParams<{ q: string; mode: SEARCH_MODE; regex: string }>()
  const [searchTerm, setSearchTerm] = useState(q || '')
  const [searchMode, setSearchMode] = useState(mode)
  const [selectedRepositories, setSelectedRepositories] = useState<SelectOption[]>([])
  const [selectedLanguages, setSelectedLanguages] = useState<(SelectOption & { extension?: string })[]>([])
  const [keywordSearchResults, setKeyowordSearchResults] = useState<KeywordSearchResponse>()
  const projectId = space?.split('/')[2]

  const [recursiveSearchEnabled, setRecursiveSearchEnabled] = useState(!projectId ? true : false)
  const [curScopeLabel, setCurScopeLabel] = useState<SelectOption>()
  const [regexEnabled, setRegexEnabled] = useState<boolean>(regex === 'true' ? true : false)

  //semantic
  // const [loadingSearch, setLoadingSearch] = useState(false)
  const [semanticSearchResult, setSemanticSearchResult] = useState<SemanticSearchResultType[]>([])
  const [uniqueFiles, setUniqueFiles] = useState(0)
  const history = useHistory()
  //
  const { mutate, loading: isSearching } = useMutate<KeywordSearchResponse>({
    path: `/api/v1/search`,
    verb: 'POST'
  })
  const {
    mutate: sendSemanticSearch,
    loading: loadingSearch,
    cancel: cancelPreviousSearch
  } = useMutate<SemanticSearchResultType[]>({
    verb: 'POST',
    path: `/api/v1/repos/${repoMetadata?.path}/+/semantic/search`
  })
  const debouncedSearch = useCallback(
    debounce(async (text: string) => {
      try {
        if (text.length > 2) {
          const maxResultCount = Number(text.match(/count:(\d*)/)?.[1]) || 50

          const repoPaths = selectedRepositories.map(option => String(option.value))
          let query = text.replace(/(?:repo|count):(?:[^\s]+|$)/g, '').trim()
          query = `( ${query} )`

          if (selectedLanguages.length) {
            if (selectedLanguages.length === 1) {
              query += ` lang:${String(selectedLanguages[0].value)}`
            } else {
              const multiLangQuery = `(${selectedLanguages.map(option => `lang:${String(option.value)}`).join(' or ')})`
              query += ` ${multiLangQuery}`
            }
          }

          if (!query.match(/case:(yes|no)/gi)?.length) {
            query += ` case:no`
          }

          // Clear previous results
          setKeyowordSearchResults(undefined)
          const res = await mutate({
            repo_paths: repoPath ? [repoPath] : repoPaths,
            space_paths: !repoPath && !repoPaths.length ? [space] : [],
            query,
            max_result_count: maxResultCount,
            recursive: recursiveSearchEnabled,
            enable_regex: regexEnabled
          })

          setKeyowordSearchResults(res)
        } else {
          setKeyowordSearchResults(undefined)
        }
      } catch (error) {
        showError(getErrorMessage(error))
      }
    }, 300), // eslint-disable-next-line react-hooks/exhaustive-deps
    [selectedLanguages, selectedRepositories, repoPath, mode, recursiveSearchEnabled, regexEnabled]
  )

  const performSemanticSearch = useCallback(() => {
    // setLoadingSearch(true)
    // history.replace({ pathname: location.pathname, search: `q=${searchTerm}` })

    sendSemanticSearch({ query: searchTerm })
      .then(response => {
        setSemanticSearchResult(response)
        const countUniqueFiles = () => new Set(response.map(item => item.file_path)).size
        setUniqueFiles(countUniqueFiles)
      })
      .catch(exception => {
        showError(getErrorMessage(exception), 0)
      })
      .finally(() => {
        // setLoadingSearch(false)
      }) // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [searchTerm, history, location, repoPath, sendSemanticSearch, showError, mode])

  useEffect(() => {
    if (q && mode !== SEARCH_MODE.SEMANTIC) {
      debouncedSearch(searchTerm)
    } else if (searchTerm && repoMetadata?.path && mode === SEARCH_MODE.SEMANTIC) {
      performSemanticSearch()
    } // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedLanguages, selectedRepositories, repoMetadata?.path])

  useEffect(() => {
    setTimeout(() => {
      debouncedSearch(searchTerm)
    }, 0) // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [setRecursiveSearchEnabled, recursiveSearchEnabled])

  return (
    <Container className={css.main}>
      <Container padding="medium" border={{ bottom: true }} flex className={css.header}>
        {repoName ? (
          <CodeSearchBar
            searchMode={searchMode}
            setSearchMode={setSearchMode}
            value={searchTerm}
            regexEnabled={regexEnabled}
            setRegexEnabled={setRegexEnabled}
            onChange={text => {
              setSearchTerm(text)
            }}
            onSearch={text => {
              cancelPreviousSearch()
              setKeyowordSearchResults(undefined)
              setSemanticSearchResult([])
              updateQueryParams({ q: text, mode: searchMode })
              if (searchMode === SEARCH_MODE.SEMANTIC) {
                performSemanticSearch()
              } else {
                debouncedSearch(text)
              }
            }}
          />
        ) : (
          <KeywordSearchbar
            value={searchTerm}
            regexEnabled={regexEnabled}
            setRegexEnabled={setRegexEnabled}
            onChange={text => {
              setSearchTerm(text)
            }}
            onSearch={text => {
              setKeyowordSearchResults(undefined)
              updateQueryParams({ q: text })
              debouncedSearch(text)
            }}
          />
        )}
      </Container>
      <Container padding="xlarge">
        <LoadingSpinner className={css.loadingSpinner} visible={isSearching || loadingSearch} />
        {keywordSearchResults && (
          <KeywordSearchFilters
            isRepoLevelSearch={Boolean(repoName)}
            selectedLanguages={selectedLanguages}
            selectedRepositories={selectedRepositories}
            setLanguages={setSelectedLanguages}
            setRepositories={setSelectedRepositories}
            recursiveSearchEnabled={recursiveSearchEnabled}
            setRecursiveSearchEnabled={setRecursiveSearchEnabled}
            curScopeLabel={curScopeLabel}
            setCurScopeLabel={setCurScopeLabel}
          />
        )}

        {keywordSearchResults?.file_matches.length ? (
          <>
            <Layout.Horizontal spacing="xsmall" margin={{ bottom: 'large' }}>
              <Text font={{ variation: FontVariation.SMALL_SEMI }}>
                {keywordSearchResults?.stats.total_files} {getString('files')}
              </Text>
              <Text font={{ variation: FontVariation.SMALL_SEMI }} color={Color.GREY_400}>
                {getString('results')}
              </Text>
            </Layout.Horizontal>
            {keywordSearchResults?.file_matches?.map(fileMatch => {
              if (fileMatch.matches.length) {
                return <SearchResult key={fileMatch.file_name} fileMatch={fileMatch} searchTerm={searchTerm} />
              }
            })}
          </>
        ) : null}
        {/* semantic search results -> */}
        {semanticSearchResult?.length ? (
          <>
            <Layout.Horizontal spacing="xsmall" margin={{ bottom: 'large' }}>
              {!loadingSearch && (
                <Text font={{ variation: FontVariation.SMALL_SEMI }}>
                  {uniqueFiles} {getString('files')}
                </Text>
              )}
              <Text font={{ variation: FontVariation.SMALL_SEMI }} color={Color.GREY_400}>
                {getString('results')}
              </Text>
            </Layout.Horizontal>
            {loadingSearch ? (
              <Text></Text>
            ) : (
              <Text>
                {semanticSearchResult.map((result, index) => (
                  <SemanticSearchResult key={index} result={result} index={index} query={searchTerm} />
                ))}
              </Text>
            )}
          </>
        ) : null}
        <NoResultCard
          showWhen={() =>
            !isSearching &&
            !keywordSearchResults?.file_matches?.length &&
            !loadingSearch &&
            !semanticSearchResult?.length
          }
          forSearch={true}
        />
      </Container>
    </Container>
  )
}

export default Search

// KEYOWORD SEARCH CODE
interface CodeBlock {
  lineNumberOffset: number
  codeBlock: string
  fragmentMatches: string[]
}

export const SearchResult = ({ fileMatch, searchTerm }: { fileMatch: FileMatch; searchTerm: string }) => {
  const { routes } = useAppContext()
  const space = useGetSpaceParam()
  const accId = space?.split('/')[0]

  const projectId = space?.split('/')[2]
  const orgId = space?.split('/')[1]

  const [isCollapsed, setIsCollapsed] = useToggle(false)
  const [showMoreMatchs, setShowMoreMatches] = useState(false)

  const isCaseSensitive = searchTerm.includes('case:yes')

  const codeBlocks: CodeBlock[] = useMemo(() => {
    const codeBlocksArr: CodeBlock[] = []

    const sortedMatches = sortBy(fileMatch.matches, 'line_num')

    sortedMatches.forEach(keywordMatch => {
      const lines: string[] = []

      if (keywordMatch.before.trim()) {
        lines.push(keywordMatch.before)
      }

      const line: string[] = []

      keywordMatch.fragments.forEach(fragmentMatch => {
        line.push(fragmentMatch.pre + fragmentMatch.match + fragmentMatch.post)
      })

      lines.push(line.join(''))

      if (keywordMatch.after.trim()) {
        lines.push(keywordMatch.after)
      }

      const codeBlock = lines.join('\n')

      const lineNumberOffset = keywordMatch.before.trim()
        ? keywordMatch.line_num - Math.floor(codeBlock.split('\n').length / 2)
        : keywordMatch.line_num

      codeBlocksArr.push({
        lineNumberOffset,
        codeBlock,
        fragmentMatches: keywordMatch.fragments.map(fragmentMatch => escapeRegExp(fragmentMatch.match))
      })
    })

    return codeBlocksArr
  }, [fileMatch])

  const collapsedCodeBlocks = showMoreMatchs ? codeBlocks.slice(0, 25) : codeBlocks.slice(0, 2)
  const repoPathParts = fileMatch.repo_path.split('/')
  let repoName = ''

  if (accId && !orgId && !projectId) {
    repoName = repoPathParts.slice(1).join('/')
  } else if (accId && orgId && !projectId) {
    repoName = repoPathParts.slice(2).join('/')
  } else if (accId && orgId && projectId) {
    repoName = fileMatch.repo_path.split('/').pop() as string
  } else {
    repoName = fileMatch.repo_path.split('/').pop() as string
  }
  const isFileMatch = fileMatch.matches?.[0]?.line_num === 0

  const flattenedMatches = flatten(codeBlocks.map(codeBlock => codeBlock.fragmentMatches))
  const allFileMatches = isCaseSensitive ? flattenedMatches : uniq(flattenedMatches.map(match => match.toLowerCase()))
  return (
    <Container className={css.searchResult}>
      <Layout.Horizontal spacing="small" className={css.resultHeader}>
        <Icon name={isCollapsed ? 'chevron-up' : 'chevron-down'} {...ButtonRoleProps} onClick={setIsCollapsed} />
        <Link to={routes.toCODERepository({ repoPath: fileMatch.repo_path })}>
          <Text
            icon="code-repo"
            font={{ variation: FontVariation.SMALL_SEMI }}
            color={Color.GREY_500}
            border={{ right: true }}
            padding={{ right: 'small' }}>
            {repoName}
          </Text>
        </Link>
        <Link
          to={`${routes.toCODERepository({
            repoPath: fileMatch.repo_path,
            gitRef: fileMatch.repo_branch,
            resourcePath: fileMatch.file_name
          })}?keyword=${allFileMatches.join('|')}`}>
          <Text font={{ variation: FontVariation.SMALL_BOLD }} color={Color.PRIMARY_7}>
            <Keywords
              value={isFileMatch ? fileMatch.matches?.[0]?.fragments?.[0].match : ''}
              backgroundColor="var(--yellow-300)">
              {fileMatch.file_name}
            </Keywords>
          </Text>
        </Link>
      </Layout.Horizontal>
      <div className={cx({ [css.isCollapsed]: isCollapsed })}>
        {collapsedCodeBlocks.map((codeBlock, index) => {
          return (
            <CodeBlock
              key={`${fileMatch.file_name}_${index}`}
              codeBlock={codeBlock}
              fileName={fileMatch.file_name}
              isCaseSensitive={isCaseSensitive}
              showMoreMatchesFooter={codeBlocks.length > 2 && index === collapsedCodeBlocks.length - 1}
              totalCodeBlocks={codeBlocks.length}
              repoBranch={fileMatch.repo_branch}
              repoPath={fileMatch.repo_path}
              allFileMatches={allFileMatches}
              showMoreMatchs={showMoreMatchs}
              setShowMoreMatches={setShowMoreMatches}
            />
          )
        })}
      </div>
    </Container>
  )
}

const CodeBlock = ({
  codeBlock,
  fileName,
  isCaseSensitive,
  showMoreMatchesFooter,
  repoPath,
  repoBranch,
  totalCodeBlocks,
  allFileMatches,
  showMoreMatchs,
  setShowMoreMatches
}: {
  codeBlock: CodeBlock
  showMoreMatchesFooter: boolean
  isCaseSensitive: boolean
  fileName: string
  repoPath: string
  repoBranch: string
  totalCodeBlocks: number
  allFileMatches: string[]
  showMoreMatchs: boolean
  setShowMoreMatches: React.Dispatch<React.SetStateAction<boolean>>
}) => {
  const { routes } = useAppContext()
  const { getString } = useStrings()

  const matchDecoratorPlugin: Extension = useMemo(() => {
    const placeholderMatcher = new MatchDecorator({
      regexp: new RegExp(codeBlock.fragmentMatches.join('|'), isCaseSensitive ? 'g' : 'gi'),
      decoration: Decoration.mark({ class: css.highlight })
    })

    return ViewPlugin.fromClass(
      class {
        placeholders: DecorationSet
        constructor(view: EditorView) {
          this.placeholders = placeholderMatcher.createDeco(view)
        }
        update(update: ViewUpdate) {
          this.placeholders = placeholderMatcher.updateDeco(update, this.placeholders)
        }
      },
      {
        decorations: instance => instance.placeholders,
        provide: plugin =>
          EditorView.atomicRanges.of(view => {
            return view.plugin(plugin)?.placeholders || Decoration.none
          })
      }
    )
  }, [isCaseSensitive])

  /** File Match */
  if (codeBlock.lineNumberOffset === 0) {
    return null
  }

  return (
    <>
      <Editor
        inGitBlame
        content={codeBlock.codeBlock}
        standalone={true}
        readonly={true}
        repoMetadata={undefined}
        filename={fileName}
        extensions={[
          matchDecoratorPlugin,
          lineNumbers({
            formatNumber: n => String(n - 1 + codeBlock.lineNumberOffset)
          })
        ]}
        className={css.editorCtn}
      />
      {showMoreMatchesFooter ? (
        <Layout.Horizontal
          spacing="small"
          className={css.showMoreCtn}
          onClick={() => setShowMoreMatches(prevVal => !prevVal)}
          flex={{ alignItems: 'center', justifyContent: 'flex-start' }}
          {...ButtonRoleProps}>
          {!showMoreMatchs ? <Plus /> : <Minus />}
          <Text font={{ variation: FontVariation.TINY_SEMI }} color={Color.GREY_400}>
            {!showMoreMatchs ? getString('showNMoreMatches', { n: totalCodeBlocks - 2 }) : getString('showLessMatches')}
          </Text>
          {totalCodeBlocks > 25 && showMoreMatchs ? (
            <Text
              font={{ variation: FontVariation.TINY }}
              border={{ left: true }}
              padding={{ left: 'small' }}
              color={Color.GREY_400}>
              {getString('nMoreMatches', { n: totalCodeBlocks - 25 })}{' '}
              <Link
                target="_blank"
                referrerPolicy="no-referrer"
                to={`${routes.toCODERepository({
                  repoPath,
                  gitRef: repoBranch,
                  resourcePath: fileName
                })}?keyword=${allFileMatches.join('|')}`}>
                {getString('seeNMoreMatches', { n: totalCodeBlocks })}
              </Link>
            </Text>
          ) : null}
        </Layout.Horizontal>
      ) : null}
    </>
  )
}

// SEMANTIC SEARCH CODE

interface SemanticCodeBlock {
  lineNumberOffset: number
  codeBlock: string
  result: SemanticSearchResultType
}

export const SemanticSearchResult = ({
  result,
  index,
  query
}: {
  result: SemanticSearchResultType
  index: number
  query: string
}) => {
  const { routes, customComponents } = useAppContext()
  const AIDAFeedback = customComponents ? customComponents.UsefulOrNot : defaultUsefulOrNot
  const { gitRef, repoName, repoMetadata } = useGetRepositoryMetadata()
  const [isCollapsed, setIsCollapsed] = useToggle(false)
  const [showMoreMatchs, setShowMoreMatches] = useState(false)
  const totalLines = result.end_line - result.start_line + 1
  const { getString } = useStrings()
  const showLines = totalLines > 10 ? (showMoreMatchs ? result.lines : result.lines.slice(0, 10)) : result.lines
  return (
    <Container className={css.searchResult}>
      <Layout.Horizontal spacing="small" className={cx(css.resultHeader)}>
        <Icon name={isCollapsed ? 'chevron-up' : 'chevron-down'} {...ButtonRoleProps} onClick={setIsCollapsed} />
        <Link to={routes.toCODERepository({ repoPath: repoMetadata?.path as string })}>
          <Text
            icon="code-repo"
            font={{ variation: FontVariation.SMALL_SEMI }}
            color={Color.GREY_500}
            border={{ right: true }}
            padding={{ right: 'small' }}>
            {repoName}
          </Text>
        </Link>
        <Link
          to={routes.toCODERepository({
            repoPath: repoMetadata?.path as string,
            gitRef: gitRef,
            resourcePath: result.file_path
          })}>
          <Text font={{ variation: FontVariation.SMALL_BOLD }} color={Color.PRIMARY_7}>
            {result.file_path}
          </Text>
        </Link>
        <span className={css.semanticStamp}>{getString('AIDA')}</span>
      </Layout.Horizontal>
      <div className={cx({ [css.isCollapsed]: isCollapsed })}>
        <SemanticCodeBlock
          key={`${result.file_name}_${showLines}_${index}`}
          result={result}
          showMoreMatchesFooter={totalLines > 10}
          showMoreLines={showMoreMatchs}
          setShowMoreMatches={setShowMoreMatches}
          showLines={showLines}
        />
        <AIDAFeedback
          className={css.aidaFeedback}
          allowCreateTicket={true}
          allowFeedback={true}
          telemetry={{
            aidaClient: AidaClient.CODE_SEMANTIC_SEARCH,
            metadata: {
              query,
              searchResponse: JSON.stringify(result)
            }
          }}
          // onVote={() => {
          //    hide
          // }}
        />
      </div>
    </Container>
  )
}

const SemanticCodeBlock = ({
  showMoreMatchesFooter,
  result,
  showMoreLines,
  setShowMoreMatches,
  showLines
}: {
  showMoreMatchesFooter: boolean
  result: SemanticSearchResultType
  showMoreLines: boolean
  setShowMoreMatches: React.Dispatch<React.SetStateAction<boolean>>
  showLines: string[]
}) => {
  const { getString } = useStrings()
  const totalLines = result.end_line - result.start_line + 1
  if (totalLines === 0) {
    return null
  }
  return (
    <>
      <Editor
        inGitBlame
        content={showLines.join('\n')}
        standalone={true}
        readonly={true}
        repoMetadata={undefined}
        filename={result.file_name}
        extensions={[
          lineNumbers({
            formatNumber: n => String(n - 1 + result.start_line)
          })
        ]}
        className={css.editorCtn}
      />
      {showMoreMatchesFooter ? (
        <Layout.Horizontal
          spacing="small"
          className={css.showMoreCtn}
          onClick={() => setShowMoreMatches(prevVal => !prevVal)}
          flex={{ alignItems: 'center', justifyContent: 'flex-start' }}
          {...ButtonRoleProps}>
          {!showMoreLines ? <Plus /> : <Minus />}
          <Text font={{ variation: FontVariation.TINY_SEMI }} color={Color.GREY_400}>
            {!showMoreLines ? `Show ${totalLines - 10} more lines` : getString('showLessMatches')}
          </Text>
        </Layout.Horizontal>
      ) : null}
    </>
  )
}
