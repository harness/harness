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

import React from 'react'
import { Button, ButtonSize, ButtonVariation } from '@harnessio/uicore'

import { useStrings } from '@ar/frameworks/strings'

import TreeNode, { TreeNodeProps } from './TreeNode'

export default function TreeLoadMoreNode(props: Omit<TreeNodeProps, 'heading'>) {
  const { getString } = useStrings()
  return (
    <TreeNode
      disabled
      level={props.level}
      heading={
        <Button
          variation={ButtonVariation.LINK}
          size={ButtonSize.SMALL}
          minimal
          onClick={props.onClick}
          disabled={props.disabled}>
          {getString('loadMore')}
        </Button>
      }
    />
  )
}
