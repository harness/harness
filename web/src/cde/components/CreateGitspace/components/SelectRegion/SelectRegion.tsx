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
import { Layout, SelectOption, Text } from '@harnessio/uicore'
import { Menu, MenuItem } from '@blueprintjs/core'
import { Map } from 'iconoir-react'
import { useFormikContext } from 'formik'
import { GitspaceSelect } from 'cde/components/GitspaceSelect/GitspaceSelect'
import { useStrings } from 'framework/strings'
import { GitspaceRegion } from 'cde/constants'
import USWest from './assests/USWest.png'
import USEast from './assests/USEast.png'
import Australia from './assests/Aus.png'
import Europe from './assests/Europe.png'
import type { GitspaceFormInterface } from '../../CreateGitspace'

interface SelectRegionInterface {
  options: SelectOption[]
}

export const SelectRegion = ({ options }: SelectRegionInterface) => {
  const { getString } = useStrings()
  const { values, errors, setFieldValue: onChange } = useFormikContext<GitspaceFormInterface>()
  const { region = '' } = values
  const [regionState, setRegionState] = useState<string>(region)

  return (
    <GitspaceSelect
      overridePopOverWidth
      text={
        <Layout.Horizontal spacing={'small'}>
          <Map />
          <Text font={'normal'}>{region || getString('cde.region')}</Text>
        </Layout.Horizontal>
      }
      formikName="region"
      errorMessage={errors.region}
      renderMenu={
        <Layout.Horizontal padding={{ top: 'small', bottom: 'small' }}>
          <Menu>
            {options.map(({ label }) => {
              return (
                <MenuItem
                  key={label}
                  active={label === regionState}
                  text={<Text font={{ size: 'normal', weight: 'bold' }}>{label}</Text>}
                  onClick={() => {
                    onChange('region', label)
                  }}
                  onMouseOver={(e: React.MouseEvent<HTMLAnchorElement, MouseEvent>) => {
                    setRegionState(e.currentTarget.innerText)
                  }}
                />
              )
            })}
          </Menu>
          <Menu>
            {regionState === GitspaceRegion.USEast && <img src={USEast} />}
            {regionState === GitspaceRegion.USWest && <img src={USWest} />}
            {regionState === GitspaceRegion.Europe && <img src={Europe} />}
            {regionState === GitspaceRegion.Australia && <img src={Australia} />}
          </Menu>
        </Layout.Horizontal>
      }
    />
  )
}
