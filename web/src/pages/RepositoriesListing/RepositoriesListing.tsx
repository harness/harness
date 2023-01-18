import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import {
  ButtonVariation,
  Container,
  FlexExpander,
  Layout,
  PageBody,
  PageHeader,
  TableV2 as Table,
  Text,
  Color,
  Icon
} from '@harness/uicore'
import type { CellProps, Column } from 'react-table'
import Keywords from 'react-keywords'
import cx from 'classnames'
import { useGet } from 'restful-react'
import { useHistory } from 'react-router-dom'
import { useStrings } from 'framework/strings'
import { voidFn, formatDate, getErrorMessage, LIST_FETCHING_LIMIT } from 'utils/Utils'
import { NewRepoModalButton } from 'components/NewRepoModalButton/NewRepoModalButton'
import type { TypesRepository } from 'services/code'
import { usePageIndex } from 'hooks/usePageIndex'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { SearchInputWithSpinner } from 'components/SearchInputWithSpinner/SearchInputWithSpinner'
import { useAppContext } from 'AppContext'
import { NoResultCard } from 'components/NoResultCard/NoResultCard'
import { ResourceListingPagination } from 'components/ResourceListingPagination/ResourceListingPagination'
import emptyStateImage from './empty-state.svg'
import css from './RepositoriesListing.module.scss'

export default function RepositoriesListing() {
  const { getString } = useStrings()
  const history = useHistory()
  const rowContainerRef = useRef<HTMLDivElement>(null)
  const [nameTextWidth, setNameTextWidth] = useState(600)
  const space = useGetSpaceParam()
  const [searchTerm, setSearchTerm] = useState<string | undefined>()
  const { routes } = useAppContext()
  const [page, setPage] = usePageIndex()
  const {
    data: repositories,
    error,
    loading,
    refetch,
    response
  } = useGet<TypesRepository[]>({
    path: `/api/v1/spaces/${space}/+/repos`,
    queryParams: { page, limit: LIST_FETCHING_LIMIT, query: searchTerm }
  })

  useEffect(() => {
    setSearchTerm(undefined)
    setPage(1)
  }, [space, setPage])

  const columns: Column<TypesRepository>[] = useMemo(
    () => [
      {
        Header: getString('repos.name'),
        width: 'calc(100% - 180px)',
        Cell: ({ row }: CellProps<TypesRepository>) => {
          const record = row.original
          return (
            <Container className={css.nameContainer}>
              <Layout.Horizontal spacing="small" style={{ flexGrow: 1 }}>
                <Layout.Vertical flex className={css.name} ref={rowContainerRef}>
                  <Text className={css.repoName} width={nameTextWidth} lineClamp={2}>
                    <Keywords value={searchTerm}>{record.uid}</Keywords>
                  </Text>
                  {record.description && (
                    <Text className={css.desc} width={nameTextWidth} lineClamp={1}>
                      {record.description}
                    </Text>
                  )}
                </Layout.Vertical>
              </Layout.Horizontal>
            </Container>
          )
        }
      },
      {
        Header: getString('repos.updated'),
        width: '180px',
        Cell: ({ row }: CellProps<TypesRepository>) => {
          return (
            <Layout.Horizontal style={{ alignItems: 'center' }}>
              <Text color={Color.BLACK} lineClamp={1} rightIconProps={{ size: 10 }} width={120}>
                {formatDate(row.original.updated as number)}
              </Text>
              {row.original.is_public === false ? <Icon name="lock" size={10} /> : undefined}
            </Layout.Horizontal>
          )
        },
        disableSortBy: true
      }
    ],
    [nameTextWidth, getString, searchTerm]
  )
  const onResize = useCallback(() => {
    if (rowContainerRef.current) {
      setNameTextWidth((rowContainerRef.current.closest('div[role="cell"]') as HTMLDivElement)?.offsetWidth - 100)
    }
  }, [setNameTextWidth])
  const NewRepoButton = (
    <NewRepoModalButton
      space={space}
      modalTitle={getString('newRepo')}
      text={getString('newRepo')}
      variation={ButtonVariation.PRIMARY}
      icon="plus"
      onSubmit={repoInfo => history.push(routes.toCODERepository({ repoPath: repoInfo.path as string }))}
    />
  )

  useEffect(() => {
    onResize()
    window.addEventListener('resize', onResize)
    return () => {
      window.removeEventListener('resize', onResize)
    }
  }, [onResize])

  return (
    <Container className={css.main}>
      <PageHeader title={getString('repositories')} />
      <PageBody
        loading={loading && searchTerm === undefined}
        className={cx({ [css.withError]: !!error })}
        error={error ? getErrorMessage(error) : null}
        retryOnError={voidFn(refetch)}
        noData={{
          when: () => repositories?.length === 0 && searchTerm === undefined,
          image: emptyStateImage,
          message: getString('repos.noDataMessage'),
          button: NewRepoButton
        }}>
        <Container padding="xlarge">
          <Layout.Horizontal spacing="large" className={css.layout}>
            {NewRepoButton}
            <FlexExpander />
            <SearchInputWithSpinner loading={loading} query={searchTerm} setQuery={setSearchTerm} />
          </Layout.Horizontal>
          <Container margin={{ top: 'medium' }}>
            {!!repositories?.length && (
              <Table<TypesRepository>
                className={css.table}
                columns={columns}
                data={repositories || []}
                onRowClick={repoInfo => history.push(routes.toCODERepository({ repoPath: repoInfo.path as string }))}
                getRowClassName={row => cx(css.row, !row.original.description && css.noDesc)}
              />
            )}

            <NoResultCard
              showWhen={() => !!repositories && repositories.length === 0 && !!searchTerm?.length}
              forSearch={true}
            />
          </Container>
          <ResourceListingPagination response={response} page={page} setPage={setPage} />
        </Container>
      </PageBody>
    </Container>
  )
}
