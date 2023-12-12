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

import React, { FC } from 'react'

import { SearchInputWithSpinner } from 'components/SearchInputWithSpinner/SearchInputWithSpinner'

import css from './KeywordSearchbar.module.scss'

interface KeywordSearchbarProps {
  value: string
  onChange: (value: string) => void
  onSearch?: (searchTerm: string) => void
  onKeyDown?: (e: React.KeyboardEvent<HTMLElement>) => void
}

const KEYWORD_REGEX = /((?:(?:-{0,1})(?:repo|lang|file|case|count)):\S*|(?: or|and ))/gi

const KeywordSearchbar: FC<KeywordSearchbarProps> = ({ value, onChange, onSearch, onKeyDown }) => {
  return (
    <div className={css.searchCtn}>
      <div className={css.textCtn}>
        {value.split(KEYWORD_REGEX).map(text => {
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
        setQuery={onChange}
        onSearch={onSearch}
        onKeyDown={onKeyDown}
      />
    </div>
  )
}

export default KeywordSearchbar
