import React, { useMemo, useState } from 'react'
import {
  Button,
  ButtonVariation,
  Container,
  FlexExpander,
  Layout,
  PageBody,
  TableV2 as Table,
  Text
} from '@harnessio/uicore'
import { Color } from '@harnessio/design-system'
import cx from 'classnames'
import type { CellProps, Column } from 'react-table'
import Keywords from 'react-keywords'
import { useHistory } from 'react-router-dom'
import { useGet } from 'restful-react'
import { useStrings } from 'framework/strings'
import { LoadingSpinner } from 'components/LoadingSpinner/LoadingSpinner'
import { SearchInputWithSpinner } from 'components/SearchInputWithSpinner/SearchInputWithSpinner'
import { NoResultCard } from 'components/NoResultCard/NoResultCard'
import { LIST_FETCHING_LIMIT, PageBrowserProps, formatDate, getErrorMessage, voidFn } from 'utils/Utils'
import type { TypesPipeline } from 'services/code'
import { useQueryParams } from 'hooks/useQueryParams'
import { usePageIndex } from 'hooks/usePageIndex'
import { ResourceListingPagination } from 'components/ResourceListingPagination/ResourceListingPagination'
import { useAppContext } from 'AppContext'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { RepositoryPageHeader } from 'components/RepositoryPageHeader/RepositoryPageHeader'
import noPipelineImage from '../RepositoriesListing/no-repo.svg'
import css from './PipelineList.module.scss'

const PipelineList = () => {
  const { routes } = useAppContext()
  const history = useHistory()
  const { getString } = useStrings()
  const [searchTerm, setSearchTerm] = useState<string | undefined>()
  const pageBrowser = useQueryParams<PageBrowserProps>()
  const pageInit = pageBrowser.page ? parseInt(pageBrowser.page) : 1
  const [page, setPage] = usePageIndex(pageInit)

  const { repoMetadata, error, loading, refetch } = useGetRepositoryMetadata()

  const {
    data: pipelines,
    error: pipelinesError,
    loading: pipelinesLoading,
    response
  } = useGet<TypesPipeline[]>({
    path: `/api/v1/repos/${repoMetadata?.path}/+/pipelines`,
    queryParams: { page, limit: LIST_FETCHING_LIMIT, query: searchTerm },
    lazy: !repoMetadata
  })

  const NewPipelineButton = (
    <Button
      text={getString('pipelines.newPipelineButton')}
      variation={ButtonVariation.PRIMARY}
      icon="plus"
      onClick={() => {
        history.push(routes.toCODEPipelinesNew({ space }))
      }}></Button>
  )

  const columns: Column<TypesPipeline>[] = useMemo(
    () => [
      {
        Header: getString('pipelines.name'),
        width: 'calc(100% - 180px)',
        Cell: ({ row }: CellProps<TypesPipeline>) => {
          const record = row.original
          return (
            <Container className={css.nameContainer}>
              <Layout.Horizontal spacing="small" style={{ flexGrow: 1 }}>
                <Layout.Vertical flex className={css.name}>
                  <Text className={css.repoName}>
                    <Keywords value={searchTerm}>{record.uid}</Keywords>
                  </Text>
                  {record.description && <Text className={css.desc}>{record.description}</Text>}
                </Layout.Vertical>
              </Layout.Horizontal>
            </Container>
          )
        }
      },
      {
        Header: getString('repos.updated'),
        width: '180px',
        Cell: ({ row }: CellProps<TypesPipeline>) => {
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
    [getString, searchTerm]
  )

  return (
    <Container className={css.main}>
      <RepositoryPageHeader
        repoMetadata={repoMetadata}
        title={getString('pageTitle.pipelines')}
        dataTooltipId="repositoryPipelines"
      />
      <PageBody
        className={cx({ [css.withError]: !!error })}
        error={error ? getErrorMessage(error || pipelinesError) : null}
        retryOnError={voidFn(refetch)}
        noData={{
          when: () => pipelines?.length === 0 && searchTerm === undefined,
          image: noPipelineImage,
          message: getString('pipelines.noData'),
          button: NewPipelineButton
        }}>
        <LoadingSpinner
          visible={(loading || pipelinesLoading) && !searchTerm}
          withBorder={!!pipelines && pipelinesLoading}
        />

        <Container padding="xlarge">
          <Layout.Horizontal spacing="large" className={css.layout}>
            {NewPipelineButton}
            <FlexExpander />
            <SearchInputWithSpinner loading={loading} query={searchTerm} setQuery={setSearchTerm} />
          </Layout.Horizontal>

          <Container margin={{ top: 'medium' }}>
            {!!pipelines?.length && (
              <Table<TypesPipeline>
                className={css.table}
                columns={columns}
                data={pipelines || []}
                onRowClick={pipelineInfo =>
                  history.push(
                    routes.toCODEExecutions({
                      repoPath: repoMetadata?.path as string,
                      pipeline: pipelineInfo.uid as string
                    })
                  )
                }
                getRowClassName={row => cx(css.row, !row.original.description && css.noDesc)}
              />
            )}

            <NoResultCard
              showWhen={() => !!pipelines && pipelines?.length === 0 && !!searchTerm?.length}
              forSearch={true}
            />
          </Container>
          <ResourceListingPagination response={response} page={page} setPage={setPage} />
        </Container>
      </PageBody>
    </Container>
  )
}

export default PipelineList
