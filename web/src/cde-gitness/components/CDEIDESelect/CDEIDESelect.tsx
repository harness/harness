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
import { Layout, Text } from '@harnessio/uicore'
import { Color } from '@harnessio/design-system'
import { Menu } from '@blueprintjs/core'
import { Code } from 'iconoir-react'
import { getIDETypeOptions, groupEnums } from 'cde-gitness/constants'
import { useStrings } from 'framework/strings'
import { CDECustomDropdown } from '../CDECustomDropdown/CDECustomDropdown'
import { CustomIDESection } from '../IDEDropdownSection/IDEDropdownSection'
import type { IDEOption } from '../../constants'
import css from './CDEIDESelect.module.scss'

interface CDEIDESelectProps {
  onChange: (field: string, value: IDEOption['value']) => void
  selectedIde?: string
  filteredIdeOptions?: IDEOption[]
}

export const CDEIDESelect = ({ onChange, selectedIde, filteredIdeOptions = [] }: CDEIDESelectProps) => {
  const { getString } = useStrings()
  const ideOptions = getIDETypeOptions(getString) ?? []

  const selectedIDEOption = selectedIde ? ideOptions.find(item => item.value === selectedIde) : undefined

  return (
    <CDECustomDropdown
      ideDropdown={true}
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
        <Menu>
          <CustomIDESection
            options={filteredIdeOptions.filter(val => val.group === groupEnums.VSCODE)}
            heading={getString('cde.ide.bymircosoft')}
            value={selectedIde}
            onChange={onChange}
          />
          <hr className={css.divider} />
          <CustomIDESection
            options={filteredIdeOptions.filter(val => val.group === groupEnums.JETBRAIN)}
            heading={getString('cde.ide.byjetbrain')}
            value={selectedIde}
            onChange={onChange}
          />
        </Menu>
      }
    />
  )
}
