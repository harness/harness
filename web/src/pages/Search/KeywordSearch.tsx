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

import KeywordSearchFilters from './KeywordSearchFilters'
import type { FileMatch, KeywordSearchResponse } from './KeywordSearch.types'

import css from './Search.module.scss'

const Search = () => {
  const { getString } = useStrings()
  const space = useGetSpaceParam()
  const { repoName } = useGetRepositoryMetadata()
  const { updateQueryParams } = useUpdateQueryParams()
  const { showError } = useToaster()

  const repoPath = repoName ? `${space}/${repoName}` : undefined

  const { q } = useQueryParams<{ q: string }>()
  const [searchTerm, setSearchTerm] = useState(q || '')

  const [selectedRepositories, setSelectedRepositories] = useState<SelectOption[]>([])
  const [selectedLanguages, setSelectedLanguages] = useState<SelectOption[]>([])

  const [searchResults, setSearchResults] = useState<KeywordSearchResponse>()

  const { mutate, loading: isSearching } = useMutate<KeywordSearchResponse>({
    path: `/api/v1/search`,
    verb: 'POST'
  })

  const debouncedSearch = useCallback(
    debounce(async (text: string) => {
      try {
        if (text.length > 2) {
          const maxResultCount = Number(text.match(/count:(\d*)/)?.[1]) || 50

          const repoPaths = selectedRepositories.map(option => String(option.value))
          let query = text.replace(/(?:repo|count):(?:[^\s]+|$)/g, '').trim()
          query = `(${query})`

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

          const res = await mutate({
            repo_paths: repoPath ? [repoPath] : repoPaths,
            space_paths: !repoPath && !repoPaths.length ? [space] : [],
            query,
            max_result_count: maxResultCount
          })

          setSearchResults(res)
        } else {
          setSearchResults(undefined)
        }
      } catch (error) {
        showError(getErrorMessage(error))
      }
    }, 300),
    [selectedLanguages, selectedRepositories, repoPath]
  )

  useEffect(() => {
    if (searchTerm) {
      debouncedSearch(searchTerm)
    }
  }, [selectedLanguages, selectedRepositories])

  return (
    <Container className={css.main}>
      <Container padding="medium" border={{ bottom: true }} flex className={css.header}>
        <KeywordSearchbar
          value={searchTerm}
          onChange={text => {
            setSearchTerm(text)
          }}
          onSearch={text => {
            setSearchResults(undefined)
            updateQueryParams({ q: text })
            debouncedSearch(text)
          }}
        />
      </Container>
      <Container padding="xlarge">
        <LoadingSpinner visible={isSearching} />
        <KeywordSearchFilters
          isRepoLevelSearch={Boolean(repoName)}
          selectedLanguages={selectedLanguages}
          selectedRepositories={selectedRepositories}
          setLanguages={setSelectedLanguages}
          setRepositories={setSelectedRepositories}
        />
        {searchResults?.file_matches.length ? (
          <>
            <Layout.Horizontal spacing="xsmall" margin={{ bottom: 'large' }}>
              <Text font={{ variation: FontVariation.SMALL_SEMI }}>
                {searchResults?.stats.total_files} {getString('files')}
              </Text>
              <Text font={{ variation: FontVariation.SMALL_SEMI }} color={Color.GREY_400}>
                {getString('results')}
              </Text>
            </Layout.Horizontal>
            {searchResults?.file_matches?.map(fileMatch => {
              if (fileMatch.matches.length) {
                return <SearchResult key={fileMatch.file_name} fileMatch={fileMatch} searchTerm={searchTerm} />
              }
            })}
          </>
        ) : null}
        <NoResultCard showWhen={() => !isSearching && !searchResults?.file_matches?.length} forSearch={true} />
      </Container>
    </Container>
  )
}

export default Search

interface CodeBlock {
  lineNumberOffset: number
  codeBlock: string
  fragmentMatches: string[]
}

export const SearchResult = ({ fileMatch, searchTerm }: { fileMatch: FileMatch; searchTerm: string }) => {
  const { routes } = useAppContext()

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
  const repoName = fileMatch.repo_path.split('/').pop()

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
