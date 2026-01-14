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

import React, { ElementType } from 'react'
import mustache from 'mustache'
import { get } from 'lodash-es'
import { useStringsContext, StringKeys } from './StringsContext'

export interface UseStringsReturn {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  getString(key: StringKeys, vars?: Record<string, any>): string
}

export function useStrings(): UseStringsReturn {
  const { data: strings, getString } = useStringsContext()

  return {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    getString(key: StringKeys, vars: Record<string, any> = {}) {
      if (typeof getString === 'function') {
        return getString(key, vars)
      }

      const template = get(strings, key)

      if (typeof template !== 'string') {
        throw new Error(`No valid template with id "${key}" found in any namespace`)
      }

      return mustache.render(template, { ...vars, $: strings })
    }
  }
}

export interface StringProps extends React.DetailedHTMLProps<React.HTMLAttributes<HTMLElement>, HTMLElement> {
  stringID: StringKeys
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  vars?: Record<string, any>
  useRichText?: boolean
  tagName: keyof JSX.IntrinsicElements
}

export function String(props: StringProps): React.ReactElement | null {
  const { stringID, vars, useRichText, tagName, ...rest } = props
  const { getString } = useStrings()
  const Tag = tagName as ElementType

  try {
    const text = getString(stringID, vars)

    return useRichText ? (
      <Tag {...(rest as unknown as {})} dangerouslySetInnerHTML={{ __html: text }} />
    ) : (
      <Tag {...(rest as unknown as {})}>{text}</Tag>
    )
  } catch (e) {
    if (process.env.NODE_ENV !== 'production') {
      return <Tag style={{ color: 'var(--red-500)' }}>{get(e, 'message', e)}</Tag>
    }

    return null
  }
}

String.defaultProps = {
  tagName: 'span'
}
