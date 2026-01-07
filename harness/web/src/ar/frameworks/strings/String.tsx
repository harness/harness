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

import React, { useCallback, ComponentType } from 'react'
import mustache from 'mustache'
import { get } from 'lodash-es'

import { useStringsContext, StringKeys } from './StringsContext'

export interface UseStringsReturn {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  getString(key: StringKeys, vars?: Record<string, any>): string
}

export function useStrings(): UseStringsReturn {
  const { data: strings, getString: getStringFromContext } = useStringsContext()

  const getString: UseStringsReturn['getString'] = useCallback(
    (key: StringKeys, vars: Record<string, unknown> = {}) => {
      if (getStringFromContext) {
        return getStringFromContext(key, vars)
      }
      const template = get(strings, key)

      if (typeof template !== 'string') {
        throw new Error(`No valid template with id "${key}" found in any namespace`)
      }

      return mustache.render(template, { ...vars, $: strings })
    },
    [getStringFromContext, strings]
  )

  return {
    getString
  }
}

type TagProps = {
  dangerouslySetInnerHTML?: {
    __html: string
  }
  style?: React.CSSProperties
}

export interface StringProps extends React.DetailedHTMLProps<React.HTMLAttributes<HTMLElement>, HTMLElement> {
  stringID: StringKeys
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  vars?: Record<string, any>
  useRichText?: boolean
  tagName: ComponentType<TagProps> | string
}

export function String(props: StringProps): React.ReactElement | null {
  const { stringID, vars, useRichText, tagName: Tag, ...rest } = props
  const { getString } = useStrings()

  try {
    const text = getString(stringID, vars)

    return useRichText ? (
      <Tag {...(typeof rest === 'object' ? rest : {})} dangerouslySetInnerHTML={{ __html: text }} />
    ) : (
      <Tag {...(rest as unknown as {})}>{text}</Tag>
    )
  } catch (e: any) {
    if (process.env.NODE_ENV !== 'production') {
      return <Tag style={{ color: 'var(--red-500)' }}>{e.message}</Tag>
    }

    return null
  }
}

String.defaultProps = {
  tagName: 'span'
}
