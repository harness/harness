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

import React, { useRef } from 'react'
import { Button, ButtonVariation, Layout, TextInput } from '@harnessio/uicore'
import type { TextInputProps } from '@harnessio/uicore/dist/components/TextInput/TextInput'

import { useStrings } from '@ar/frameworks/strings'

import css from './ManageMetadata.module.scss'

interface SeachInputProps extends Omit<TextInputProps, 'onChange' | 'rightElement' | 'leftElement'> {
  onChange: (query: string) => void
  rightElement?: React.ReactNode
  leftElement?: React.ReactNode
}

function SearchInput(props: SeachInputProps) {
  const { rightElement, leftElement, value, ...rest } = props
  const inputRef = useRef<HTMLInputElement | null>(null)
  const { getString } = useStrings()

  const handleChangeQuery = (e: React.ChangeEvent<HTMLInputElement>) => {
    const _value = e.currentTarget.value || ''
    props.onChange(_value)
  }

  return (
    <Layout.Horizontal
      className={css.searchInputContainer}
      spacing="xsmall"
      flex={{ alignItems: 'center', justifyContent: 'flex-start' }}>
      {leftElement}
      <TextInput
        {...rest}
        inputRef={ref => {
          // Using callback ref pattern which is safe and supported by TextInput
          if (ref) {
            inputRef.current = ref
          }
        }}
        value={value}
        wrapperClassName={css.searchInput}
        autoFocus
        placeholder={props.placeholder || getString('search')}
        onChange={handleChangeQuery}
      />
      {value && (
        <Button
          variation={ButtonVariation.ICON}
          icon="code-close"
          onClick={() => {
            props.onChange('')
            inputRef.current?.focus()
          }}
        />
      )}
    </Layout.Horizontal>
  )
}

export default SearchInput
