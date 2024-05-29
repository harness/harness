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
import { Layout, Text } from '@harnessio/uicore'
import { Menu, MenuItem } from '@blueprintjs/core'
import { Cpu } from 'iconoir-react'
import { useFormikContext } from 'formik'
import { GitspaceSelect } from 'cde/components/GitspaceSelect/GitspaceSelect'
import type { TypesInfraProviderResourceResponse } from 'services/cde'
import { useStrings } from 'framework/strings'
import RAM8 from './assests/RAM8.svg?url'
import RAM16 from './assests/RAM16.svg?url'
import Storage32 from './assests/Storage32.svg?url'
import Storage64 from './assests/Storage64.svg?url'
import CPU4Cores from './assests/CPU4Cores.svg?url'
import CPU8Cores from './assests/CPU8Cores.svg?url'
import type { GitspaceFormInterface } from '../../CreateGitspace'

export const machineIdToLabel = {
  '4core_8gb_32gb': 'Standard',
  '8core_16gb_64gb': 'Large'
}

export const labelToMachineId = {
  Standard: '4core_8gb_32gb',
  Large: '8core_16gb_64gb'
}

interface SelectMachineInterface {
  options: TypesInfraProviderResourceResponse[]
}

export const SelectMachine = ({ options }: SelectMachineInterface) => {
  const { getString } = useStrings()
  const { values, errors, setFieldValue: onChange } = useFormikContext<GitspaceFormInterface>()
  const { infra_provider_resource_id: machine } = values
  const [machineState, setMachineState] = useState<string>(machine || '')

  const machineTypes = options.map(item => {
    const { cpu, disk, memory, infra_provider_config_id } = item
    return {
      infra_provider_config_id,
      id: `${cpu}_${disk}_${memory}`,
      label: machineIdToLabel[`${cpu}_${disk}_${memory}` as keyof typeof machineIdToLabel]
    }
  })

  return (
    <GitspaceSelect
      overridePopOverWidth
      text={
        <Layout.Horizontal spacing={'small'}>
          <Cpu />
          <Text font={'normal'}>{machine || getString('cde.machine')}</Text>
        </Layout.Horizontal>
      }
      errorMessage={errors.infra_provider_resource_id}
      formikName="infra_provider_resource_id"
      renderMenu={
        <Layout.Horizontal padding={{ top: 'small', bottom: 'small' }}>
          <Menu>
            {machineTypes.map(item => {
              return (
                <MenuItem
                  key={item.infra_provider_config_id}
                  active={machineState === item.id}
                  text={
                    <Text font={{ size: 'normal', weight: 'bold' }}>
                      {machineIdToLabel[item.id as keyof typeof machineIdToLabel]}
                    </Text>
                  }
                  onClick={() => {
                    onChange('infra_provider_resource_id', item.infra_provider_config_id || '')
                  }}
                  onMouseOver={(e: React.MouseEvent<HTMLAnchorElement, MouseEvent>) => {
                    const dd = e.currentTarget.innerText as keyof typeof labelToMachineId
                    setMachineState(labelToMachineId[dd])
                  }}
                />
              )
            })}
          </Menu>
          <Menu>
            {machineState === labelToMachineId.Standard && (
              <Layout.Vertical>
                <Layout.Horizontal>
                  <img src={CPU4Cores} />
                  <img src={RAM8} />
                </Layout.Horizontal>
                <img src={Storage32} />
              </Layout.Vertical>
            )}
            {machineState === labelToMachineId.Large && (
              <Layout.Vertical>
                <Layout.Horizontal>
                  <img src={CPU8Cores} />
                  <img src={RAM16} />
                </Layout.Horizontal>
                <img src={Storage64} />
              </Layout.Vertical>
            )}
          </Menu>
        </Layout.Horizontal>
      }
    />
  )
}
