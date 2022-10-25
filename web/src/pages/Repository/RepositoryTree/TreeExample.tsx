import * as React from 'react'
import { FocusStyleManager } from '@blueprintjs/core'
import { StaticTreeDataProvider, Tree, UncontrolledTreeEnvironment } from 'react-complex-tree'
import { Container } from '@harness/uicore'
import { renderers } from './renderers'
import { sampleTree } from './demodata'

const TREE_ID = 'repoTree'

export const BlueprintJsTree = (): JSX.Element => (
  <Container onMouseDown={FocusStyleManager.onlyShowFocusOnTabs} onKeyDown={FocusStyleManager.alwaysShowFocus}>
    <UncontrolledTreeEnvironment<string>
      canDragAndDrop={false}
      canDropOnItemWithChildren={true}
      canReorderItems={true}
      dataProvider={new StaticTreeDataProvider(sampleTree.items, (item, data) => ({ ...item, data }))}
      getItemTitle={item => item.data}
      canSearchByStartingTyping={false}
      keyboardBindings={{
        startSearch: ['f1']
      }}
      viewState={{
        [TREE_ID]: {
          expandedItems: ['src/components']
        }
      }}
      onRenameItem={(item, name) => alert(`${item.data} renamed to ${name}`)}
      onFocusItem={(data, _treeId) => console.log('Focus', data)}
      onSelectItems={(data, _treeId) => console.log('Selected', data)}
      {...renderers}>
      <Tree treeId={TREE_ID} rootItem="root" />
    </UncontrolledTreeEnvironment>
  </Container>
)
