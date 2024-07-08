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

import React, { useEffect, useState } from 'react'
import { Container, Layout, Text } from '@harnessio/uicore'
import { Menu, MenuItem } from '@blueprintjs/core'
import { Map } from 'iconoir-react'
import { useFormikContext } from 'formik'
import { GitspaceSelect } from 'cde/components/GitspaceSelect/GitspaceSelect'
import { useStrings } from 'framework/strings'
import { GitspaceRegion } from 'cde/constants'
import type { OpenapiCreateGitspaceRequest, TypesInfraProviderResourceResponse } from 'services/cde'
import USWest from './assests/USWest.png'
import USEast from './assests/USEast.png'
import Australia from './assests/Aus.png'
import Europe from './assests/Europe.png'
import Empty from './assests/Empty.png'

interface SelectRegionInterface {
  disabled?: boolean
  defaultValue: { label: string; value: TypesInfraProviderResourceResponse[] }
  options: { label: string; value: TypesInfraProviderResourceResponse[] }[]
}

export const getMapFromRegion = (region: string) => {
  switch (region) {
    case GitspaceRegion.USEast:
      return USEast
    case GitspaceRegion.USWest:
      return USWest
    case GitspaceRegion.Europe:
      return Europe
    case GitspaceRegion.Australia:
      return Australia
    default:
      return Empty
  }
}

export const SelectRegion = ({ options, disabled, defaultValue }: SelectRegionInterface) => {
  const { getString } = useStrings()
  const {
    values: { metadata },
    errors,
    setFieldValue: onChange
  } = useFormikContext<OpenapiCreateGitspaceRequest>()
  const [regionState, setRegionState] = useState<string | undefined>(metadata?.region)

  useEffect(() => {
    if (!regionState && !disabled) {
      setRegionState(defaultValue?.label?.toLowerCase())
      onChange('metadata.region', defaultValue?.label?.toLowerCase())
    }
  }, [defaultValue?.label?.toLowerCase()])

  return (
    <Container width={'50%'}>
      <GitspaceSelect
        disabled={disabled}
        overridePopOverWidth
        text={
          <Layout.Horizontal spacing={'small'} flex={{ alignItems: 'center', justifyContent: 'flex-start' }}>
            <Map height={20} width={20} style={{ marginRight: '12px', alignItems: 'center' }} />
            <Layout.Vertical>
              <Text font={'normal'}>{getString('cde.region')}</Text>
              <Text font={'normal'}>{metadata?.region || getString('cde.region')}</Text>
            </Layout.Vertical>
          </Layout.Horizontal>
        }
        formikName="metadata.region"
        errorMessage={
          (
            errors['metadata'] as unknown as {
              [key: string]: string
            }
          )?.region as unknown as string
        }
        renderMenu={
          <Layout.Horizontal padding={{ top: 'small', bottom: 'small' }}>
            <Menu>
              {options.map(({ label }) => {
                return (
                  <MenuItem
                    key={label}
                    active={label === regionState?.toLowerCase()}
                    text={<Text font={{ size: 'normal', weight: 'bold' }}>{label.toUpperCase()}</Text>}
                    onClick={() => {
                      onChange('metadata.region', label.toLowerCase())
                      onChange('infra_provider_resource_id', undefined)
                    }}
                    onMouseOver={(e: React.MouseEvent<HTMLAnchorElement, MouseEvent>) => {
                      setRegionState(e.currentTarget.innerText)
                    }}
                  />
                )
              })}
            </Menu>
            <Menu>
              <img src={getMapFromRegion(regionState?.toLowerCase() || '')} />
            </Menu>
          </Layout.Horizontal>
        }
      />
    </Container>
  )
}
