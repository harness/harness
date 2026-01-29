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

import React, { Fragment } from 'react'
import { Container, Layout, Text } from '@harnessio/uicore'
import { Color, FontVariation, type KVO } from '@harnessio/design-system'
import type {
  ArtifactScanDetails,
  LicensePolicyFailureDetailConfig,
  PackageAgeViolationPolicyFailureDetailConfig,
  PolicyFailureDetail,
  SecurityPolicyFailureDetailConfig
} from '@harnessio/react-har-service-client'

import { useStrings } from '@ar/frameworks/strings'
import InformationMetrics from './InformationMetrics'
import css from './ViolationDetailsContent.module.scss'

interface ViolationFailureDetailsItemProps {
  data: any
}

function SecurityPolicyFailureDetailsItem({ data }: { data: SecurityPolicyFailureDetailConfig }) {
  const { vulnerabilities } = data
  const { getString } = useStrings()
  return (
    <Container className={css.securityViolationGridContainer}>
      <Text font={{ variation: FontVariation.BODY, weight: 'semi-bold' }} color={Color.GREY_700}>
        {getString('violationsList.violationDetailsModal.violatedPoliciesSection.securityViolation.cveId')}
      </Text>
      <Text font={{ variation: FontVariation.BODY, weight: 'semi-bold' }} color={Color.GREY_700}>
        {getString('violationsList.violationDetailsModal.violatedPoliciesSection.securityViolation.cvssScore')}
      </Text>
      <Text font={{ variation: FontVariation.BODY, weight: 'semi-bold' }} color={Color.GREY_700}>
        {getString('violationsList.violationDetailsModal.violatedPoliciesSection.securityViolation.cvssThreshold')}
      </Text>
      {vulnerabilities.map(each => (
        <Fragment key={each.cveId}>
          <Text lineClamp={1} font={{ variation: FontVariation.BODY }} color={Color.GREY_700}>
            {each.cveId}
          </Text>
          <Text lineClamp={1} font={{ variation: FontVariation.BODY }} color={Color.GREY_700}>
            {each.cvssScore}
          </Text>
          <Text lineClamp={1} font={{ variation: FontVariation.BODY }} color={Color.GREY_700}>
            {each.cvssThreshold}
          </Text>
        </Fragment>
      ))}
    </Container>
  )
}

function LicensePolicyFailureDetailItem({ data }: { data: LicensePolicyFailureDetailConfig }) {
  const { allowedLicenses, blockedLicense } = data
  const { getString } = useStrings()
  return (
    <Container className={css.gridContainer}>
      <InformationMetrics.Text
        label={getString(
          'violationsList.violationDetailsModal.violatedPoliciesSection.licenseViolation.allowedLicenses'
        )}
        value={allowedLicenses.join(', ')}
      />
      <InformationMetrics.Text
        label={getString(
          'violationsList.violationDetailsModal.violatedPoliciesSection.licenseViolation.blockedLicense'
        )}
        value={blockedLicense.toString()}
      />
    </Container>
  )
}

function PackageAgeViolationPolicyFailureDetailItem({ data }: { data: PackageAgeViolationPolicyFailureDetailConfig }) {
  const { packageAgeThreshold, publishedOn } = data
  const { getString } = useStrings()
  return (
    <Container className={css.gridContainer}>
      <InformationMetrics.Text
        label={getString(
          'violationsList.violationDetailsModal.violatedPoliciesSection.packageAgeViolation.packageAgeThreshold'
        )}
        value={packageAgeThreshold.toString()}
      />
      <InformationMetrics.Text
        label={getString(
          'violationsList.violationDetailsModal.violatedPoliciesSection.packageAgeViolation.publishedOn'
        )}
        value={publishedOn.toString()}
      />
    </Container>
  )
}

function GenericPolicyFailureDetailItem({ data }: { data: KVO }) {
  const { name, category, ...rest } = data
  return (
    <Container className={css.gridContainer}>
      {Object.entries(rest).map(([key, value]) => {
        if (typeof value === 'string' || typeof value === 'number') {
          return <InformationMetrics.Text key={key} label={key} value={value.toString()} />
        }
        return <></>
      })}
    </Container>
  )
}

function ViolationFailureDetailsItem({ data }: ViolationFailureDetailsItemProps) {
  const { policyName, category } = data
  const { getString } = useStrings()
  const renderContent = () => {
    switch (data.category as PolicyFailureDetail['category']) {
      case 'Security':
        return <SecurityPolicyFailureDetailsItem data={data} />
      case 'License':
        return <LicensePolicyFailureDetailItem data={data} />
      case 'PackageAge':
        return <PackageAgeViolationPolicyFailureDetailItem data={data} />
      default:
        return <GenericPolicyFailureDetailItem data={data} />
    }
  }
  if (!policyName) return <></>
  return (
    <Layout.Vertical spacing="large">
      <Text
        margin={{ bottom: 'medium' }}
        font={{ variation: FontVariation.BODY, weight: 'bold' }}
        color={Color.PRIMARY_7}>
        {policyName}
      </Text>
      <InformationMetrics.Text
        label={getString('violationsList.violationDetailsModal.violatedPoliciesSection.category')}
        value={category}
      />
      {renderContent()}
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
        <ViolationFailureDetailsItem key={item.policyName} data={item} />
      ))}
    </Layout.Vertical>
  )
}

export default ViolationFailureDetails
