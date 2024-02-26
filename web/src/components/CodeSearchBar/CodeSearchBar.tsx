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

import React, { Dispatch, FC, SetStateAction } from 'react'

import cx from 'classnames'
import { Switch } from '@blueprintjs/core'
import { Text } from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'
import { SearchInputWithSpinner } from 'components/SearchInputWithSpinner/SearchInputWithSpinner'
import { useAppContext } from 'AppContext'
import { useStrings } from 'framework/strings'
import { SEARCH_MODE } from 'components/CodeSearch/CodeSearch'
import svg from '../CodeSearch/search-background.svg'
import css from './CodeSearchBar.module.scss'

interface CodeSearchBarProps {
  value: string
  onChange: (value: string) => void
  onSearch?: (searchTerm: string) => void
  onKeyDown?: (e: React.KeyboardEvent<HTMLElement>) => void
  setSearchMode: Dispatch<SetStateAction<SEARCH_MODE>>
  searchMode: SEARCH_MODE
}

const KEYWORD_REGEX = /((?:(?:-{0,1})(?:repo|lang|file|case|count)):\S*|(?: or|and ))/gi

const CodeSearchBar: FC<CodeSearchBarProps> = ({ value, onChange, onSearch, onKeyDown, searchMode, setSearchMode }) => {
  const { getString } = useStrings()
  const { hooks } = useAppContext()
  const { SEMANTIC_SEARCH_ENABLED: isSemanticSearchEnabled } = hooks?.useFeatureFlags()
  const isSemanticMode = isSemanticSearchEnabled && searchMode === SEARCH_MODE.SEMANTIC
  return (
    <div className={css.searchCtn} data-search-mode={searchMode}>
      <div className={css.textCtn}>
        {!isSemanticMode &&
          value.split(KEYWORD_REGEX).map(text => {
            const isMatch = text.match(KEYWORD_REGEX)

            if (text.match(/ or|and /gi)) {
              return (
                <span key={text} className={css.andOr}>
                  {text}
                </span>
              )
            }

            const separatorIdx = text.indexOf(':')
            const [keyWord, keyVal] = [text.slice(0, separatorIdx), text.slice(separatorIdx + 1)]

            if (isMatch) {
              if (keyWord.match(/ or|and /gi)) {
                return (
                  <span key={keyWord} className={css.andOr}>
                    {keyWord}
                  </span>
                )
              }

              return (
                <>
                  {keyWord}:<span className={css.highltedText}>{keyVal}</span>
                </>
              )
            } else {
              return text
            }
          })}
      </div>
      <SearchInputWithSpinner
        width={'100%' as any}
        query={value}
        spinnerPosition="right"
        setQuery={onChange}
        onSearch={onSearch}
        onKeyDown={onKeyDown}
        placeholder={isSemanticMode ? getString('codeSearchModal') : getString('keywordSearchPlaceholder')}
      />
      {isSemanticSearchEnabled && (
        <>
          {isSemanticMode ? (
            <img className={cx(css.toggleLogo)} src={svg} width={122} height={25} />
          ) : (
            <Text
              className={cx(css.toggleTitle)}
              font={{ variation: FontVariation.SMALL_SEMI }}
              color={Color.GREY_300}
              style={{ whiteSpace: 'nowrap' }}>
              {getString('enableAISearch')}
            </Text>
          )}
        </>
      )}

      {isSemanticSearchEnabled && (
        <Switch
          onChange={() => {
            searchMode === SEARCH_MODE.KEYWORD
              ? setSearchMode(SEARCH_MODE.SEMANTIC)
              : setSearchMode(SEARCH_MODE.KEYWORD)
          }}
          className={cx(css.toggleBtn)}
          checked={SEARCH_MODE.SEMANTIC === searchMode}></Switch>
      )}
      {isSemanticSearchEnabled && (
        <button
          className={cx(css.toggleHiddenBtn, css.toggleBtn)}
          onClick={() =>
            searchMode === SEARCH_MODE.KEYWORD
              ? setSearchMode(SEARCH_MODE.SEMANTIC)
              : setSearchMode(SEARCH_MODE.KEYWORD)
          }></button>
      )}
    </div>
  )
}

export default CodeSearchBar
