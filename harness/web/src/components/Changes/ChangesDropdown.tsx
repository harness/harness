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

import React from 'react'
import { Container, ButtonVariation, Layout, Text, StringSubstitute, Button } from '@harnessio/uicore'
import { Color } from '@harnessio/design-system'
import { Icon } from '@harnessio/icons'
import { Menu, MenuItem } from '@blueprintjs/core'
import { useStrings } from 'framework/strings'
import { CodeIcon } from 'utils/GitUtils'
import { PipeSeparator } from 'components/PipeSeparator/PipeSeparator'
import type { DiffFileEntry } from 'utils/types'
import css from './ChangesDropdown.module.scss'

interface ChangesDropdownProps {
  diffs: DiffFileEntry[]
  onJumpToFile: (diff: DiffFileEntry) => void
}

export const ChangesDropdown: React.FC<ChangesDropdownProps> = ({ diffs, onJumpToFile }) => {
  const { getString } = useStrings()

  return (
    <Button
      variation={ButtonVariation.LINK}
      className={css.link}
      tooltip={
        <Container padding="small" className={css.filesMenu}>
          <Menu>
            {diffs?.map((diff, index) => (
              <MenuItem
                key={index}
                className={css.menuItem}
                icon={<Icon name={CodeIcon.File} padding={{ right: 'xsmall' }} />}
                labelElement={
                  <Layout.Horizontal spacing="xsmall">
                    {!!diff.addedLines && (
                      <Text color={Color.GREEN_600} style={{ fontSize: '12px' }}>
                        +{diff.addedLines}
                      </Text>
                    )}
                    {!!diff.addedLines && !!diff.deletedLines && <PipeSeparator height={8} />}
                    {!!diff.deletedLines && (
                      <Text color={Color.RED_500} style={{ fontSize: '12px' }}>
                        -{diff.deletedLines}
                      </Text>
                    )}
                  </Layout.Horizontal>
                }
                text={
                  diff.isDeleted ? diff.oldName : diff.isRename ? `${diff.oldName} -> ${diff.newName}` : diff.newName
                }
                onClick={() => onJumpToFile(diff)}
              />
            ))}
          </Menu>
        </Container>
      }
      tooltipProps={{ interactionKind: 'click', hasBackdrop: true, popoverClassName: css.popover }}>
      <StringSubstitute str={getString('pr.showLink')} vars={{ count: diffs?.length || '0' }} />
    </Button>
  )
}
