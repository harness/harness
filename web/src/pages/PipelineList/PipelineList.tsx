import React, { useMemo, useState } from 'react'
import {
  Avatar,
  Button,
  ButtonVariation,
  Container,
  FlexExpander,
  Layout,
  PageBody,
  TableV2 as Table,
  Text,
  Utils
} from '@harnessio/uicore'
import { Color } from '@harnessio/design-system'
import cx from 'classnames'
import type { CellProps, Column } from 'react-table'
import Keywords from 'react-keywords'
import { useHistory } from 'react-router-dom'
import { useGet } from 'restful-react'
import { Icon } from '@harnessio/icons'
import { Calendar, Timer, GitFork } from 'iconoir-react'
import { useStrings } from 'framework/strings'
import { LoadingSpinner } from 'components/LoadingSpinner/LoadingSpinner'
import { SearchInputWithSpinner } from 'components/SearchInputWithSpinner/SearchInputWithSpinner'
import { NoResultCard } from 'components/NoResultCard/NoResultCard'
import { LIST_FETCHING_LIMIT, PageBrowserProps, getErrorMessage, timeDistance, voidFn } from 'utils/Utils'
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
    queryParams: { page, limit: LIST_FETCHING_LIMIT, query: searchTerm, latest: true },
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
        width: 'calc(50% - 90px)',
        Cell: ({ row }: CellProps<TypesPipeline>) => {
          const record = row.original
          return (
            <Layout.Horizontal spacing="small" className={css.nameContainer}>
              {/* TODO this icon need to depend on the status */}
              <Icon name="success-tick" size={24} />
              <Text className={css.repoName}>
                <Keywords value={searchTerm}>{record.uid}</Keywords>
              </Text>
            </Layout.Horizontal>
          )
        }
      },
      {
        Header: getString('pipelines.lastExecution'),
        width: 'calc(50% - 90px)',
        Cell: ({ row }: CellProps<TypesPipeline>) => {
          const record = row.original.execution

          return record ? (
            <Layout.Vertical className={css.executionContainer}>
              <Layout.Horizontal spacing={'small'} style={{ alignItems: 'center' }}>
                {/* TODO this icon need to depend on the status */}
                <Text className={css.desc}>{`#${record.number}`}</Text>
                <Text className={css.divider}>{`|`}</Text>
                <Text className={css.desc}>{record.title}</Text>
              </Layout.Horizontal>
              <Layout.Horizontal spacing={'xsmall'} style={{ alignItems: 'center' }}>
                <Avatar
                  email={record.author_email}
                  name={record.author_name}
                  size="small"
                  hoverCard={false}
                  className={css.avatar}
                />
                {/* TODO need logic here for different trigger types */}
                <Text className={css.author}>{record.author_name}</Text>
                <Text className={css.divider}>{`|`}</Text>
                <GitFork height={12} width={12} color={Utils.getRealCSSColor(Color.GREY_500)} />
                <Text className={css.author}>{record.source}</Text>
                <Text className={css.divider}>{`|`}</Text>
                {/* TODO Will need to replace this with commit component - wont match Yifan designs */}
                <a rel="noreferrer noopener" className={css.hash}>
                  {/* {record.after} */}
                  hardcoded
                </a>
              </Layout.Horizontal>
            </Layout.Vertical>
          ) : (
            <div className={css.spacer} />
          )
        }
      },
      {
        Header: getString('pipelines.time'),
        width: '180px',
        Cell: ({ row }: CellProps<TypesPipeline>) => {
          const record = row.original.execution

          return record ? (
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
          ) : (
            <div className={css.spacer} />
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
