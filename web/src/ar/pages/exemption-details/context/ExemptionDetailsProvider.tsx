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

import React, { createContext, PropsWithChildren } from 'react'
import {
  ListFirewallExceptionsV3QueryQueryParams,
  useListFirewallExceptionsV3Query,
  type FirewallExceptionResponseV3
} from '@harnessio/react-har-service-client'
import PageContent from '@ar/components/PageContent/PageContent'
import { useAppStore } from '@ar/hooks'

interface ExemptionDetailsContextProps {
  data: FirewallExceptionResponseV3
}

export const ExemptionDetailsContext = createContext<ExemptionDetailsContextProps>({
  data: {} as FirewallExceptionResponseV3
})

interface ExemptionDetailsProviderProps {
  exemptionId: string
}

export default function ExemptionDetailsProvider(props: PropsWithChildren<ExemptionDetailsProviderProps>) {
  const { scope } = useAppStore()
  const { accountId, orgIdentifier, projectIdentifier } = scope

  const { data, isFetching, error, refetch } = useListFirewallExceptionsV3Query({
    queryParams: {
      account_identifier: accountId || '',
      org_identifier: orgIdentifier,
      project_identifier: projectIdentifier,
      exception_id: props.exemptionId
    } as ListFirewallExceptionsV3QueryQueryParams
  })

  const exemptionDetails = data?.content?.items?.[0]
  return (
    <PageContent loading={isFetching} error={error?.error as any} refetch={refetch}>
      {exemptionDetails && (
        <ExemptionDetailsContext.Provider value={{ data: exemptionDetails }}>
          {props.children}
        </ExemptionDetailsContext.Provider>
      )}
    </PageContent>
  )
}
