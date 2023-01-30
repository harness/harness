import React, { useMemo, useState } from 'react'
import {
  Container,
  PageBody,
  Text,
  Color,
  TableV2,
  Layout,
  Utils,
  useToaster,
  IconName,
  Toggle
} from '@harness/uicore'
import { useHistory } from 'react-router-dom'
import { useGet, useMutate } from 'restful-react'
import type { CellProps, Column } from 'react-table'
import { useAppContext } from 'AppContext'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { useStrings } from 'framework/strings'
import { RepositoryPageHeader } from 'components/RepositoryPageHeader/RepositoryPageHeader'
import { voidFn, getErrorMessage, LIST_FETCHING_LIMIT } from 'utils/Utils'
import { OptionsMenuButton } from 'components/OptionsMenuButton/OptionsMenuButton'
import { useConfirmAct } from 'hooks/useConfirmAction'
import { usePageIndex } from 'hooks/usePageIndex'
import { ResourceListingPagination } from 'components/ResourceListingPagination/ResourceListingPagination'
import { LoadingSpinner } from 'components/LoadingSpinner/LoadingSpinner'
import { NoResultCard } from 'components/NoResultCard/NoResultCard'
import type { OpenapiWebhookType } from 'services/code'
import { WebhooksHeader } from './WebhooksHeader/WebhooksHeader'
import css from './Webhooks.module.scss'

export default function Webhooks() {
  const { getString } = useStrings()
  const history = useHistory()
  const { routes } = useAppContext()
  const [page, setPage] = usePageIndex()
  const [searchTerm, setSearchTerm] = useState('')
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
    lazy: !repoMetadata
  })

  const columns: Column<OpenapiWebhookType>[] = useMemo(
    () => [
      {
        id: 'title',
        width: 'calc(100% - 75px)',
        Cell: ({ row }: CellProps<OpenapiWebhookType>) => {
          const [checked, setChecked] = useState<boolean>(row.original.enabled || false)

          const { mutate } = useMutate<OpenapiWebhookType>({
            verb: 'PATCH',
            path: `/api/v1/repos/${repoMetadata?.path}/+/webhooks/${row.original?.id}`
          })
          return (
            <Layout.Horizontal spacing="medium" padding={{ left: 'medium' }} flex={{ alignItems: 'center' }}>
              <Container onClick={Utils.stopEvent}>
                <Toggle
                  key={row.original.id}
                  className={css.toggle}
                  checked={checked}
                  onChange={() => {
                    const data = { enabled: !checked }
                    mutate(data)
                      .then(() => {
                        showSuccess(getString('webhookUpdated'))
                        refetchWebhooks()
                      })
                      .catch(e => {
                        showError(e)
                      })
                    setChecked(!checked)
                  }}></Toggle>
              </Container>
              <Container padding={{ left: 'small' }} style={{ flexGrow: 1 }}>
                <Layout.Horizontal spacing="small">
                  <Text color={Color.PRIMARY_7} className={css.title}>
                    {row.original.display_name}
                  </Text>
                  {!!row.original.triggers?.length && (
                    <Text color={Color.GREY_500}>({row.original.triggers.join(', ')})</Text>
                  )}
                  {!row.original.triggers?.length && (
                    <Text color={Color.GREY_500}>{getString('webhookAllEventsSelected')}</Text>
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
          return <Text margin={{ right: 'medium' }} {...generateLastExecutionStateIcon(row.original)}></Text>
        }
      },
      {
        id: 'action',
        width: '60px',
        Cell: ({ row }: CellProps<OpenapiWebhookType>) => {
          const { mutate: deleteWebhook } = useMutate({
            verb: 'DELETE',
            path: `/api/v1/repos/${repoMetadata?.path}/+/webhooks/${row.original.id}`
          })
          const confirmDelete = useConfirmAct()

          return (
            <Container margin={{ left: 'medium' }} onClick={Utils.stopEvent}>
              <OptionsMenuButton
                width="100px"
                isDark
                items={[
                  {
                    hasIcon: true,
                    iconName: 'Edit',
                    text: getString('edit'),
                    onClick: () => {
                      history.push(
                        routes.toCODEWebhookDetails({
                          repoPath: repoMetadata?.path as string,
                          webhookId: String(row.original?.id)
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
                  }
                ]}
              />
            </Container>
          )
        }
      }
    ],
    [history, getString, refetchWebhooks, repoMetadata?.path, routes, setPage]
  )

  return (
    <Container className={css.main}>
      <RepositoryPageHeader repoMetadata={repoMetadata} title={getString('webhooks')} dataTooltipId="webhooks" />
      <PageBody error={getErrorMessage(error || webhooksError)} retryOnError={voidFn(refetch)}>
        <LoadingSpinner visible={loading} />

        {repoMetadata && (
          <Layout.Vertical>
            <WebhooksHeader
              repoMetadata={repoMetadata}
              loading={webhooksLoading}
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
                          webhookId: String(row.id)
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
                buttonText={getString('createWebhook')}
                onButtonClick={() =>
                  history.push(
                    routes.toCODEWebhookNew({
                      repoPath: repoMetadata?.path as string
                    })
                  )
                }
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
      icon = 'coverage-status-error'
      break
    case 'success':
      icon = 'coverage-status-success'
      break
    default:
      color = Color.GREY_250
  }

  return { icon, ...(color ? { iconProps: { color } } : undefined) }
}
