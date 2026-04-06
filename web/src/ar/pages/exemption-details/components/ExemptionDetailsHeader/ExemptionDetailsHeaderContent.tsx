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
import { Expander } from '@blueprintjs/core'
import { useHistory } from 'react-router-dom'
import { Color } from '@harnessio/design-system'
import { ButtonVariation, Container, Layout, Text } from '@harnessio/uicore'
import type { FirewallExceptionResponseV3 } from '@harnessio/react-har-service-client'

import { useParentComponents, useRoutes } from '@ar/hooks'
import { queryClient } from '@ar/utils/queryClient'
import { useStrings } from '@ar/frameworks/strings'
import HeaderTitle from '@ar/components/Header/Title'
import { DEFAULT_DATE_TIME_FORMAT } from '@ar/constants'
import { getReadableDateTime } from '@ar/common/dateUtils'
import type { RepositoryPackageType } from '@ar/common/types'
import RepositoryIcon from '@ar/frameworks/RepositoryStep/RepositoryIcon'
import ExemptionStatusBadge from '@ar/components/Badge/ExemptionStatusBadge'
import { PermissionIdentifier, ResourceType } from '@ar/common/permissionTypes'
import ExemptionActions from '@ar/pages/exemption-list/components/ExemptionActions/ExemptionActions'

import useApproveExemption from '../../hooks/useApproveExemption'
import useRejectExemption from '../../hooks/useRejectExemption'

import css from './ExemptionDetailsHeader.module.scss'

interface ExemptionDetailsHeaderContentProps {
  data: FirewallExceptionResponseV3
  iconSize?: number
}

export default function ExemptionDetailsHeaderContent(props: ExemptionDetailsHeaderContentProps): JSX.Element {
  const { data, iconSize = 40 } = props
  const { getString } = useStrings()
  const history = useHistory()
  const routes = useRoutes()
  const { RbacButton } = useParentComponents()

  const handleAfterStatusChange = () => {
    queryClient.invalidateQueries(['ListFirewallExceptionsV3'])
    history.push(routes.toARDependencyFirewallExceptions())
  }

  const { triggerApprove } = useApproveExemption({ exemptionId: data.exceptionId, onSuccess: handleAfterStatusChange })
  const { triggerReject } = useRejectExemption({ exemptionId: data.exceptionId, onSuccess: handleAfterStatusChange })

  const showActions = data.status === 'PENDING'

  return (
    <Container>
      <Layout.Horizontal data-testid="exemption-header-container" spacing="medium" flex={{ alignItems: 'center' }}>
        <RepositoryIcon packageType={data.packageType as RepositoryPackageType} iconProps={{ size: iconSize }} />
        <Layout.Vertical spacing="small" className={css.nameContainer}>
          <Layout.Horizontal spacing="small" flex={{ alignItems: 'center', justifyContent: 'flex-start' }}>
            <HeaderTitle data-testid="registry-title">{data.packageName}</HeaderTitle>
            <ExemptionStatusBadge status={data.status} helperText={data.notes || ''} />
          </Layout.Horizontal>
          <Text data-testid="exemption-id" font={{ size: 'small' }} color={Color.GREY_500} width={800} lineClamp={1}>
            {getString('exemptionDetails.exemptionId', { exemptionId: data.exceptionId })}
          </Text>
          <Layout.Horizontal spacing="small">
            <Text font={{ size: 'small', weight: 'semi-bold' }} color={Color.BLACK} margin={{ right: 'small' }}>
              {getString('lastUpdated')}:
            </Text>
            <Text data-testid="registry-last-modified-at" font={{ size: 'small' }}>
              {data.updatedAt ? getReadableDateTime(data.updatedAt, DEFAULT_DATE_TIME_FORMAT) : getString('na')}
            </Text>
          </Layout.Horizontal>
        </Layout.Vertical>
        <Expander />
        <Layout.Horizontal spacing="small">
          {showActions && (
            <>
              <RbacButton
                icon="main-tick"
                text={getString('exemptionDetails.actions.approve')}
                variation={ButtonVariation.PRIMARY}
                intent="success"
                onClick={triggerApprove}
                permission={{
                  permission: PermissionIdentifier.ARTIFACT_FIREWALL_EXCEPTIONS_APPROVE,
                  resource: {
                    resourceType: ResourceType.ARTIFACT_FIREWALL_EXCEPTIONS
                  }
                }}
              />
              <RbacButton
                icon="cross"
                text={getString('exemptionDetails.actions.reject')}
                variation={ButtonVariation.PRIMARY}
                intent="danger"
                onClick={triggerReject}
                permission={{
                  permission: PermissionIdentifier.ARTIFACT_FIREWALL_EXCEPTIONS_APPROVE,
                  resource: {
                    resourceType: ResourceType.ARTIFACT_FIREWALL_EXCEPTIONS
                  }
                }}
              />
            </>
          )}
          <ExemptionActions data={data} exemptionId={data.exceptionId} />
        </Layout.Horizontal>
      </Layout.Horizontal>
    </Container>
  )
}
