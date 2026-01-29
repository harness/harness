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
import { Link } from 'react-router-dom'
import type { IconProps } from '@harnessio/icons'
import { Layout, Text } from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'
import type { ArtifactScan } from '@harnessio/react-har-service-client'

import ScanBadge from '@ar/components/Badge/ScanBadge'

interface TextMetricProps {
  label: string
  labelIconProps?: IconProps
  value: string
  valueIconProps?: IconProps
}
function TextMetric(props: TextMetricProps) {
  return (
    <Layout.Vertical spacing="xsmall">
      <Text
        font={{ variation: FontVariation.BODY }}
        color={Color.GREY_700}
        icon={props.labelIconProps?.name}
        iconProps={props.labelIconProps}>
        {props.label}
      </Text>
      <Text
        font={{ variation: FontVariation.BODY }}
        color={Color.GREY_800}
        icon={props.valueIconProps?.name}
        iconProps={props.valueIconProps}>
        {props.value}
      </Text>
    </Layout.Vertical>
  )
}

interface LinkMetricProps extends TextMetricProps {
  linkTo: string
}

function LinkMetric(props: LinkMetricProps) {
  return (
    <Layout.Vertical spacing="xsmall">
      <Text
        font={{ variation: FontVariation.BODY }}
        color={Color.GREY_700}
        icon={props.labelIconProps?.name}
        iconProps={props.labelIconProps}>
        {props.label}
      </Text>
      <Link to={props.linkTo}>
        <Text
          font={{ variation: FontVariation.BODY }}
          color={Color.PRIMARY_7}
          icon={props.valueIconProps?.name}
          iconProps={props.valueIconProps}>
          {props.value}
        </Text>
      </Link>
    </Layout.Vertical>
  )
}

interface StatusMetricProps {
  label: string
  labelIconProps?: IconProps
  status: ArtifactScan['scanStatus']
  scanId: string
}

function ScanStatusMetric(props: StatusMetricProps) {
  return (
    <Layout.Vertical spacing="xsmall">
      <Text
        font={{ variation: FontVariation.BODY }}
        color={Color.GREY_700}
        icon={props.labelIconProps?.name}
        iconProps={props.labelIconProps}>
        {props.label}
      </Text>
      <ScanBadge scanId={props.scanId} status={props.status} />
    </Layout.Vertical>
  )
}

const InformationMetrics = {
  Text: TextMetric,
  Link: LinkMetric,
  ScanStatus: ScanStatusMetric
}

export default InformationMetrics
