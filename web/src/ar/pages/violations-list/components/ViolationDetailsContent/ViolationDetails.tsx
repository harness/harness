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
import { Collapse, Layout, Text } from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'
import type { ArtifactScanDetailsV3, PolicySetFailureDetailV3 } from '@harnessio/react-har-service-client'

import { useStrings } from '@ar/frameworks/strings'
import { Separator } from '@ar/components/Separator/Separator'

import InformationMetrics from './InformationMetrics'
import ViolationFailureDetails from './ViolationFailureDetails'
import useGetPolicySetDetailsPageUrl from '../../hooks/useGetPolicyDetailsPageUrl'

import css from './ViolationDetailsContent.module.scss'

interface PolicySetCollapseTitleProps {
  data: PolicySetFailureDetailV3
}
function PolicySetCollapseTitle({ data }: PolicySetCollapseTitleProps) {
  const { getString } = useStrings()
  const policySetURL = useGetPolicySetDetailsPageUrl(data.policySetRef)
  return (
    <Layout.Horizontal className={css.collapseHeaderContainer}>
      <InformationMetrics.Link
        label={getString('violationsList.violationDetailsModal.violatedPoliciesSection.policySetViolated')}
        value={data.policySetName || data.policySetRef}
        linkTo={policySetURL}
      />
      <InformationMetrics.Text
        label={getString('violationsList.violationDetailsModal.violatedPoliciesSection.policiesViolated')}
        value={data.policyFailureDetails.length.toString()}
      />
    </Layout.Horizontal>
  )
}

interface ViolationDetailsProps {
  data: ArtifactScanDetailsV3
}

function ViolationDetails(props: ViolationDetailsProps) {
  const { getString } = useStrings()

  return (
    <Layout.Vertical data-testid="violation-details-content" padding={{ top: 'medium' }}>
      <Text font={{ variation: FontVariation.H5, weight: 'bold' }} color={Color.GREY_700}>
        {getString('violationsList.violationDetailsModal.violatedPoliciesSection.title')}
      </Text>
      {props.data.policySetFailureDetails.map(policySet => (
        <>
          <Collapse
            key={policySet.policySetRef}
            expandedIcon="chevron-down"
            collapsedIcon="chevron-right"
            collapseClassName={css.collapseMain}
            isOpen
            heading={<PolicySetCollapseTitle data={policySet} />}>
            <ViolationFailureDetails data={policySet} fixVersionDetails={props.data.fixVersionDetails} />
          </Collapse>
          <Separator />
        </>
      ))}
    </Layout.Vertical>
  )
}

export default ViolationDetails
