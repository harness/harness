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
import { Layout } from '@harnessio/uicore'
import { Error, useGetArtifactScanDetailsQuery } from '@harnessio/react-har-service-client'

import { useAppStore } from '@ar/hooks'
import PageContent from '@ar/components/PageContent/PageContent'

import EvaluationInformationContent from './EvaluationInformationContent'
import FixInformationContent from './FixInformationContent'
import ViolationFailureDetails from './ViolationFailureDetails'

interface ViolationDetailsProps {
  scanId: string
  policySetRef: string
  onClose?: () => void
}

function ViolationDetails(props: ViolationDetailsProps) {
  const { scope } = useAppStore()
  const {
    data,
    isFetching: loading,
    error,
    refetch
  } = useGetArtifactScanDetailsQuery({
    queryParams: {
      account_identifier: scope.accountId || '',
      policy_set_ref: props.policySetRef
    },
    scan_id: props.scanId
  })

  const responseData = data?.content?.data

  return (
    <PageContent loading={loading} error={error?.error as Error} refetch={refetch}>
      {!!responseData && (
        <Layout.Vertical data-testid="violation-details-content" padding={{ top: 'medium' }}>
          <FixInformationContent data={responseData} />
          <ViolationFailureDetails data={responseData} />
          <EvaluationInformationContent data={responseData} />
        </Layout.Vertical>
      )}
    </PageContent>
  )
}

export default ViolationDetails
