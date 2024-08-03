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
import { Filter } from 'iconoir-react'
import { useHistory } from 'react-router-dom'
import { useHotkeys } from 'react-hotkeys-hook'
import { noop } from 'lodash-es'
import { useAppContext } from 'AppContext'
import { useStrings } from 'framework/strings'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { ButtonRoleProps } from 'utils/Utils'
import type { GitInfoProps } from 'utils/GitUtils'
import { SearchInputWithSpinner } from 'components/SearchInputWithSpinner/SearchInputWithSpinner'
import KeywordSearchbar from 'components/KeywordSearchbar/KeywordSearchbar'
import css from './KeywordSearch.module.scss'

interface KeywordSearchProps {
  repoMetadata?: GitInfoProps['repoMetadata']
}

const KeywordSearch = ({ repoMetadata }: KeywordSearchProps) => {
  const { getString } = useStrings()
  const { routes } = useAppContext()
  const space = useGetSpaceParam()
  const history = useHistory()
  const [regexEnabled, setRegexEnabled] = useState<boolean>(false)

  const [search, setSearch] = useState('')
  const [showSearchModal, setShowSearchModal] = useState(false)
  const [searchSampleQueryIndex, setSearchSampleQueryIndex] = useState<number>(0)

  const performSearch = useCallback(
    (q: string) => {
      if (repoMetadata?.path) {
        history.push({
          pathname: routes.toCODERepositorySearch({
            repoPath: repoMetadata.path as string
          }),
          search: `q=${q}&regex=${regexEnabled}`
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
    [history, repoMetadata?.path, routes, regexEnabled]
  )
  const onSearch = useCallback(() => {
    if (search?.trim()) {
      performSearch(search)
    }
  }, [performSearch, search])

  useHotkeys(
    'ctrl+k',
    () => {
      if (!showSearchModal) {
        setShowSearchModal(true)
      }
    },
    [showSearchModal]
  )

  return (
    <Container
      className={css.searchBox}
      {...ButtonRoleProps}
      onClick={() => {
        setShowSearchModal(true)
      }}>
      <SearchInputWithSpinner readOnly placeholder={getString('codeSearch') + ` (ctrl-k)`} query={''} setQuery={noop} />
      {showSearchModal && (
        <Container onClick={Utils.stopEvent}>
          <Dialog
            className={css.searchModal}
            backdropClassName={css.backdrop}
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
                      <KeywordSearchbar
                        value={search}
                        onChange={setSearch}
                        onSearch={onSearch}
                        regexEnabled={regexEnabled}
                        setRegexEnabled={setRegexEnabled}
                        onKeyDown={e => {
                          if (!search?.trim()) {
                            switch (e.key) {
                              case 'ArrowDown':
                                setSearchSampleQueryIndex(index => {
                                  return index + 1 > searchSampleQueries.length ? 1 : index + 1
                                })
                                break
                              case 'ArrowUp':
                                setSearchSampleQueryIndex(index => {
                                  return index - 1 > 0 ? index - 1 : searchSampleQueries.length
                                })
                                break
                            }
                          }
                        }}
                      />
                    </Container>
                  </Layout.Horizontal>
                </Container>
                <Text className={css.sectionHeader}>{getString('searchExamples')}</Text>
                <Container>
                  {searchSampleQueries.map(sampleQuery => {
                    const { keyword, description } = sampleQuery

                    return (
                      <Layout.Horizontal
                        key={keyword}
                        className={cx(css.sampleQuery, {
                          [css.selected]: searchSampleQueries.indexOf(sampleQuery) === searchSampleQueryIndex - 1
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

export default KeywordSearch

const searchSampleQueries = [
  { keyword: `class`, description: 'Search for class' },
  { keyword: `class file:^cmd`, description: 'Search for class in files starting with cmd' },
  {
    keyword: `class or printf`,
    description: 'Include only results from file paths matching the given search pattern'
  },
  { keyword: 'initial commit', description: 'Search for exact phrase initial commit' }
]
