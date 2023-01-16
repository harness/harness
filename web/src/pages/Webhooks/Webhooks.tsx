import React, { useMemo } from 'react'
import {
  Button,
  Container,
  ButtonVariation,
  PageBody,
  Text,
  Color,
  TableV2,
  Layout,
  Icon,
  Utils,
  useToaster,
  IconName
} from '@harness/uicore'
import { useHistory } from 'react-router-dom'
import { useGet, useMutate } from 'restful-react'
import type { CellProps, Column } from 'react-table'
import { CodeIcon } from 'utils/GitUtils'
import { useAppContext } from 'AppContext'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { useStrings } from 'framework/strings'
import { RepositoryPageHeader } from 'components/RepositoryPageHeader/RepositoryPageHeader'
import { voidFn, getErrorMessage, LIST_FETCHING_LIMIT } from 'utils/Utils'
import emptyStateImage from 'images/empty-state.svg'
import { OptionsMenuButton } from 'components/OptionsMenuButton/OptionsMenuButton'
import { useConfirmAct } from 'hooks/useConfirmAction'
import { usePageIndex } from 'hooks/usePageIndex'
import type { OpenapiWebhookType } from 'services/code'
import { WebhooksHeader } from './WebhooksHeader/WebhooksHeader'
import css from './Webhooks.module.scss'

export default function Webhooks() {
  const { getString } = useStrings()
  const history = useHistory()
  const { routes } = useAppContext()
  const [pageIndex, setPageIndex] = usePageIndex()
  const { repoMetadata, error, loading, refetch } = useGetRepositoryMetadata()
  const {
    data: webhooks,
    loading: webhooksLoading,
    error: webhooksError,
    refetch: refetchWebhooks
  } = useGet<OpenapiWebhookType[]>({
    path: `/api/v1/repos/${repoMetadata?.path}/+/webhooks`,
    queryParams: {
      limit: LIST_FETCHING_LIMIT,
      page: pageIndex + 1,
      sort: 'date',
      order: 'asc'
    },
    lazy: !repoMetadata
  })

  const columns: Column<OpenapiWebhookType>[] = useMemo(
    () => [
      {
        id: 'title',
        width: 'calc(100% - 40px)',
        Cell: ({ row }: CellProps<OpenapiWebhookType>) => {
          return (
            <Layout.Horizontal spacing="medium" padding={{ left: 'medium' }} flex={{ alignItems: 'center' }}>
              <Icon name="code-webhook" size={32} />
              <Container padding={{ left: 'small' }} style={{ flexGrow: 1 }}>
                <Layout.Vertical spacing="small">
                  <Text {...generateLastExecutionStateIcon(row.original)} color={Color.GREY_800} className={css.title}>
                    {row.original.display_name}
                  </Text>
                  {!!row.original.triggers?.length && (
                    <Text color={Color.GREY_500}>({row.original.triggers.join(', ')})</Text>
                  )}
                  {!row.original.triggers?.length && (
                    <Text color={Color.GREY_500}>{getString('webhookAllEventsSelected')}</Text>
                  )}
                </Layout.Vertical>
              </Container>
            </Layout.Horizontal>
          )
        }
      },
      {
        id: 'action',
        width: '40px',
        Cell: ({ row }: CellProps<OpenapiWebhookType>) => {
          const { mutate: deleteWebhook } = useMutate({
            verb: 'DELETE',
            path: `/api/v1/repos/${repoMetadata?.path}/+/webhooks/${row.original.id}`
          })
          const { showSuccess, showError } = useToaster()
          const confirmDelete = useConfirmAct()

          return (
            <Container onClick={Utils.stopEvent}>
              <OptionsMenuButton
                width="100px"
                items={[
                  {
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
                    isDanger: true,
                    text: getString('delete'),
                    onClick: async () => {
                      confirmDelete({
                        message: getString('confirmDeleteWebhook'),
                        action: async () => {
                          deleteWebhook({})
                            .then(() => {
                              showSuccess(getString('webhookDeleted'), 5000)
                              setPageIndex(0)
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
    [history, getString, refetchWebhooks, repoMetadata?.path, routes, setPageIndex]
  )

  return (
    <Container className={css.main}>
      <RepositoryPageHeader repoMetadata={repoMetadata} title={getString('webhooks')} dataTooltipId="webhooks" />
      <PageBody
        loading={loading || webhooksLoading}
        error={getErrorMessage(error || webhooksError)}
        retryOnError={voidFn(refetch)}
        noData={{
          // TODO: Use NoDataCard, this won't render toolbar
          // when search returns empty response
          when: () => webhooks?.length === 0,
          message: getString('noWebHooks'),
          image: emptyStateImage,
          button: (
            <Button
              variation={ButtonVariation.PRIMARY}
              text={getString('createWebhook')}
              icon={CodeIcon.Add}
              onClick={() => history.push(routes.toCODEWebhookNew({ repoPath: repoMetadata?.path as string }))}
            />
          )
        }}>
        {repoMetadata && (
          <Layout.Vertical>
            <WebhooksHeader repoMetadata={repoMetadata} />
            {!!webhooks?.length && (
              <Container padding="xlarge">
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
              </Container>
            )}
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
