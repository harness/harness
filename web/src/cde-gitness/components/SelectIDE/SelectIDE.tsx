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
import { Menu, MenuItem } from '@blueprintjs/core'
import { Layout, Text, Container } from '@harnessio/uicore'
import { useFormikContext } from 'formik'
import { GitspaceSelect } from 'cde-gitness/components/GitspaceSelect/GitspaceSelect'
import { useStrings, type UseStringsReturn } from 'framework/strings'
import type { OpenapiCreateGitspaceRequest } from 'services/cde'
import { IDEType } from 'cde-gitness/constants'
import vsCodeWebIcon from 'cde-gitness/assests/vsCodeWeb.svg?url'
import VSCode from 'cde-gitness/assests/VSCode.svg?url'

export const getIDESelectItems = (getString: UseStringsReturn['getString']) => {
  return [
    { label: getString('cde.ide.desktop'), value: IDEType.VSCODE },
    { label: getString('cde.ide.browser'), value: IDEType.VSCODEWEB }
  ]
}

export const SelectIDE = () => {
  const { values, errors, setFieldValue: onChange } = useFormikContext<OpenapiCreateGitspaceRequest>()
  const { ide } = values
  const { getString } = useStrings()
  const IDESelectItems = getIDESelectItems(getString)
  const IDELabel = IDESelectItems.find(item => item.value === ide)?.label

  return (
    <GitspaceSelect
      text={
        <Layout.Horizontal flex={{ alignItems: 'center', justifyContent: 'flex-start' }}>
          <img
            src={ide === IDEType.VSCODE ? VSCode : vsCodeWebIcon}
            height={20}
            width={20}
            style={{ marginRight: '12px' }}
          />
          <Layout.Vertical>
            <Text font={ide ? 'small' : 'normal'}>
              {ide ? getString('cde.ide.title') : getString('cde.ide.selectIDE')}
            </Text>
            {ide && (
              <Text font={{ size: 'normal', weight: 'bold' }}>{`${IDELabel}` || getString('cde.ide.title')}</Text>
            )}
          </Layout.Vertical>
        </Layout.Horizontal>
      }
      formikName="ide"
      errorMessage={errors.ide}
      renderMenu={
        <Container padding={{ top: 'small', bottom: 'small' }}>
          <Menu>
            {IDESelectItems.map(({ label, value }) => {
              return (
                <MenuItem
                  key={label}
                  active={value === ide}
                  text={
                    <Layout.Vertical spacing="small">
                      <Text font={{ size: 'normal', weight: 'bold' }}>{label}</Text>
                      <Text font="small">1.81.0</Text>
                    </Layout.Vertical>
                  }
                  onClick={() => {
                    onChange('ide', value)
                  }}
                />
              )
            })}
          </Menu>
        </Container>
      }
    />
  )
}
