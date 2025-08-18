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

import React, { useEffect } from 'react'
import { Container, Layout, Text } from '@harnessio/uicore'
import { Menu, MenuItem } from '@blueprintjs/core'
import { Cpu } from 'iconoir-react'
import { useFormikContext } from 'formik'
import { Color, FontVariation } from '@harnessio/design-system'
import { useParams } from 'react-router-dom'
import type { OpenapiCreateGitspaceRequest, TypesInfraProviderResource } from 'services/cde'
import { useStrings } from 'framework/strings'
import { CDECustomDropdown } from 'cde-gitness/components/CDECustomDropdown/CDECustomDropdown'
import css from './SelectMachine.module.scss'

export const machineIdToLabel = {
  '4core_8gb_32gb': 'Standard',
  '8core_16gb_64gb': 'Large'
}

export const labelToMachineId = {
  Standard: '4core_8gb_32gb',
  Large: '8core_16gb_64gb'
}

interface SelectMachineInterface {
  options: TypesInfraProviderResource[]
  defaultValue: TypesInfraProviderResource
  isDisabled?: boolean
}

export const SelectMachine = ({ options, defaultValue, isDisabled = false }: SelectMachineInterface) => {
  const { getString } = useStrings()
  const { values, setFieldValue: onChange } = useFormikContext<OpenapiCreateGitspaceRequest>()
  const { resource_identifier: machine } = values
  const { gitspaceId = '' } = useParams<{ gitspaceId?: string }>()

  const machineTypes = options.map(item => {
    const { cpu, disk, memory, identifier, name, space_path, metadata } = item
    return {
      identifier,
      label: name,
      cpu,
      disk,
      memory,
      space_path,
      disk_type: metadata?.persistent_disk_type
    }
  })

  useEffect(() => {
    if (defaultValue && !gitspaceId) {
      onChange('resource_identifier', defaultValue.identifier)
      onChange('resource_space_ref', defaultValue.space_path)
    }
  }, [defaultValue?.identifier, gitspaceId])

  const data = (machineTypes?.find(item => item.identifier === machine) || {}) as (typeof machineTypes)[0]

  const isNoMachineData = values?.metadata?.infraProvider && isDisabled && options.length === 0

  return (
    <Container>
      <CDECustomDropdown
        overridePopOverWidth
        isDisabled={isDisabled}
        leftElement={
          <Layout.Horizontal>
            <Cpu className={css.icon} />
            <Layout.Vertical spacing="small">
              <Text color={Color.GREY_500} font={{ weight: 'bold' }}>
                {getString('cde.create.machineType')}
              </Text>
              <Text font="small"> {getString('cde.create.machineTypeNote')}</Text>
            </Layout.Vertical>
          </Layout.Horizontal>
        }
        label={
          <Layout.Horizontal spacing={'small'} flex={{ alignItems: 'center', justifyContent: 'flex-start' }}>
            <Layout.Vertical>
              <Text font={'normal'}>
                {isNoMachineData ? getString('cde.update.allMachineDisabled') : data.label || getString('cde.machine')}
              </Text>
            </Layout.Vertical>
          </Layout.Horizontal>
        }
        menu={
          <Layout.Horizontal padding={{ top: 'small', bottom: 'small' }}>
            <Menu>
              {machineTypes.length ? (
                <>
                  {machineTypes.map(item => {
                    return (
                      <MenuItem
                        key={item.identifier}
                        active={values.resource_identifier === item.identifier}
                        text={
                          <Layout.Vertical>
                            <Text font={{ size: 'normal', weight: 'bold' }}>{item.label?.toUpperCase()}</Text>
                            <Layout.Horizontal spacing={'small'}>
                              <Text padding={'small'} className={css.tags} font={{ variation: FontVariation.SMALL }}>
                                {getString('cde.cpu')}: {item.cpu?.toUpperCase()}
                              </Text>
                              <Text padding={'small'} className={css.tags} font={{ variation: FontVariation.SMALL }}>
                                {getString('cde.memory')}: {item.memory?.toUpperCase()}GB
                              </Text>
                              <Text padding={'small'} className={css.tags} font={{ variation: FontVariation.SMALL }}>
                                {getString('cde.disk')}: {item.disk?.toUpperCase()}GB{' '}
                                {item.disk_type ? `(${item.disk_type})` : ''}
                              </Text>
                            </Layout.Horizontal>
                          </Layout.Vertical>
                        }
                        onClick={() => {
                          onChange('resource_identifier', item.identifier || '')
                          onChange('resource_space_ref', item.space_path || '')
                        }}
                      />
                    )
                  })}
                </>
              ) : (
                <>
                  <Text font={{ size: 'normal', weight: 'bold' }}>{getString('cde.regionSelectWarning')}</Text>
                </>
              )}
            </Menu>
          </Layout.Horizontal>
        }
      />
    </Container>
  )
}
