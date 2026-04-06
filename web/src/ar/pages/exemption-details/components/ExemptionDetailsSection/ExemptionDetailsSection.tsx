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
import { Container } from '@harnessio/uicore'
import type { FirewallExceptionResponseV3 } from '@harnessio/react-har-service-client'

import { useStrings } from '@ar/frameworks/strings'
import { DEFAULT_DATE_TIME_FORMAT } from '@ar/constants'
import { getReadableDateTime } from '@ar/common/dateUtils'

import { Label, Value } from './Components'
import css from './ExemptionDetailsSection.module.scss'

interface ExemptionDetailsSectionProps {
  data: FirewallExceptionResponseV3
}

function ExemptionDetailsSection({ data }: ExemptionDetailsSectionProps) {
  const { getString } = useStrings()
  return (
    <Container className={css.gridContainer}>
      {/* Requested At */}
      <Label>{getString('exemptionDetails.cards.section2.requestedDate')}</Label>
      <Value>{getReadableDateTime(data.createdAt, DEFAULT_DATE_TIME_FORMAT)}</Value>
      {/* Exemption Duration */}
      <Label>{getString('exemptionDetails.cards.section2.duration')}</Label>
      <Value>{getString('exemptionList.expireAfter', { days: data.expireAfter })}</Value>
      {/* Approval/rejection notes */}
      {data.notes && (
        <>
          <Label>{getString('exemptionDetails.cards.section2.notes')}</Label>
          <Value className={css.preWrap}>{data.notes}</Value>
        </>
      )}
      {/* Expiration Date */}
      {data.expirationAt && (
        <>
          <Label>{getString('exemptionDetails.cards.section2.expirationDate')}</Label>
          <Value>{getReadableDateTime(data.expirationAt, DEFAULT_DATE_TIME_FORMAT)}</Value>
        </>
      )}
      {/* Business Justification */}
      <Label>{getString('exemptionDetails.cards.section2.businessJustification')}</Label>
      <Value className={css.preWrap}>{data.businessJustification}</Value>
      {/* Remediation Plan */}
      <Label>{getString('exemptionDetails.cards.section2.remediationPlan')}</Label>
      <Value className={css.preWrap}>{data.remediationPlan}</Value>
    </Container>
  )
}

export default ExemptionDetailsSection
