/*
 * Copyright 2024 Harness, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import React, { useEffect, useState } from 'react'
import { debounce, isEmpty } from 'lodash-es'
import { Menu, MenuItem } from '@blueprintjs/core'
import { Button, ButtonSize, ButtonVariation, Container, TextInput } from '@harnessio/uicore'

import { useStrings } from '@ar/frameworks/strings'

import type { AIOption } from './types'
import css from './AISearchInput.module.scss'

interface AISearchInputProps {
  searchTerm: string
  onChange: (val: string) => void
  placeholder?: string
  options: AIOption[]
}

function AISearchInput({ searchTerm, onChange, placeholder, options }: AISearchInputProps) {
  const [query, setQuery] = useState(searchTerm)
  const [focus, setFocus] = useState(false)
  const { getString } = useStrings()

  useEffect(() => {
    if (isEmpty(searchTerm)) {
      setQuery('')
    }
  }, [searchTerm])

  const deboucedOnChange = debounce(onChange, 500)
  const deboucedSetFocus = debounce(setFocus, 400)

  const handleChange = (value: string) => {
    setQuery(value)
    deboucedOnChange(value)
  }
  return (
    <Container className={css.popoverContainer}>
      <TextInput
        value={query}
        wrapperClassName={css.textInputWrapper}
        onChange={(evt: React.ChangeEvent<HTMLInputElement>) => handleChange(evt.currentTarget.value as string)}
        placeholder={placeholder ?? getString('search')}
        leftIcon="search"
        onFocus={() => {
          setFocus(true)
        }}
        onBlur={() => {
          deboucedSetFocus(false)
        }}
        leftIconProps={{
          name: 'search',
          className: css.iconClass
        }}
        rightElement="harness-copilot"
        rightElementProps={{
          className: css.rightIconClass
        }}
      />
      <Button
        className={css.aiButton}
        icon="harness-copilot"
        variation={ButtonVariation.AI}
        text={getString('harnessAI')}
        size={ButtonSize.SMALL}
      />
      {focus && (
        <Container width={600} className={css.popoverContent}>
          <Menu>
            {options.map(each => (
              <MenuItem
                key={each.value}
                text={each.label}
                disabled={each.disabled}
                onClick={() => {
                  handleChange(each.value)
                  setFocus(false)
                }}
              />
            ))}
          </Menu>
        </Container>
      )}
    </Container>
  )
}

export default AISearchInput
