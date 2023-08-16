import React, { useMemo } from 'react'
import {
  Button,
  ButtonVariation,
  Color,
  Container,
  FlexExpander,
  Layout,
  PageBody,
  PageHeader,
  TableV2 as Table,
  Text
} from '@harness/uicore'
import cx from 'classnames'
import type { CellProps, Column } from 'react-table'
import { useHistory, useParams } from 'react-router-dom'
import { useGet } from 'restful-react'
import { useStrings } from 'framework/strings'
import { LoadingSpinner } from 'components/LoadingSpinner/LoadingSpinner'
import { useAppContext } from 'AppContext'
import { NoResultCard } from 'components/NoResultCard/NoResultCard'
import { LIST_FETCHING_LIMIT, PageBrowserProps, formatDate, getErrorMessage, voidFn } from 'utils/Utils'
import type { CODEProps } from 'RouteDefinitions'
import type { TypesExecution } from 'services/code'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { useQueryParams } from 'hooks/useQueryParams'
import { usePageIndex } from 'hooks/usePageIndex'
import { ResourceListingPagination } from 'components/ResourceListingPagination/ResourceListingPagination'
import noExecutionImage from '../RepositoriesListing/no-repo.svg'
import css from './ExecutionList.module.scss'

const ExecutionList = () => {
  const { routes } = useAppContext()
  const space = useGetSpaceParam()
  const { pipeline } = useParams<CODEProps>()
  const history = useHistory()
  const { getString } = useStrings()
  const pageBrowser = useQueryParams<PageBrowserProps>()
  const pageInit = pageBrowser.page ? parseInt(pageBrowser.page) : 1
  const [page, setPage] = usePageIndex(pageInit)

  const {
    data: executions,
    error,
    loading,
    refetch,
    response
  } = useGet<TypesExecution[]>({
    path: `/api/v1/pipelines/${space}/${pipeline}/+/executions`,
    queryParams: { page, limit: LIST_FETCHING_LIMIT }
  })

  const NewExecutionButton = (
    <Button
      text={getString('executions.newExecutionButton')}
      variation={ButtonVariation.PRIMARY}
      disabled={true}
      icon="plus"></Button>
  )

  const columns: Column<TypesExecution>[] = useMemo(
    () => [
      {
        Header: getString('repos.name'),
        width: 'calc(100% - 180px)',
        Cell: ({ row }: CellProps<TypesExecution>) => {
          const record = row.original
          return (
            <Container className={css.nameContainer}>
              <Layout.Horizontal spacing="small" style={{ flexGrow: 1 }}>
                <Layout.Vertical flex className={css.name}>
                  <Text className={css.repoName}>{record.number}</Text>
                  {record.status && <Text className={css.desc}>{record.status}</Text>}
                </Layout.Vertical>
              </Layout.Horizontal>
            </Container>
          )
        }
      },
      {
        Header: getString('repos.updated'),
        width: '180px',
        Cell: ({ row }: CellProps<TypesExecution>) => {
          return (
            <Layout.Horizontal style={{ alignItems: 'center' }}>
              <Text color={Color.BLACK} lineClamp={1} rightIconProps={{ size: 10 }} width={120}>
                {formatDate(row.original.updated as number)}
              </Text>
            </Layout.Horizontal>
          )
        },
        disableSortBy: true
      }
    ],
    [getString]
  )

  return (
    <Container className={css.main}>
      <PageHeader title={getString('pageTitle.executions')} />
      <PageBody
        className={cx({ [css.withError]: !!error })}
        error={error ? getErrorMessage(error) : null}
        retryOnError={voidFn(refetch)}
        noData={{
          when: () => executions?.length === 0,
          image: noExecutionImage,
          message: getString('executions.noData'),
          button: NewExecutionButton
        }}>
        <LoadingSpinner visible={loading} />

        <Container padding="xlarge">
          <Layout.Horizontal spacing="large" className={css.layout}>
            {NewExecutionButton}
            <FlexExpander />
          </Layout.Horizontal>

          <Container margin={{ top: 'medium' }}>
            {!!executions?.length && (
              <Table<TypesExecution>
                className={css.table}
                columns={columns}
                data={executions || []}
                onRowClick={executionInfo =>
                  history.push(
                    routes.toCODEExecution({
                      space,
                      pipeline: pipeline as string,
                      execution: String(executionInfo.id)
                    })
                  )
                }
                getRowClassName={row => cx(css.row, !row.original.number && css.noDesc)}
              />
            )}

            <NoResultCard showWhen={() => !!executions && executions.length === 0} forSearch={true} />
          </Container>
          <ResourceListingPagination response={response} page={page} setPage={setPage} />
        </Container>
      </PageBody>
    </Container>
  )
}

export default ExecutionList
