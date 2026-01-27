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
import type { FormikProps } from 'formik'
import { Color, FontVariation } from '@harnessio/design-system'
import { CardSelect, CardSelectType, Checkbox, Container, Layout, Text } from '@harnessio/uicore'

import { useStrings } from '@ar/frameworks/strings'
import {
  UpstreamProxyConfigFirewallModeEnum,
  type UpstreamRegistryRequest
} from '@ar/pages/upstream-proxy-details/types'

import { AllowedModes } from './constants'

import css from './FormContent.module.scss'

interface DependencyFirewallConfigurationFormContentProps {
  formikProps: FormikProps<UpstreamRegistryRequest>
  isEdit: boolean
  disabled: boolean
}

function DependencyFirewallConfigurationFormContent(props: DependencyFirewallConfigurationFormContentProps) {
  const { getString } = useStrings()
  const { formikProps } = props
  const { values } = formikProps
  const selectedMode = values.config.firewallMode
  const enabled = [UpstreamProxyConfigFirewallModeEnum.WARN, UpstreamProxyConfigFirewallModeEnum.BLOCK].includes(
    selectedMode as UpstreamProxyConfigFirewallModeEnum
  )
  return (
    <Container>
      <Layout.Vertical spacing="medium">
        <Text font={{ variation: FontVariation.CARD_TITLE }}>
          {getString('repositoryDetails.repositoryForm.dependencyFirewallTitle')}
        </Text>
        <Checkbox
          label={getString('repositoryDetails.repositoryForm.enableDependencyFirewall')}
          disabled={props.disabled}
          onClick={e => {
            const isChecked = e.currentTarget.checked
            if (isChecked) {
              formikProps.setFieldValue('config.firewallMode', UpstreamProxyConfigFirewallModeEnum.WARN)
            } else {
              formikProps.setFieldValue('config.firewallMode', UpstreamProxyConfigFirewallModeEnum.ALLOW)
            }
          }}
        />
        {enabled && (
          <CardSelect
            id="card-select-dependency-firewall"
            className={css.cardSelect}
            cornerSelected
            data={[UpstreamProxyConfigFirewallModeEnum.WARN, UpstreamProxyConfigFirewallModeEnum.BLOCK]}
            type={CardSelectType.CardView}
            renderItem={item => {
              const option = AllowedModes[item]
              if (!option) return <></>
              return (
                <Layout.Vertical>
                  <Text font={{ variation: FontVariation.BODY, weight: 'bold' }}>{getString(option.label)}</Text>
                  <Text font={{ variation: FontVariation.SMALL }} color={Color.GREY_600}>
                    {getString(option.subTitle)}
                  </Text>
                </Layout.Vertical>
              )
            }}
            selected={selectedMode}
            onChange={item => {
              if (props.disabled) return
              if (item) {
                formikProps.setFieldValue('config.firewallMode', item)
              }
            }}
          />
        )}
      </Layout.Vertical>
    </Container>
  )
}

export default DependencyFirewallConfigurationFormContent
