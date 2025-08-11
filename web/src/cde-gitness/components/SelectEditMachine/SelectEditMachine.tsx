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

import React, { useEffect, useMemo } from 'react'
import { Container, Layout, Text } from '@harnessio/uicore'
import { Menu, MenuItem } from '@blueprintjs/core'
import { Cpu } from 'iconoir-react'
import { useFormikContext } from 'formik'
import { Color, FontVariation } from '@harnessio/design-system'
import { useParams } from 'react-router-dom'
import type { TypesInfraProviderResource } from 'services/cde'
import { useStrings } from 'framework/strings'
import { CDECustomDropdown } from 'cde-gitness/components/CDECustomDropdown/CDECustomDropdown'
import css from './SelectEditMachine.module.scss'

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
  isEditMode?: boolean
  isDisabled?: boolean
  loading?: boolean
}

interface resourceData {
  resource?: {
    identifier: string
    config_identifier: string
    name: string
    region: string
  }
}

export const SelectEditMachine = ({
  options,
  defaultValue,
  isEditMode = false,
  isDisabled = false,
  loading = false
}: SelectMachineInterface) => {
  const { getString } = useStrings()
  const { values, setFieldValue: onChange } = useFormikContext<resourceData>()
  const { resource: machine } = values
  const { gitspaceId = '' } = useParams<{ gitspaceId?: string }>()

  const isEmpty = !options || options.length === 0
  isDisabled = isDisabled || isEmpty || loading

  const machineTypes = useMemo(() => {
    return options.map(item => {
      const { cpu, disk, memory, identifier, name, space_path } = item
      return {
        identifier,
        label: name,
        cpu,
        disk,
        memory,
        space_path
      }
    })
  }, [options])

  useEffect(() => {
    if (defaultValue && !gitspaceId) {
      onChange('resource', {
        ...values.resource,
        identifier: defaultValue.identifier || ''
      })
    }
  }, [defaultValue?.identifier, gitspaceId])

  const data = (machineTypes?.find(item => item.identifier === machine?.identifier) || {}) as (typeof machineTypes)[0]

  return (
    <Container>
      <CDECustomDropdown
        overridePopOverWidth={isEditMode}
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
              {loading ? (
                <Text icon="loading" font={'normal'}></Text>
              ) : !isDisabled ? (
                <Text font={'normal'}>{data.label || getString('cde.machine')}</Text>
              ) : (
                <Text font={'normal'}>{getString('cde.update.allMachineDisabled')}</Text>
              )}
            </Layout.Vertical>
          </Layout.Horizontal>
        }
        menu={
          <Layout.Horizontal
            padding={{ top: 'small', bottom: 'small' }}
            className={isEditMode ? css.editModal : undefined}>
            <Menu>
              {machineTypes.length ? (
                <>
                  {machineTypes.map(item => {
                    return (
                      <MenuItem
                        key={item.identifier}
                        active={values.resource?.identifier === item.identifier}
                        text={
                          <Layout.Vertical>
                            <Text font={{ size: 'normal', weight: 'bold' }}>{item.label?.toUpperCase()}</Text>
                            <Layout.Horizontal spacing={'small'}>
                              <Text padding={'small'} className={css.tags} font={{ variation: FontVariation.SMALL }}>
                                {getString('cde.cpu')}: {item.cpu?.toUpperCase()}
                              </Text>
                              <Text padding={'small'} className={css.tags} font={{ variation: FontVariation.SMALL }}>
                                {getString('cde.memory')}: {item.memory?.toUpperCase()}
                              </Text>
                              <Text padding={'small'} className={css.tags} font={{ variation: FontVariation.SMALL }}>
                                {getString('cde.disk')}: {item.disk?.toUpperCase()}
                              </Text>
                            </Layout.Horizontal>
                          </Layout.Vertical>
                        }
                        onClick={() => {
                          onChange('resource', {
                            ...values.resource,
                            identifier: item.identifier || ''
                          })
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
