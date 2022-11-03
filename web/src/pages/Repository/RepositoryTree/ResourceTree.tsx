import React, { useState } from 'react'
import cx from 'classnames'
import { PopoverInteractionKind } from '@blueprintjs/core'
import { Container, Color, DropDown, ButtonVariation, SplitButton, SplitButtonOption, Layout } from '@harness/uicore'
import { useStrings } from 'framework/strings'
import { OptionsMenuButton } from 'components/OptionsMenuButton/OptionsMenuButton'
import { BlueprintJsTree } from './TreeExample'
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
        <BlueprintJsTree />
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
