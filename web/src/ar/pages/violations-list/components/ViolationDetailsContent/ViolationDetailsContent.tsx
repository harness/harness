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

import React, { useEffect, useState } from 'react'
import { ButtonVariation, Container, Layout, Text, useToaster } from '@harnessio/uicore'
import { FontVariation } from '@harnessio/design-system'
import { Error, evaluateArtifactScan, useGetArtifactScansQuery, V3Error } from '@harnessio/react-har-service-client'

import { useAppStore, useParentComponents } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'
import { queryClient } from '@ar/utils/queryClient'
import PageContent from '@ar/components/PageContent/PageContent'
import { PermissionIdentifier, ResourceType } from '@ar/common/permissionTypes'

import ViolationDetails from './ViolationDetails'
import BasicInformationContent from './BasicInformationContent'

import css from './ViolationDetailsContent.module.scss'

interface ViolationDetailsContentProps {
  scanId: string
  onClose?: () => void
}

function ViolationDetailsContent(props: ViolationDetailsContentProps) {
  const [selectedPolicySet, setPolicySet] = useState<string | undefined>()
  const { scope } = useAppStore()
  const { RbacButton } = useParentComponents()
  const { clear, showSuccess, showError } = useToaster()
  const { getString } = useStrings()

  const {
    data,
    isFetching: loading,
    error,
    refetch
  } = useGetArtifactScansQuery({
    queryParams: {
      account_identifier: scope.accountId || '',
      project_identifier: scope.projectIdentifier,
      org_identifier: scope.orgIdentifier,
      scan_id: props.scanId,
      scope: 'descendants'
    }
  })

  const responseData = data?.content.data
  const scanInfo = responseData?.[0]

  useEffect(() => {
    if (!scanInfo) return
    if (selectedPolicySet) return
    const policySets = scanInfo.policySets
    if (policySets && policySets.length > 0) {
      setPolicySet(policySets[0].policySetRef)
    }
  }, [scanInfo, selectedPolicySet])

  const handleRescan = (scanId: string) => {
    return evaluateArtifactScan({
      queryParams: { account_identifier: scope.accountId || '' },
      body: { scanId }
    })
      .then(() => {
        clear()
        showSuccess(getString('versionList.messages.reEvaluateSuccess'))
      })
      .catch((err: V3Error) => {
        clear()
        showError(err?.error?.message ?? getString('versionList.messages.reEvaluateFailed'))
      })
      .finally(() => {
        queryClient.invalidateQueries(['GetArtifactScans'])
        props.onClose?.()
      })
  }

  return (
    <Container className={css.container}>
      <Container>
        <Layout.Vertical>
          <Layout.Horizontal data-testid="policy-evaluation-header" className={css.titleContainer} spacing="medium">
            <Text font={{ variation: FontVariation.H3 }}>
              {getString('violationsList.violationDetailsModal.title')}
            </Text>
          </Layout.Horizontal>
        </Layout.Vertical>
        <PageContent loading={loading} error={error?.error as Error} refetch={refetch}>
          {!!scanInfo && (
            <Layout.Vertical data-testid="policy-evaluation-body" className={css.contentContainer} spacing="medium">
              <BasicInformationContent
                data={scanInfo}
                selectedPolicySet={selectedPolicySet}
                onChangePolicySet={setPolicySet}
              />
              {selectedPolicySet && <ViolationDetails scanId={props.scanId} policySetRef={selectedPolicySet} />}
            </Layout.Vertical>
          )}
        </PageContent>
      </Container>
      <Layout.Horizontal className={css.footerContainer} data-testid="policy-evaluation-footer" spacing="medium">
        {!!responseData && (
          <RbacButton
            text={getString('versionList.actions.reEvaluate')}
            variation={ButtonVariation.SECONDARY}
            onClick={() => handleRescan(props.scanId)}
            permission={{
              permission: PermissionIdentifier.UPLOAD_ARTIFACT,
              resource: {
                resourceType: ResourceType.ARTIFACT_REGISTRY,
                resourceIdentifier: scanInfo?.registryName
              }
            }}
          />
        )}
      </Layout.Horizontal>
    </Container>
  )
}

export default ViolationDetailsContent
