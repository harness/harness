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

import React, { useEffect, useMemo, useState } from 'react'
import {
  Container,
  PageBody,
  Text,
  TableV2,
  Layout,
  Utils,
  useToaster,
  Toggle,
  Popover,
  Button,
  ButtonVariation,
  StringSubstitute
} from '@harnessio/uicore'
import { Icon, IconName } from '@harnessio/icons'
import { Color, FontVariation } from '@harnessio/design-system'
import { useHistory } from 'react-router-dom'
import cx from 'classnames'
import { Position } from '@blueprintjs/core'
import { useGet, useMutate } from 'restful-react'
import type { CellProps, Column } from 'react-table'
import { useAppContext } from 'AppContext'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { useStrings } from 'framework/strings'
import { RepositoryPageHeader } from 'components/RepositoryPageHeader/RepositoryPageHeader'
import { voidFn, getErrorMessage, LIST_FETCHING_LIMIT, PageBrowserProps, permissionProps } from 'utils/Utils'
import { OptionsMenuButton } from 'components/OptionsMenuButton/OptionsMenuButton'
import { useConfirmAct } from 'hooks/useConfirmAction'
import { usePageIndex } from 'hooks/usePageIndex'
import { useUpdateQueryParams } from 'hooks/useUpdateQueryParams'
import { useQueryParams } from 'hooks/useQueryParams'
import { ResourceListingPagination } from 'components/ResourceListingPagination/ResourceListingPagination'
import { LoadingSpinner } from 'components/LoadingSpinner/LoadingSpinner'
import { NoResultCard } from 'components/NoResultCard/NoResultCard'
import type { OpenapiWebhookType } from 'services/code'
import { WebhookTabs, formatTriggers } from 'utils/GitUtils'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { WebhooksHeader } from './WebhooksHeader/WebhooksHeader'
import css from './Webhooks.module.scss'

