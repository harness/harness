/*
 * Copyright 2023 Harness, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
import React, { useEffect, useMemo } from 'react'
import { Container, TableV2, Text, Button, ButtonVariation, useToaster } from '@harnessio/uicore'

import type { CellProps, Column } from 'react-table'
import { useGet } from 'restful-react'
import { Color, FontVariation } from '@harnessio/design-system'
import { useHistory } from 'react-router-dom'
import { defaultTo, isEmpty } from 'lodash-es'
import { useQueryParams } from 'hooks/useQueryParams'
import { usePageIndex } from 'hooks/usePageIndex'
import { LIST_FETCHING_LIMIT, type PageBrowserProps } from 'utils/Utils'
import { ExecutionTabs, WebhookIndividualEvent, getEventDescription } from 'utils/GitUtils'
import { ResourceListingPagination } from 'components/ResourceListingPagination/ResourceListingPagination'
import { NoResultCard } from 'components/NoResultCard/NoResultCard'
import { useStrings } from 'framework/strings'
import { useUpdateQueryParams } from 'hooks/useUpdateQueryParams'
import { LoadingSpinner } from 'components/LoadingSpinner/LoadingSpinner'
import { getConfig } from 'services/config'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import type { TypesWebhookExecution } from 'services/code'
import { TimePopoverWithLocal } from 'utils/timePopoverLocal/TimePopoverWithLocal'
import { useWeebhookLogDrawer } from 'pages/WebhookExecutions/WebhookExecutionLogs/useWeebhookLogDrawer'
import { ExecutionStatusLabel } from 'components/ExecutionStatusLabel/ExecutionStatusLabel'
import css from './WebhookExecutions.module.scss'

const WebhookExecutions = () => {
  const { repoMetadata, webhookId } = useGetRepositoryMetadata()
  const { getString } = useStrings()
  const { showError, showSuccess } = useToaster()
  const history = useHistory()
  const pageBrowser = useQueryParams<PageBrowserProps>()
  const { updateQueryParams, replaceQueryParams } = useUpdateQueryParams()
  const pageInit = pageBrowser.page ? parseInt(pageBrowser.page) : 1
  const [page, setPage] = usePageIndex(pageInit)

  useEffect(() => {
    const params = {
      ...pageBrowser,
      ...(page > 1 && { page: page.toString() })
    }
    updateQueryParams(params, undefined, true)

    if (page <= 1) {
      const updateParams = { ...params }
      delete updateParams.page
      replaceQueryParams(updateParams, undefined, true)
    }
  }, [page]) // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    if (parseInt(pageBrowser.page ?? '1') !== page) {
      setPage(parseInt(pageBrowser.page ?? '1'))
    }
  }, [pageBrowser])

  const {
    data: executionList,
    loading: executionListLoading,
    refetch: refetchExecutionList,
    response
  } = useGet<TypesWebhookExecution[]>({
    base: getConfig('code/api/v1'),
    path: `/repos/${repoMetadata?.path}/+/webhooks/${webhookId}/executions`,
    queryParams: {
      limit: LIST_FETCHING_LIMIT,
      page: page
    }
  })

  const { openExecutionLogs } = useWeebhookLogDrawer(refetchExecutionList)

  const columns: Column<TypesWebhookExecution>[] = useMemo(
    () => [
      {
        Header: getString('executionId'),
        id: getString('executionId'),
        sort: 'true',
        width: '16.66%',
        Cell: ({ row }: CellProps<TypesWebhookExecution>) => {
          return (
            <Text color={Color.GREY_900} icon={'execution'}>
              {row.original.id}
            </Text>
          )
        }
      },
      {
        Header: getString('lastTriggeredAt'),
        id: getString('lastTriggeredAt'),
        sort: 'true',
        width: '16.66%',
        Cell: ({ row }: CellProps<TypesWebhookExecution>) => {
          return (
            <Text>
              <TimePopoverWithLocal
                time={defaultTo(row.original.created as number, 0)}
                inline={false}
                font={{ variation: FontVariation.BODY2_SEMI }}
                color={Color.GREY_400}
              />
            </Text>
          )
        }
      },
      {
        Header: getString('event'),
        id: getString('event'),
        sort: 'true',
        width: '16.66%',
        Cell: ({ row }: CellProps<TypesWebhookExecution>) => {
          return <Text>{getEventDescription(row.original.trigger_type as WebhookIndividualEvent)}</Text>
        }
      },
      {
        Header: getString('requestPayload'),
        id: getString('requestPayload'),
        sort: 'true',
        width: '16.66%',
        Cell: ({ row }: CellProps<TypesWebhookExecution>) => {
          return (
            <Button
              variation={ButtonVariation.LINK}
              font={{ variation: FontVariation.FORM_LABEL }}
              text={'View'}
              iconProps={{ size: 16 }}
              icon={'file'}
              onClick={() => openExecutionLogs(row.original, ExecutionTabs.PAYLOAD, repoMetadata)}
            />
          )
        }
      },
      {
        Header: getString('serverResponse'),
        id: getString('serverResponse'),
        sort: 'true',
        width: '16.66%',
        Cell: ({ row }: CellProps<TypesWebhookExecution>) => {
          return (
            <Button
              variation={ButtonVariation.LINK}
              font={{ variation: FontVariation.FORM_LABEL }}
              text={'View'}
              iconProps={{ size: 18 }}
              icon={'sto-dast'}
              onClick={() => openExecutionLogs(row.original, ExecutionTabs.SERVER_RESPONSE, repoMetadata)}
            />
          )
        }
      },
      {
        Header: getString('status'),
        id: getString('status'),
        sort: 'true',
        width: '16.66%',
        Cell: ({ row }: CellProps<TypesWebhookExecution>) => {
          return (
            <ExecutionStatusLabel
              data={row.original.result === 'success' ? { state: 'success' } : { state: 'failed' }}
            />
          )
        }
      }
    ], // eslint-disable-next-line react-hooks/exhaustive-deps
    [history, getString, repoMetadata?.path, setPage, showError, showSuccess]
  )

  return (
    <Container>
      <Container className={css.main} padding={{ bottom: 'large', right: 'xlarge', left: 'xlarge' }}>
        {executionList && !executionListLoading && !!executionList.length && (
          <TableV2<TypesWebhookExecution>
            className={css.table}
            columns={columns}
            data={executionList}
            sortable
            autoResetExpanded={true}
            getRowClassName={() => css.row}
          />
        )}
        <LoadingSpinner visible={executionListLoading} />
        <ResourceListingPagination response={response} page={page} setPage={setPage} />
      </Container>
      <NoResultCard
        showWhen={() => !executionListLoading && isEmpty(executionList)}
        forSearch={true}
        title={getString('noExecutionsFound')}
        emptySearchMessage={getString('noExecutionsFoundForWebhook')}
      />
    </Container>
  )
}

export default WebhookExecutions
