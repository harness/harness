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
import { FontVariation } from '@harnessio/design-system'
import { useParams } from 'react-router-dom'
import { GitspaceSelect } from 'cde/components/GitspaceSelect/GitspaceSelect'
import type { OpenapiCreateGitspaceRequest, TypesInfraProviderResourceResponse } from 'services/cde'
import { useStrings } from 'framework/strings'
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
  options: TypesInfraProviderResourceResponse[]
  defaultValue: TypesInfraProviderResourceResponse
}

export const SelectMachine = ({ options, defaultValue }: SelectMachineInterface) => {
  const { getString } = useStrings()
  const { values, errors, setFieldValue: onChange } = useFormikContext<OpenapiCreateGitspaceRequest>()
  const { infra_provider_resource_id: machine } = values
  const { gitspaceId = '' } = useParams<{ gitspaceId?: string }>()

  const machineTypes = options.map(item => {
    const { cpu, disk, memory, id, name } = item
    return {
      id,
      label: name,
      cpu,
      disk,
      memory
    }
  })

  useEffect(() => {
    if (defaultValue && !gitspaceId) {
      onChange('infra_provider_resource_id', defaultValue.id)
    }
  }, [defaultValue?.id, gitspaceId])

  const data = (machineTypes?.find(item => item.id === machine) || {}) as (typeof machineTypes)[0]

  return (
    <Container width={'50%'}>
      <GitspaceSelect
        overridePopOverWidth
        text={
          <Layout.Horizontal spacing={'small'} flex={{ alignItems: 'center', justifyContent: 'flex-start' }}>
            <Cpu height={20} width={20} style={{ marginRight: '12px', alignItems: 'center' }} />
            <Layout.Vertical>
              <Text font={'normal'}>{getString('cde.machine')}</Text>
              <Text font={'normal'}>{data.label || getString('cde.machine')}</Text>
            </Layout.Vertical>
          </Layout.Horizontal>
        }
        errorMessage={errors.infra_provider_resource_id}
        formikName="infra_provider_resource_id"
        renderMenu={
          <Layout.Horizontal padding={{ top: 'small', bottom: 'small' }}>
            <Menu>
              {machineTypes.length ? (
                <>
                  {machineTypes.map(item => {
                    return (
                      <MenuItem
                        key={item.id}
                        active={values.infra_provider_resource_id === item.id}
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
                          onChange('infra_provider_resource_id', item.id || '')
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