export default function Webhooks() {
  const { getString } = useStrings()
  const history = useHistory()
  const { routes } = useAppContext()
  const { updateQueryParams } = useUpdateQueryParams()

  const pageBrowser = useQueryParams<PageBrowserProps>()
  const pageInit = pageBrowser.page ? parseInt(pageBrowser.page) : 1
  const [page, setPage] = usePageIndex(pageInit)
  const [searchTerm, setSearchTerm] = useState<string | undefined>()
  const { repoMetadata, error, loading, refetch } = useGetRepositoryMetadata()
  const { showError, showSuccess } = useToaster()
  const {
    data: webhooks,
    loading: webhooksLoading,
    error: webhooksError,
    refetch: refetchWebhooks,
    response
  } = useGet<OpenapiWebhookType[]>({
    path: `/api/v1/repos/${repoMetadata?.path}/+/webhooks`,
    queryParams: {
      limit: LIST_FETCHING_LIMIT,
      page,
      sort: 'date',
      order: 'desc',
      query: searchTerm
    },
    debounce: 500,
    lazy: !repoMetadata
  })
  useEffect(() => {
    if (page > 1) {
      updateQueryParams({ page: page.toString() })
    }
  }, [setPage]) // eslint-disable-line react-hooks/exhaustive-deps

  const columns: Column<OpenapiWebhookType>[] = useMemo(
    () => [
      {
        id: 'title',
        width: 'calc(100% - 75px)',
        Cell: ({ row }: CellProps<OpenapiWebhookType>) => {
          const [checked, setChecked] = useState<boolean>(row.original.enabled || false)

          const { mutate } = useMutate<OpenapiWebhookType>({
            verb: 'PATCH',
            path: `/api/v1/repos/${repoMetadata?.path}/+/webhooks/${row.original?.identifier}`
          })
          const [popoverDialogOpen, setPopoverDialogOpen] = useState(false)

          return (
            <Layout.Horizontal spacing="medium" padding={{ left: 'medium' }} flex={{ alignItems: 'center' }}>
              <Icon name="code-webhook" size={24} />

              <Container onClick={Utils.stopEvent}>
                <Popover
                  isOpen={popoverDialogOpen && permPushResult}
                  onInteraction={nextOpenState => {
                    setPopoverDialogOpen(nextOpenState)
                  }}
                  content={
                    <Container padding={'medium'} width={250}>
                      <Layout.Vertical>
                        <Text font={{ variation: FontVariation.H5, size: 'medium' }}>
                          {checked ? getString('disableWebhookTitle') : getString('enableWebhookTitle')}
                        </Text>
                        <Text
                          padding={{ top: 'medium', bottom: 'medium' }}
                          font={{ variation: FontVariation.BODY2_SEMI }}>
                          <StringSubstitute
                            str={checked ? getString('disableWebhookContent') : getString('enableWebhookContent')}
                            vars={{
                              name: <strong>{row.original?.identifier}</strong>
                            }}
                          />
                        </Text>
                        <Layout.Horizontal>
                          <Button
                            variation={ButtonVariation.PRIMARY}
                            text={getString('confirm')}
                            onClick={() => {
                              const data = { enabled: !checked }
                              mutate(data)
                                .then(() => {
                                  showSuccess(getString('webhookUpdated'))
                                })
                                .catch(err => {
                                  showError(getErrorMessage(err))
                                })
                              setChecked(!checked)
                              setPopoverDialogOpen(false)
                            }}></Button>
                          <Container>
                            <Button
                              className={css.cancelButton}
                              text={getString('cancel')}
                              onClick={() => {
                                setPopoverDialogOpen(false)
                              }}></Button>
                          </Container>
                        </Layout.Horizontal>
                      </Layout.Vertical>
                    </Container>
                  }
                  position={Position.RIGHT}
                  interactionKind="click">
                  <Toggle
                    {...permissionProps(permPushResult, standalone)}
                    key={row.original.identifier}
                    className={cx(css.toggle, checked ? css.toggleEnable : css.toggleDisable)}
                    checked={checked}></Toggle>
                </Popover>
              </Container>
              <Container padding={{ left: 'small' }} style={{ flexGrow: 1 }}>
                <Layout.Horizontal spacing="small">
                  <Text
                    color={Color.PRIMARY_7}
                    padding={{ right: 'small' }}
                    lineClamp={1}
                    width={300}
                    className={css.title}>
                    {row.original.identifier}
                  </Text>
                  {!!row.original.triggers?.length && (
                    <Text padding={{ left: 'small', right: 'small' }} color={Color.GREY_500}>
                      ({formatTriggers(row.original?.triggers).join(', ')})
                    </Text>
                  )}
                  {!row.original.triggers?.length && (
                    <Text padding={{ left: 'small', right: 'small' }} color={Color.GREY_500}>
                      {getString('webhookAllEventsSelected')}
                    </Text>
                  )}
                </Layout.Horizontal>
              </Container>
            </Layout.Horizontal>
          )
        }
      },
      {
        id: 'executionStatus',
        width: '15px',
        Cell: ({ row }: CellProps<OpenapiWebhookType>) => {
          return (
            <Text
              iconProps={{ size: 24 }}
              margin={{ right: 'medium' }}
              {...generateLastExecutionStateIcon(row.original)}></Text>
          )
        }
      },
      {
        id: 'action',
        width: '60px',
        Cell: ({ row }: CellProps<OpenapiWebhookType>) => {
          const { mutate: deleteWebhook } = useMutate({
            verb: 'DELETE',
            path: `/api/v1/repos/${repoMetadata?.path}/+/webhooks/${row.original.identifier}`
          })
          const confirmDelete = useConfirmAct()

          return (
            <Container margin={{ left: 'medium' }} onClick={Utils.stopEvent}>
              <OptionsMenuButton
                width="100px"
                isDark
                {...permissionProps(permPushResult, standalone)}
                items={[
                  {
                    hasIcon: true,
                    iconName: 'Edit',
                    text: getString('edit'),
                    onClick: () => {
                      history.push(
                        routes.toCODEWebhookDetails({
                          repoPath: repoMetadata?.path as string,
                          webhookId: String(row.original?.identifier)
                        })
                      )
                    }
                  },
                  {
                    hasIcon: true,
                    iconName: 'main-trash',
                    text: getString('delete'),
                    onClick: async () => {
                      confirmDelete({
                        message: getString('confirmDeleteWebhook'),
                        action: async () => {
                          deleteWebhook({})
                            .then(() => {
                              showSuccess(getString('webhookDeleted'), 5000)
                              setPage(1)
                              refetchWebhooks()
                            })
                            .catch(exception => {
                              showError(getErrorMessage(exception), 0, 'failedToDeleteWebhook')
                            })
                        }
                      })
                    }
                  },
                  {
                    hasIcon: true,
                    iconName: 'execution',
                    iconSize: 16,
                    text: getString('executionHistory'),
                    onClick: () => {
                      history.push(
                        `${routes.toCODEWebhookDetails({
                          repoPath: repoMetadata?.path as string,
                          webhookId: String(row.original?.identifier)
                        })}?tab=${WebhookTabs.EXECUTIONS}`
                      )
                    }
                  }
                ]}
              />
            </Container>
          )
        }
      }
    ], // eslint-disable-next-line react-hooks/exhaustive-deps
    [history, getString, refetchWebhooks, repoMetadata?.path, routes, setPage, showError, showSuccess]
  )
  const { hooks, standalone } = useAppContext()
  const space = useGetSpaceParam()
  const permPushResult = hooks?.usePermissionTranslate?.(
    {
      resource: {
        resourceType: 'CODE_REPOSITORY',
        resourceIdentifier: repoMetadata?.identifier as string
      },
      permissions: ['code_repo_edit']
    },
    [space]
  )
  return (
    <Container className={css.main}>
      <RepositoryPageHeader repoMetadata={repoMetadata} title={getString('webhooks')} dataTooltipId="webhooks" />
      <PageBody error={getErrorMessage(error || webhooksError)} retryOnError={voidFn(refetch)}>
        <LoadingSpinner visible={loading || (webhooksLoading && searchTerm === undefined)} />

        {repoMetadata && (
          <Layout.Vertical>
            <WebhooksHeader
              repoMetadata={repoMetadata}
              loading={webhooksLoading && searchTerm !== undefined}
              onSearchTermChanged={value => {
                setSearchTerm(value)
                setPage(1)
              }}
            />
            <Container padding="xlarge">
              {!!webhooks?.length && (
                <>
                  <TableV2<OpenapiWebhookType>
                    className={css.table}
                    hideHeaders
                    columns={columns}
                    data={webhooks}
                    getRowClassName={() => css.row}
                    onRowClick={row => {
                      history.push(
                        routes.toCODEWebhookDetails({
                          repoPath: repoMetadata.path as string,
                          webhookId: String(row.identifier)
                        })
                      )
                    }}
                  />

                  <ResourceListingPagination response={response} page={page} setPage={setPage} />
                </>
              )}

              <NoResultCard
                showWhen={() => webhooks?.length === 0}
                forSearch={!!searchTerm}
                message={getString('webhookEmpty')}
                buttonText={getString('newWebhook')}
                onButtonClick={() =>
                  history.push(
                    routes.toCODEWebhookNew({
                      repoPath: repoMetadata?.path as string
                    })
                  )
                }
                permissionProp={permissionProps(permPushResult, standalone)}
              />
            </Container>
          </Layout.Vertical>
        )}
      </PageBody>
    </Container>
  )
}

const generateLastExecutionStateIcon = (
  webhook: OpenapiWebhookType
): { icon: IconName; iconProps?: { color?: Color } } => {
  let icon: IconName = 'dot'
  let color: Color | undefined = undefined

  switch (webhook.latest_execution_result) {
    case 'fatal_error':
      icon = 'danger-icon'
      break
    case 'retriable_error':
      icon = 'solid-error'
      break
    case 'success':
      icon = 'success-tick'
      break
    default:
      color = Color.GREY_250
  }

  return { icon, ...(color ? { iconProps: { color } } : undefined) }
}
