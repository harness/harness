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
import { Button, ButtonSize, ButtonVariation, Container, Dialog, Layout, Text, Utils } from '@harnessio/uicore'
import { LongArrowDownLeft, Search } from 'iconoir-react'
import { useHotkeys } from 'react-hotkeys-hook'
import { useHistory } from 'react-router-dom'
import { noop } from 'lodash-es'
import cx from 'classnames'

import { useAppContext } from 'AppContext'
import { useStrings } from 'framework/strings'
import { ButtonRoleProps } from 'utils/Utils'
import type { GitInfoProps } from 'utils/GitUtils'

import { SearchInputWithSpinner } from 'components/SearchInputWithSpinner/SearchInputWithSpinner'

import svg from './search-background.svg?url'
import css from './SemanticSearch.module.scss'

const SemanticSearch = ({ repoMetadata }: Pick<GitInfoProps, 'repoMetadata'>) => {
  const { getString } = useStrings()
  const { routes } = useAppContext()
  const history = useHistory()

  const [search, setSearch] = useState('')
  const [searchSampleQueryIndex, setSearchSampleQueryIndex] = useState<number>(0)
  const [showSearchModal, setShowSearchModal] = useState(false)

  const performSearch = useCallback(
    (q: string) => {
      history.push({
        pathname: routes.toCODESemanticSearch({
          repoPath: repoMetadata?.path as string
        }),
        search: `q=${q}`
      })
    },
    [history, repoMetadata?.path, routes]
  )
  const onSearch = useCallback(() => {
    if (search?.trim()) {
      performSearch(search)
    } else if (searchSampleQueryIndex > 0 && searchSampleQueryIndex <= searchSampleQueries.length) {
      performSearch(searchSampleQueries[searchSampleQueryIndex - 1])
    }
  }, [performSearch, search, searchSampleQueryIndex])

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
      <SearchInputWithSpinner readOnly placeholder={getString('codeSearch')} query={''} setQuery={noop} />
      {<img src={svg} width={95} height={22} />}
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
                      <Search className={css.searchIcon} width={18} height={18} color="var(--ai-purple-600)" />
                      <SearchInputWithSpinner
                        placeholder={getString('codeSearchModal')}
                        spinnerPosition="right"
                        query={search}
                        type="text"
                        setQuery={q => {
                          if (q?.trim()) {
                            setSearchSampleQueryIndex(0)
                          }
                          setSearch(q)
                        }}
                        height={40}
                        onSearch={onSearch}
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
                      {!search && <img src={svg} width={132} height={28} />}
                    </Container>
                    <Button
                      variation={ButtonVariation.AI}
                      text={getString('search')}
                      size={ButtonSize.MEDIUM}
                      onClick={onSearch}
                    />
                  </Layout.Horizontal>
                </Container>
                <Text className={css.sectionHeader}>{getString('searchHeader')}</Text>
                <Container>
                  <Layout.Vertical spacing="medium">
                    {searchSampleQueries.map((sampleQuery, index) => (
                      <Text
                        key={index}
                        className={cx(css.sampleQuery, { [css.selected]: index === searchSampleQueryIndex - 1 })}
                        {...ButtonRoleProps}
                        onClick={() => {
                          performSearch(sampleQuery)
                        }}>
                        {sampleQuery}
                        <LongArrowDownLeft color="" />
                      </Text>
                    ))}
                  </Layout.Vertical>
                </Container>
              </Layout.Vertical>
            </Container>
          </Dialog>
        </Container>
      )}
    </Container>
  )
}

export default SemanticSearch

// These sample queries are in English only - No need to do i18n for them
const searchSampleQueries = [
  `Where is the code that handles authentication?`,
  `Where is the application entry point?`,
  `Where do we configure the logger?`
]
