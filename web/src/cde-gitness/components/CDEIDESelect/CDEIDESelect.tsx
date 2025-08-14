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

import React, { useMemo } from 'react'
import { Container, Layout, Text } from '@harnessio/uicore'
import { Color } from '@harnessio/design-system'
import { Menu } from '@blueprintjs/core'
import { Code } from 'iconoir-react'
import { groupEnums } from 'cde-gitness/constants'
import { useStrings } from 'framework/strings'
import { CDECustomDropdown } from '../CDECustomDropdown/CDECustomDropdown'
import { CustomIDESection } from '../IDEDropdownSection/IDEDropdownSection'
import type { IDEOption } from '../../constants'
import css from './CDEIDESelect.module.scss'

interface CDEIDESelectProps {
  onChange: (field: string, value: IDEOption['value']) => void
  selectedIde?: string
  filteredIdeOptions?: IDEOption[]
  isEditMode?: boolean
}

export const CDEIDESelect = ({
  onChange,
  selectedIde,
  filteredIdeOptions = [],
  isEditMode = false
}: CDEIDESelectProps) => {
  const { getString } = useStrings()

  const selectedIDEOption = useMemo(() => {
    if (!selectedIde) {
      return undefined
    }

    const foundOption = filteredIdeOptions.find(item => item.value === selectedIde)
    return foundOption || (filteredIdeOptions.length > 0 ? filteredIdeOptions[0] : undefined)
  }, [selectedIde, filteredIdeOptions])

  const vscodeOptions = useMemo(
    () => filteredIdeOptions.filter(val => val.group === groupEnums.VSCODE),
    [filteredIdeOptions]
  )
  const jetbrainOptions = useMemo(
    () => filteredIdeOptions.filter(val => val.group === groupEnums.JETBRAIN),
    [filteredIdeOptions]
  )

  return (
    <CDECustomDropdown
      ideDropdown={true}
      overridePopOverWidth={isEditMode}
      isDisabled={filteredIdeOptions.length === 0}
      leftElement={
        <Layout.Horizontal>
          <Code className={css.icon} />
          <Layout.Vertical spacing="small">
            <Text color={Color.GREY_500} font={{ weight: 'bold' }}>
              IDE
            </Text>
            <Text font="small">Your Gitspace will open in the selected IDE to code</Text>
          </Layout.Vertical>
        </Layout.Horizontal>
      }
      label={
        filteredIdeOptions.length === 0 ? (
          <Layout.Horizontal width="100%" spacing="medium" flex={{ alignItems: 'center', justifyContent: 'start' }}>
            <Text>{getString('cde.create.ideEmpty')}</Text>
          </Layout.Horizontal>
        ) : (
          <Layout.Horizontal width="100%" spacing="medium" flex={{ alignItems: 'center', justifyContent: 'start' }}>
            <img height={16} width={16} src={selectedIDEOption?.icon} />
            <Text>{selectedIDEOption?.label}</Text>
          </Layout.Horizontal>
        )
      }
      menu={
        <Container className={isEditMode ? css.editModal : undefined}>
          <Menu>
            {vscodeOptions.length > 0 && (
              <CustomIDESection
                options={vscodeOptions}
                heading={getString('cde.ide.bymircosoft')}
                value={selectedIde}
                onChange={onChange}
              />
            )}
            {vscodeOptions.length > 0 && jetbrainOptions.length > 0 && <hr className={css.divider} />}
            {jetbrainOptions.length > 0 && (
              <CustomIDESection
                options={jetbrainOptions}
                heading={getString('cde.ide.byjetbrain')}
                value={selectedIde}
                onChange={onChange}
              />
            )}
          </Menu>
        </Container>
      }
    />
  )
}
