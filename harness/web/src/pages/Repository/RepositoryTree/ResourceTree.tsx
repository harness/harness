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

import React, { useState } from 'react'
import cx from 'classnames'
import { PopoverInteractionKind } from '@blueprintjs/core'
import { Container, DropDown, ButtonVariation, SplitButton, SplitButtonOption, Layout } from '@harnessio/uicore'
import { Color } from '@harnessio/design-system'
import { useStrings } from 'framework/strings'
import { OptionsMenuButton } from 'components/OptionsMenuButton/OptionsMenuButton'
import { TreeExample } from './TreeExample'
import css from './ResourceTree.module.scss'

export function ResourceTree() {
  const { getString } = useStrings()
  const [branch, setBranch] = useState('dev')

  return (
    <Container className={cx(css.tabContent, css.resourceTree)} background={Color.WHITE}>
      <Container padding="xlarge" className={css.repoBranch}>
        <Layout.Horizontal>
          <DropDown
            icon="git-branch"
            className={css.dropdown}
            value={branch}
            items={[
              { value: 'dev', label: 'dev' },
              { value: 'master', label: 'master' },
              { value: 'dev1', label: 'dev1' },
              { value: 'release/1', label: 'release/1' }
            ]}
            onChange={e => setBranch(e.value as string)}
            popoverClassName={css.branchDropdown}
          />
          <OptionsMenuButton
            items={[
              { text: 'New File...', icon: 'document' },
              { text: 'New Folder...', icon: 'folder-new' },
              { text: 'Import', icon: 'import' }
            ]}
            tooltipProps={{ minimal: true, interactionKind: PopoverInteractionKind.CLICK }}
          />
        </Layout.Horizontal>
      </Container>
      <Container className={css.fileTree}>
        <TreeExample />
      </Container>
      <Container padding="xlarge" className={css.fileNewActions}>
        <SplitButton text={getString('newFile')} icon="plus" variation={ButtonVariation.SECONDARY}>
          <SplitButtonOption
            icon="folder-new"
            text={getString('newFolder')}
            onClick={() => {
              alert('TBD')
            }}
          />
        </SplitButton>
      </Container>
    </Container>
  )
}
