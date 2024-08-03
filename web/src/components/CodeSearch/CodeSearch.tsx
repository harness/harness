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

import React, { useCallback, useState } from 'react'
import cx from 'classnames'
import { Container, Dialog, Layout, Text, Utils } from '@harnessio/uicore'
import { Color } from '@harnessio/design-system'
import { Filter, LongArrowDownLeft } from 'iconoir-react'
import { useHistory } from 'react-router-dom'
import { useHotkeys } from 'react-hotkeys-hook'
import { noop } from 'lodash-es'
import { useAppContext } from 'AppContext'
import { useStrings } from 'framework/strings'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { ButtonRoleProps } from 'utils/Utils'
import type { GitInfoProps } from 'utils/GitUtils'
import { SearchInputWithSpinner } from 'components/SearchInputWithSpinner/SearchInputWithSpinner'
import CodeSearchBar from 'components/CodeSearchBar/CodeSearchBar'
import svg from './search-background.svg?url'
import css from './CodeSearch.module.scss'

interface CodeSearchProps {
  repoMetadata?: GitInfoProps['repoMetadata']
}
export enum SEARCH_MODE {
  KEYWORD = 'keyword',
  SEMANTIC = 'semantic'
}
const CodeSearch = ({ repoMetadata }: CodeSearchProps) => {
  const { getString } = useStrings()
  const { routes } = useAppContext()
  const space = useGetSpaceParam()
  const history = useHistory()

  const [search, setSearch] = useState('')
  const [showSearchModal, setShowSearchModal] = useState(false)
  const [searchSampleQueryIndex, setSearchSampleQueryIndex] = useState<number>(0)
  const [searchMode, setSearchMode] = useState(SEARCH_MODE.KEYWORD)
  const [regexEnabled, setRegexEnabled] = useState<boolean>(false)
  const performSearch = useCallback(
    (q: string, mode: SEARCH_MODE) => {
      if (repoMetadata?.path) {
        history.push({
          pathname: routes.toCODERepositorySearch({
            repoPath: repoMetadata.path as string
          }),
          search: `q=${q}&mode=${mode}&regex=${regexEnabled}`
        })
      } else {
        history.push({
          pathname: routes.toCODESpaceSearch({
            space
          }),
          search: `q=${q}&regex=${regexEnabled}`
        })
      }
    },
    [history, repoMetadata?.path, routes, searchMode, regexEnabled]
  )
  const onSearch = useCallback(() => {
    if (search?.trim()) {
      performSearch(search, searchMode)
    } else if (
      searchMode === SEARCH_MODE.SEMANTIC &&
      searchSampleQueryIndex > 0 &&
      searchSampleQueryIndex <= semanticSearchSampleQueries.length
    ) {
      performSearch(semanticSearchSampleQueries[searchSampleQueryIndex - 1], searchMode)
    } else if (
      searchMode === SEARCH_MODE.KEYWORD &&
      searchSampleQueryIndex > 0 &&
      searchSampleQueryIndex <= keywordSearchSampleQueries.length
    ) {
      performSearch(keywordSearchSampleQueries[searchSampleQueryIndex - 1].description, searchMode)
    }
  }, [performSearch, search, searchSampleQueryIndex, searchMode])

  useHotkeys(
    'ctrl+k',
    () => {
      if (!showSearchModal) {
        setShowSearchModal(true)
      }
    },
    [showSearchModal]
  )

  const isSemanticSearch = searchMode === SEARCH_MODE.SEMANTIC
  const keywordSearchSampleQueries = [
    { keyword: `class`, description: getString('keywordSearch.sampleQueries.searchForClass') },
    { keyword: `class file:^cmd`, description: getString('keywordSearch.sampleQueries.searchForFilesWithCMD') },
    {
      keyword: `class or printf`,
      description: getString('keywordSearch.sampleQueries.searchForPattern')
    },
    { keyword: 'initial commit', description: getString('keywordSearch.sampleQueries.searchForInitialCommit') }
  ]
  const semanticSearchSampleQueries = getString('semanticSearch.sampleQueries').split(',')
  return (
    <Container
      className={css.searchBox}
      {...ButtonRoleProps}
      onClick={() => {
        setShowSearchModal(true)
      }}>
      <SearchInputWithSpinner readOnly placeholder={getString('codeSearch') + ` (ctrl-k)`} query={''} setQuery={noop} />
      {isSemanticSearch && <img src={svg} width={95} height={22} />}
      {showSearchModal && (
        <Container onClick={Utils.stopEvent}>
          <Dialog
            className={css.searchModal}
            backdropClassName={css.backdrop}
            data-search-mode={searchMode}
            portalClassName={css.portal}
            isOpen={true}
            enforceFocus={false}
            onClose={() => {
              setShowSearchModal(false)
            }}>
            <Container>
              <Layout.Vertical spacing="large">
                <Container>
                  <Layout.Horizontal className={css.layout}>
                    <Container className={css.searchContainer}>
                      <CodeSearchBar
                        setRegexEnabled={setRegexEnabled}
                        regexEnabled={regexEnabled}
                        searchMode={searchMode}
                        setSearchMode={setSearchMode}
                        value={search}
                        onChange={setSearch}
                        onSearch={onSearch}
                        onKeyDown={e => {
                          if (isSemanticSearch) {
                            if (!search?.trim()) {
                              switch (e.key) {
                                case 'ArrowDown':
                                  setSearchSampleQueryIndex(index => {
                                    return index + 1 > semanticSearchSampleQueries.length ? 1 : index + 1
                                  })
                                  break
                                case 'ArrowUp':
                                  setSearchSampleQueryIndex(index => {
                                    return index - 1 > 0 ? index - 1 : semanticSearchSampleQueries.length
                                  })
                                  break
                              }
                            }
                          } else {
                            if (!search?.trim()) {
                              switch (e.key) {
                                case 'ArrowDown':
                                  setSearchSampleQueryIndex(index => {
                                    return index + 1 > keywordSearchSampleQueries.length ? 1 : index + 1
                                  })
                                  break
                                case 'ArrowUp':
                                  setSearchSampleQueryIndex(index => {
                                    return index - 1 > 0 ? index - 1 : keywordSearchSampleQueries.length
                                  })
                                  break
                              }
                            }
                          }
                        }}
                      />
                    </Container>
                  </Layout.Horizontal>
                </Container>

                <Text className={css.sectionHeader}>
                  {isSemanticSearch ? getString('searchHeader') : getString('searchExamples')}
                </Text>
                <Container>
                  {isSemanticSearch
                    ? semanticSearchSampleQueries.map((sampleQuery, index) => (
                        <Text
                          key={index}
                          className={cx(css.sampleSemanticSearchQuery, {
                            [css.selected]: index === searchSampleQueryIndex - 1
                          })}
                          {...ButtonRoleProps}
                          onClick={() => {
                            setSearch(sampleQuery)
                          }}>
                          {sampleQuery}
                          <LongArrowDownLeft color="" />
                        </Text>
                      ))
                    : keywordSearchSampleQueries.map((sampleQuery: { keyword: string; description: string }) => {
                        const { keyword, description } = sampleQuery

                        return (
                          <Layout.Horizontal
                            key={keyword}
                            className={cx(css.sampleKeywordSearchQuery, {
                              [css.selected]:
                                keywordSearchSampleQueries.indexOf(sampleQuery) === searchSampleQueryIndex - 1
                            })}
                            padding={'small'}
                            spacing={'xsmall'}
                            flex={{ alignItems: 'center', justifyContent: 'flex-start' }}
                            onClick={() => setSearch(keyword)}>
                            <Filter />
                            <Text color={Color.PRIMARY_7} font={{ weight: 'semi-bold' }}>
                              {keyword}
                            </Text>
                            <Text color={Color.GREY_600}>{description}</Text>
                          </Layout.Horizontal>
                        )
                      })}
                </Container>
              </Layout.Vertical>
            </Container>
          </Dialog>
        </Container>
      )}
    </Container>
  )
}

export default CodeSearch
