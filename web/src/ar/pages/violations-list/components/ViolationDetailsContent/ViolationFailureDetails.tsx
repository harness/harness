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
import { Layout, Text } from '@harnessio/uicore'
import { Color, FontVariation, type KVO } from '@harnessio/design-system'
import type { ArtifactScanDetails } from '@harnessio/react-har-service-client'

import { useStrings } from '@ar/frameworks/strings'
import InformationMetrics from './InformationMetrics'

interface ViolationFailureDetailsItemProps {
  data: KVO
}

function ViolationFailureDetailsItem({ data }: ViolationFailureDetailsItemProps) {
  const { name, ...rest } = data
  if (!name) return <></>
  return (
    <Layout.Vertical spacing="large">
      <Text font={{ variation: FontVariation.BODY, weight: 'bold' }} color={Color.PRIMARY_7}>
        {name}
      </Text>
      {Object.entries(rest).map(([key, value]) => {
        if (typeof value === 'string' || typeof value === 'number') {
          return <InformationMetrics.Text key={key} label={key} value={value.toString()} />
        }
        return <></>
      })}
    </Layout.Vertical>
  )
}

interface ViolationFailureDetailsProps {
  data: ArtifactScanDetails
}

function ViolationFailureDetails(props: ViolationFailureDetailsProps) {
  const { data } = props
  const { policyFailureDetails } = data
  const { getString } = useStrings()
  return (
    <Layout.Vertical spacing="large">
      <Text font={{ variation: FontVariation.H5, weight: 'bold' }} color={Color.GREY_700}>
        {getString('violationsList.violationDetailsModal.violatedPoliciesSection.title')}
      </Text>
      {policyFailureDetails?.map(item => (
        <ViolationFailureDetailsItem key={item.name} data={item} />
      ))}
    </Layout.Vertical>
  )
}

export default ViolationFailureDetails
