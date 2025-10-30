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
import { useFormikContext } from 'formik'
import { Color } from '@harnessio/design-system'
import globe from 'cde-gitness/assests/globe.svg?url'
import { useStrings } from 'framework/strings'
import { GitspaceRegion } from 'cde-gitness/constants'
import type { OpenapiCreateGitspaceRequest, TypesInfraProviderResource } from 'services/cde'
import { CDECustomDropdown } from 'cde-gitness/components/CDECustomDropdown/CDECustomDropdown'
import USWest from './assests/USWest.png'
import USEast from './assests/USEast.png'
import Australia from './assests/Aus.png'
import Europe from './assests/Europe.png'
import Empty from './assests/Empty.png'
import css from './SelectRegion.module.scss'

interface SelectRegionInterface {
  isDisabled?: boolean
  defaultValue: { label: string; value: TypesInfraProviderResource[] }
  options: { label: string; value: TypesInfraProviderResource[] }[]
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

export const SelectRegion = ({ options, isDisabled = false, defaultValue }: SelectRegionInterface) => {
  const { getString } = useStrings()
  const {
    values: { metadata },
    setFieldValue: onChange
  } = useFormikContext<OpenapiCreateGitspaceRequest>()
  const [regionState, setRegionState] = useState<string | undefined>(metadata?.region)

  useEffect(() => {
    if ((!regionState || metadata?.region !== regionState) && !isDisabled) {
      setRegionState(defaultValue?.label?.toLowerCase())
      onChange('metadata.region', defaultValue?.label?.toLowerCase())
    }
  }, [defaultValue?.label?.toLowerCase()])

  const isNoRegionData = isDisabled && metadata?.infraProvider && options.length === 0

  return (
    <Container>
      <CDECustomDropdown
        isDisabled={isDisabled}
        label={
          <Layout.Horizontal spacing={'small'} flex={{ alignItems: 'center', justifyContent: 'flex-start' }}>
            <Layout.Vertical>
              <Text font={'normal'}>
                {isNoRegionData ? getString('cde.allRegionDisabled') : metadata?.region || getString('cde.region')}
              </Text>
            </Layout.Vertical>
          </Layout.Horizontal>
        }
        leftElement={
          <Layout.Horizontal>
            <img src={globe} className={css.icon} />
            <Layout.Vertical spacing="small">
              <Text color={Color.GREY_500} font={{ weight: 'bold' }}>
                {getString('cde.create.region')}
              </Text>
              <Text font="small"> {getString('cde.create.regionText')}</Text>
            </Layout.Vertical>
          </Layout.Horizontal>
        }
        menu={
          <Menu>
            {options.map(({ label }) => {
              return (
                <MenuItem
                  key={label}
                  active={label === metadata?.region?.toLowerCase()}
                  text={<Text font={{ size: 'normal', weight: 'bold' }}>{label.toUpperCase()}</Text>}
                  onClick={() => {
                    onChange('metadata.region', label.toLowerCase())
                    onChange('resource_identifier', undefined)
                    onChange('resource_space_ref', undefined)
                  }}
                  onMouseOver={(e: React.MouseEvent<HTMLAnchorElement, MouseEvent>) => {
                    setRegionState(e.currentTarget.innerText)
                  }}
                />
              )
            })}
          </Menu>
        }
      />
    </Container>
  )
}
