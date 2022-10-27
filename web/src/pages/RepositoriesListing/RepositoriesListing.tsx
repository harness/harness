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
  Pagination,
  Icon,
  TextInput
} from '@harness/uicore'
import type { CellProps, Column } from 'react-table'
import cx from 'classnames'
import { useGet } from 'restful-react'
import { useHistory, useParams } from 'react-router-dom'
import { useStrings } from 'framework/strings'
import { formatDate, getErrorMessage, LIST_FETCHING_PER_PAGE, X_PER_PAGE, X_TOTAL, X_TOTAL_PAGES } from 'utils/Utils'
import { NewRepoModalButton } from 'components/NewRepoModalButton/NewRepoModalButton'
import type { TypesRepository } from 'services/scm'
import type { SCMPathProps } from 'RouteDefinitions'
import { useAppContext } from 'AppContext'
import emptyStateImage from './images/empty-state.svg'
import css from './RepositoriesListing.module.scss'

export default function RepositoriesListing(): JSX.Element {
  const { getString } = useStrings()
  const history = useHistory()
  const rowContainerRef = useRef<HTMLDivElement>(null)
  const [nameTextWidth, setNameTextWidth] = useState(600)
  const [pageIndex, setPageIndex] = useState(0)
  const params = useParams<SCMPathProps>()
  const [query, setQuery] = useState<string | undefined>()
  const { space = params.space || '', routes } = useAppContext()
  const path = useMemo(
    () =>
      `/api/v1/spaces/${space}/+/repos?page=${pageIndex + 1}&per_page=${LIST_FETCHING_PER_PAGE}${
        query ? `&query=${query}` : ''
      }`,
    [space, query, pageIndex]
  )
  const { data: repositories, error, loading, refetch, response } = useGet<TypesRepository[]>({ path })
  const itemCount = useMemo(() => parseInt(response?.headers?.get(X_TOTAL) || '0'), [response])
  const pageCount = useMemo(() => parseInt(response?.headers?.get(X_TOTAL_PAGES) || '0'), [response])
  const pageSize = useMemo(() => parseInt(response?.headers?.get(X_PER_PAGE) || '0'), [response])

  useEffect(() => {
    setQuery(undefined)
    setPageIndex(0)
  }, [space])

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
                    {record.name}
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
              {row.original.isPublic === false ? <Icon name="lock" size={10} /> : undefined}
            </Layout.Horizontal>
          )
        },
        disableSortBy: true
      }
    ],
    [nameTextWidth, getString]
  )
  const onResize = useCallback((): void => {
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
      onSubmit={repoInfo => history.push(routes.toSCMRepository({ repoPath: repoInfo.path as string }))}
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
        loading={loading && query === undefined}
        className={cx({ [css.withError]: !!error })}
        error={error ? getErrorMessage(error) : null}
        retryOnError={() => refetch()}
        noData={{
          when: () => repositories?.length === 0 && query === undefined,
          image: emptyStateImage,
          message: getString('repos.noDataMessage'),
          button: NewRepoButton
        }}>
        <Container padding="xlarge">
          <Layout.Horizontal spacing="large">
            {NewRepoButton}
            <FlexExpander />
            <TextInput
              placeholder={getString('search')}
              leftIcon={loading && query !== undefined ? 'steps-spinner' : 'search'}
              style={{ width: 250 }}
              autoFocus
              onInput={event => {
                setQuery(event.currentTarget.value || '')
                setPageIndex(0)
              }}
            />
          </Layout.Horizontal>
          <Container margin={{ top: 'medium' }}>
            <Table<TypesRepository>
              rowDataTestID={(_, index: number) => `scm-repo-${index}`}
              className={css.table}
              columns={columns}
              data={repositories || []}
              onRowClick={repoInfo => {
                history.push(routes.toSCMRepository({ repoPath: repoInfo.path as string }))
              }}
              getRowClassName={row => cx(css.row, !row.original.description && css.noDesc)}
            />
          </Container>
          {!!repositories?.length && (
            <Container margin={{ bottom: 'medium', left: 'xxxlarge', right: 'xxxlarge' }}>
              <Pagination
                className={css.pagination}
                hidePageNumbers
                gotoPage={index => setPageIndex(index)}
                itemCount={itemCount}
                pageCount={pageCount}
                pageIndex={pageIndex}
                pageSize={pageSize}
              />
            </Container>
          )}
        </Container>
      </PageBody>
    </Container>
  )
}
