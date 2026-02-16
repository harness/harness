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
import classNames from 'classnames'
import { Collapse, Container, Layout } from '@harnessio/uicore'
import type { KVO } from '@harnessio/design-system'
import type {
  FixVersionDetails,
  LicensePolicyFailureDetailConfig,
  OssRiskLevelPolicyFailureDetailConfig,
  PackageAgeViolationPolicyFailureDetailConfig,
  PolicyFailureDetail,
  PolicySetFailureDetail,
  SecurityPolicyFailureDetailConfig
} from '@harnessio/react-har-service-client'

import { useStrings } from '@ar/frameworks/strings'
import { DEFAULT_DATE_TIME_FORMAT } from '@ar/constants'
import { getReadableDateTime } from '@ar/common/dateUtils'

import InformationMetrics from './InformationMetrics'
import useGetPolicyDetailsPageUrl from '../../hooks/useGetPolicyDetailsPageUrl'

import css from './ViolationDetailsContent.module.scss'

interface ViolationFailureDetailsItemProps {
  data: any
  policyName: string
  policyRef: string
  category: PolicyFailureDetail['category']
  fixVersionDetails?: FixVersionDetails
}

function SecurityPolicyFailureDetailsItem({
  data,
  fixVersionDetails
}: {
  data: SecurityPolicyFailureDetailConfig
  fixVersionDetails?: FixVersionDetails
}) {
  const { vulnerabilities } = data
  const { getString } = useStrings()
  return (
    <Layout.Vertical spacing="medium">
      <Container className={css.gridContainer}>
        <InformationMetrics.Text
          label={getString('violationsList.violationDetailsModal.fixInformationSection.currentVersion')}
          value={fixVersionDetails?.currentVersion?.toString() || getString('na')}
        />
        <InformationMetrics.Text
          label={getString('violationsList.violationDetailsModal.fixInformationSection.fixedVersion')}
          value={fixVersionDetails?.fixVersion?.toString() || getString('na')}
        />
      </Container>
      <Container className={css.securityViolationGridContainer}>
        <InformationMetrics.Label
          label={getString('violationsList.violationDetailsModal.violatedPoliciesSection.securityViolation.cveId')}
        />
        <InformationMetrics.Label
          label={getString('violationsList.violationDetailsModal.violatedPoliciesSection.securityViolation.cvssScore')}
        />
        <InformationMetrics.Label
          label={getString(
            'violationsList.violationDetailsModal.violatedPoliciesSection.securityViolation.cvssThreshold'
          )}
        />
        {vulnerabilities.map(each => (
          <Fragment key={each.cveId}>
            <InformationMetrics.Value value={each.cveId} />
            <InformationMetrics.Value value={each.cvssScore.toLocaleString()} />
            <InformationMetrics.Value value={each.cvssThreshold.toLocaleString()} />
          </Fragment>
        ))}
      </Container>
    </Layout.Vertical>
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
          'violationsList.violationDetailsModal.violatedPoliciesSection.licenseViolation.packageLicense'
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
        value={publishedOn ? getReadableDateTime(Number(publishedOn), DEFAULT_DATE_TIME_FORMAT) : getString('na')}
      />
    </Container>
  )
}

function OssRiskLevelViolationPolicyFailureDetailItem({ data }: { data: OssRiskLevelPolicyFailureDetailConfig }) {
  const { ossRiskLevel } = data
  const { getString } = useStrings()
  return (
    <Container className={css.gridContainer}>
      <InformationMetrics.Text
        label={getString(
          'violationsList.violationDetailsModal.violatedPoliciesSection.ossRiskLevelViolation.ossRiskLevel'
        )}
        value={ossRiskLevel}
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

function ViolationFailureDetailsItem(props: ViolationFailureDetailsItemProps) {
  const { data, policyName, policyRef, category, fixVersionDetails } = props
  const policyDetailsURL = useGetPolicyDetailsPageUrl(policyRef)
  const { getString } = useStrings()
  const renderContent = () => {
    switch (category) {
      case 'Security':
        return <SecurityPolicyFailureDetailsItem data={data} fixVersionDetails={fixVersionDetails} />
      case 'License':
        return <LicensePolicyFailureDetailItem data={data} />
      case 'PackageAge':
        return <PackageAgeViolationPolicyFailureDetailItem data={data} />
      case 'OssRiskLevel':
        return <OssRiskLevelViolationPolicyFailureDetailItem data={data} />
      default:
        return <GenericPolicyFailureDetailItem data={data} />
    }
  }
  return (
    <Collapse
      expandedIcon="chevron-down"
      collapsedIcon="chevron-right"
      collapseClassName={classNames(css.collapseMain, css.policyContainer)}
      heading={
        <Layout.Horizontal className={css.policyHeaderContainer}>
          <InformationMetrics.Link
            label={getString('violationsList.violationDetailsModal.violatedPoliciesSection.policyViolated')}
            value={policyName || policyRef}
            linkTo={policyDetailsURL}
          />
          <InformationMetrics.ScanCategory
            label={getString('violationsList.violationDetailsModal.violatedPoliciesSection.category')}
            category={category}
          />
        </Layout.Horizontal>
      }>
      <Container className={css.policyContentContainer} padding="small">
        {renderContent()}
      </Container>
    </Collapse>
  )
}

interface ViolationFailureDetailsProps {
  data: PolicySetFailureDetail
  fixVersionDetails?: FixVersionDetails
}

function ViolationFailureDetails(props: ViolationFailureDetailsProps) {
  const { data } = props
  const { policyFailureDetails } = data
  return (
    <Layout.Vertical spacing="large" padding="small">
      {policyFailureDetails?.map(item => (
        <ViolationFailureDetailsItem
          key={item.policyRef}
          policyName={item.policyName}
          policyRef={item.policyRef}
          category={item.category}
          data={item}
          fixVersionDetails={props.fixVersionDetails}
        />
      ))}
    </Layout.Vertical>
  )
}

export default ViolationFailureDetails
