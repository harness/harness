import React, { useMemo } from 'react'
import {
  Avatar,
  Button,
  ButtonVariation,
  Container,
  FlexExpander,
  Layout,
  PageBody,
  PageHeader,
  TableV2 as Table,
  Text,
  Utils
} from '@harnessio/uicore'
import { Color } from '@harnessio/design-system'
import cx from 'classnames'
import type { CellProps, Column } from 'react-table'
import { useHistory, useParams } from 'react-router-dom'
import { useGet } from 'restful-react'
import { Icon } from '@harnessio/icons'
import { Timer, Calendar } from 'iconoir-react'
import { useStrings } from 'framework/strings'
import { LoadingSpinner } from 'components/LoadingSpinner/LoadingSpinner'
import { useAppContext } from 'AppContext'
import { NoResultCard } from 'components/NoResultCard/NoResultCard'
import { LIST_FETCHING_LIMIT, PageBrowserProps, getErrorMessage, timeDistance, voidFn } from 'utils/Utils'
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
      icon="play-outline"></Button>
  )

  const columns: Column<TypesExecution>[] = useMemo(
    () => [
      {
        Header: getString('executions.description'),
        width: 'calc(100% - 180px)',
        Cell: ({ row }: CellProps<TypesExecution>) => {
          const record = row.original
          return (
            <Container className={css.nameContainer}>
              <Layout.Vertical>
                <Layout.Horizontal spacing={'small'} style={{ alignItems: 'center' }}>
                  {/* TODO this icon need to depend on the status */}
                  <Icon name="success-tick" size={18} />
                  <Text className={css.number}>{`#${record.number}.`}</Text>
                  <Text className={css.desc}>{record.title}</Text>
                </Layout.Horizontal>
                <Layout.Horizontal spacing={'small'} style={{ alignItems: 'center', marginLeft: '1.2rem' }}>
                  <Avatar email={record.author_email} name={record.author_name} size="small" hoverCard={false} />
                  {/* TODO need logic here for different trigger types */}
                  <Text className={css.author}>{`${record.author_name} triggered manually`}</Text>
                  <Text className={css.divider}>{`|`}</Text>
                  {/* TODO Will need to replace this with commit action - wont match Yifan designs */}
                  <a rel="noreferrer noopener" className={css.hash}>
                    {record.after}
                  </a>
                </Layout.Horizontal>
              </Layout.Vertical>
            </Container>
          )
        }
      },
      {
        Header: getString('executions.time'),
        width: '180px',
        Cell: ({ row }: CellProps<TypesExecution>) => {
          const record = row.original
          return (
            <Layout.Vertical spacing={'small'}>
              <Layout.Horizontal spacing={'small'} style={{ alignItems: 'center' }}>
                <Timer color={Utils.getRealCSSColor(Color.GREY_500)} />
                <Text inline color={Color.GREY_500} lineClamp={1} width={180} font={{ size: 'small' }}>
                  {timeDistance(record.started, record.finished)}
                </Text>
              </Layout.Horizontal>
              <Layout.Horizontal spacing={'small'} style={{ alignItems: 'center' }}>
                <Calendar color={Utils.getRealCSSColor(Color.GREY_500)} />
                <Text inline color={Color.GREY_500} lineClamp={1} width={180} font={{ size: 'small' }}>
                  {timeDistance(record.finished, Date.now())} ago
                </Text>
              </Layout.Horizontal>
            </Layout.Vertical>
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
                      execution: String(executionInfo.number)
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
