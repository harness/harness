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

import * as React from 'react'
import { FocusStyleManager } from '@blueprintjs/core'
import { StaticTreeDataProvider, Tree, UncontrolledTreeEnvironment } from 'react-complex-tree'
import { Container } from '@harnessio/uicore'
import { renderers } from './renderers'
import { sampleTree } from './demodata'

const TREE_ID = 'repoTree'

export const TreeExample = (): JSX.Element => (
  <Container onMouseDown={FocusStyleManager.onlyShowFocusOnTabs} onKeyDown={FocusStyleManager.alwaysShowFocus}>
    <UncontrolledTreeEnvironment<string>
      canDragAndDrop={false}
      canDropOnItemWithChildren={true}
      canReorderItems={true}
      dataProvider={new StaticTreeDataProvider(sampleTree.items, (item, data) => ({ ...item, data }))}
      getItemTitle={item => item.data}
      canSearchByStartingTyping={true}
      keyboardBindings={{
        startSearch: ['f1']
      }}
      viewState={{
        [TREE_ID]: {
          expandedItems: [
            'config',
            'cypress',
            'cypress/integration',
            'cypress/videos',
            'src',
            'src/components',
            'scripts'
          ]
        }
      }}
      onRenameItem={(item, name) => alert(`${item.data} renamed to ${name}`)}
      onFocusItem={(data, _treeId) => alert('Focus' + data)}
      onSelectItems={(data, _treeId) => alert('Selected' + data)}
      {...renderers}>
      <Tree treeId={TREE_ID} rootItem="root" />
    </UncontrolledTreeEnvironment>
  </Container>
)
