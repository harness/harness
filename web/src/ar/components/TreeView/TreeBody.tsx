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

import React, { type PropsWithChildren } from 'react'
import { Spinner } from '@blueprintjs/core'
import { PageError, Text } from '@harnessio/uicore'

import { useStrings } from '@ar/frameworks/strings'

import TreeNode from './TreeNode'

interface TreeBodyProps {
  loading?: boolean
  error?: string
  retryOnError?: () => void
  isEmpty?: boolean
  emptyDataMessage?: string
}

export default function TreeBody(props: PropsWithChildren<TreeBodyProps>) {
  const { loading, error, retryOnError, emptyDataMessage, isEmpty } = props
  const { getString } = useStrings()
  if (loading) return <Spinner size={Spinner.SIZE_SMALL} />
  if (error) return <PageError message={error} onClick={retryOnError} />
  if (isEmpty)
    return (
      <TreeNode
        disabled
        level={1}
        isLastChild
        heading={<Text>{emptyDataMessage ?? getString('noResultsFound')}</Text>}
      />
    )
  return <>{props.children}</>
}
