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

import React, { useState, Dispatch, FC, SetStateAction, useEffect } from 'react'

import cx from 'classnames'
import { Classes, PopoverInteractionKind, PopoverPosition, Switch } from '@blueprintjs/core'
import { Container, Popover, StringSubstitute, Text } from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'
import { Link, useParams } from 'react-router-dom'
import { SearchInputWithSpinner } from 'components/SearchInputWithSpinner/SearchInputWithSpinner'
import { useAppContext } from 'AppContext'
import type { Identifier } from 'utils/types'
import { useStrings } from 'framework/strings'
import { SEARCH_MODE } from 'components/CodeSearch/CodeSearch'
import svg from '../CodeSearch/search-background.svg?url'
import Regex from '../../icons/regex.svg?url'
import RegexEnabled from '../../icons/regexEnabled.svg?url'

import css from './CodeSearchBar.module.scss'

interface CodeSearchBarProps {
  value: string
  onChange: (value: string) => void
  onSearch?: (searchTerm: string) => void
  onKeyDown?: (e: React.KeyboardEvent<HTMLElement>) => void
  setSearchMode: Dispatch<SetStateAction<SEARCH_MODE>>
  searchMode: SEARCH_MODE
  regexEnabled: boolean
  setRegexEnabled: React.Dispatch<React.SetStateAction<boolean>>
}

const KEYWORD_REGEX = /((?:(?:-{0,1})(?:repo|lang|file|case|count)):\S*|(?: or|and ))/gi

const CodeSearchBar: FC<CodeSearchBarProps> = ({
  value,
  onChange,
  onSearch,
  onKeyDown,
  searchMode,
  setSearchMode,
  setRegexEnabled,
  regexEnabled
}) => {
  const { getString } = useStrings()
  const { hooks, routingId, defaultSettingsURL, isCurrentSessionPublic } = useAppContext()
  const { SEMANTIC_SEARCH_ENABLED: isSemanticSearchFFEnabled } = hooks?.useFeatureFlags()
  const { orgIdentifier, projectIdentifier } = useParams<Identifier>()
  const { data: aidaSettingResponse, loading: isAidaSettingLoading } = hooks?.useGetSettingValue({
    identifier: 'aida',
    queryParams: { accountIdentifier: routingId, orgIdentifier, projectIdentifier }
  })
  const [enableSemanticSearch, setEnableSemanticSearch] = useState<boolean>(false)
  useEffect(() => {
    setEnableSemanticSearch(
      isSemanticSearchFFEnabled && aidaSettingResponse?.data?.value == 'true' && !isCurrentSessionPublic
    )
  }, [isAidaSettingLoading, isSemanticSearchFFEnabled])
  const isSemanticMode = enableSemanticSearch && searchMode === SEARCH_MODE.SEMANTIC
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
      <Container className={css.layoutCtn}>
        {enableSemanticSearch ? (
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
            <Switch
              onChange={() => {
                searchMode === SEARCH_MODE.KEYWORD
                  ? setSearchMode(SEARCH_MODE.SEMANTIC)
                  : setSearchMode(SEARCH_MODE.KEYWORD)
              }}
              className={cx(css.toggleBtn)}
              checked={SEARCH_MODE.SEMANTIC === searchMode}></Switch>

            <button
              className={cx(css.toggleHiddenBtn, css.toggleBtn)}
              onClick={() =>
                searchMode === SEARCH_MODE.KEYWORD
                  ? setSearchMode(SEARCH_MODE.SEMANTIC)
                  : setSearchMode(SEARCH_MODE.KEYWORD)
              }></button>
          </>
        ) : (
          aidaSettingResponse?.data?.value != 'true' &&
          isSemanticSearchFFEnabled &&
          !isAidaSettingLoading && (
            <Container
              background={Color.AI_PURPLE_50}
              padding="small"
              className={css.messageContainer}
              margin={{ top: 'xsmall', right: 'small', left: 'xsmall' }}>
              <Text
                font={{ variation: FontVariation.BODY2 }}
                margin={{ bottom: 'small' }}
                icon="info-messaging"
                iconProps={{ size: 15 }}>
                {getString('turnOnSemanticSearch')}
              </Text>
              <Text font={{ variation: FontVariation.BODY2_SEMI }} margin={{ bottom: 'small' }} color={Color.GREY_450}>
                {getString('enableAIDAMessage')}
              </Text>
              <Link to={defaultSettingsURL} color={Color.AI_PURPLE_800}>
                {getString('reviewProjectSettings')}
              </Link>
            </Container>
          )
        )}
        {isSemanticSearchFFEnabled && aidaSettingResponse?.data?.value === 'true' && (
          <Container className={css.emptyContainer}></Container>
        )}
        {!isSemanticMode && (
          <>
            <Popover
              content={
                <Container padding="medium">
                  <Text font={{ variation: FontVariation.SMALL }} color={Color.WHITE}>
                    <StringSubstitute
                      str={getString('regex.tooltip')}
                      vars={{
                        regex: <strong>{getString('regex.string')}</strong>,
                        enable: regexEnabled ? getString('regex.enabled') : getString('regex.disabled'),
                        disable: regexEnabled ? getString('regex.disable') : getString('regex.enable')
                      }}
                    />
                  </Text>
                </Container>
              }
              popoverClassName={cx(css.popover, Classes.DARK)}
              interactionKind={PopoverInteractionKind.HOVER_TARGET_ONLY}
              position={PopoverPosition.BOTTOM_RIGHT}
              isDark={true}>
              <img
                className={css.toggleRegexIconContainer}
                src={regexEnabled ? RegexEnabled : Regex}
                height={22}
                width={22}
                onClick={() => {
                  setRegexEnabled(!regexEnabled)
                }}
              />
            </Popover>
          </>
        )}
      </Container>
    </div>
  )
}

export default CodeSearchBar
