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
import { Container, Layout, Page, Text } from '@harnessio/uicore'
import { FontVariation } from '@harnessio/design-system'
import { useGetArtifactScanDetailsQuery } from '@harnessio/react-har-service-client'

import { useAppStore } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'

import BasicInformationContent from './BasicInformationContent'
import EvaluationInformationContent from './EvaluationInformationContent'
import FixInformationContent from './FixInformationContent'
import ViolationFailureDetails from './ViolationFailureDetails'

import css from './ViolationDetailsContent.module.scss'

interface ViolationDetailsContentProps {
  scanId: string
}

function ViolationDetailsContent(props: ViolationDetailsContentProps) {
  const { scope } = useAppStore()
  const { getString } = useStrings()
  const {
    data,
    isFetching: loading,
    error,
    refetch
  } = useGetArtifactScanDetailsQuery({
    queryParams: {
      account_identifier: scope.accountId || ''
    },
    scan_id: props.scanId
  })

  const responseData = data?.content?.data

  return (
    <Container>
      <Layout.Vertical>
        <Layout.Horizontal data-testid="setup-client-header" className={css.titleContainer} spacing="medium">
          <Text font={{ variation: FontVariation.H3 }}>{getString('violationsList.violationDetailsModal.title')}</Text>
        </Layout.Horizontal>
      </Layout.Vertical>
      <Page.Body
        className={css.pageBody}
        loading={loading}
        error={error?.error?.message}
        retryOnError={() => refetch()}>
        {!!responseData && (
          <Layout.Vertical data-testid="setup-client-body" className={css.contentContainer} spacing="medium">
            <BasicInformationContent data={responseData} />
            <FixInformationContent data={responseData} />
            <ViolationFailureDetails data={responseData} />
            <EvaluationInformationContent data={responseData} />
          </Layout.Vertical>
        )}
      </Page.Body>
    </Container>
  )
}

export default ViolationDetailsContent
