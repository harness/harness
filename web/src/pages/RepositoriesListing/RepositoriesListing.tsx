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
  Icon
} from '@harness/uicore'
import type { CellProps, Column } from 'react-table'
import cx from 'classnames'
import { useGet } from 'restful-react'
import { useHistory } from 'react-router-dom'
import { useStrings } from 'framework/strings'
import { formatDate, getErrorMessage, X_PER_PAGE, X_TOTAL, X_TOTAL_PAGES } from 'utils/Utils'
import { NewRepoModalButton } from 'components/NewRepoModalButton/NewRepoModalButton'
import type { RepositoryDTO } from 'types/SCMTypes'
import { useAppContext } from 'AppContext'
import emptyStateImage from './images/empty-state.svg'
import css from './RepositoriesListing.module.scss'
// import { useListRepos } from 'services/scm'

export default function Repos(): JSX.Element {
  const { getString } = useStrings()
  const history = useHistory()
  const ref = useRef<HTMLDivElement>(null)
  const [nameTextWidth, setNameTextWidth] = useState(600)
  const { space = '', routes } = useAppContext() // TODO: Proper handling `space` for standalone version
  const {
    data: repositories,
    error,
    loading,
    refetch,
    response
  } = useGet<RepositoryDTO[]>({
    path: `/api/v1/spaces/${space}/+/repos`
  })

  // DOES NOT WORK !!! API URL IS NOT CONSTRUCTED PROPERLY
  // const r = useListRepos({
  //   spaceRef: space
  // })
  // console.log({ r })

  const itemCount = useMemo(() => parseInt(response?.headers?.get(X_TOTAL) || '0'), [response])
  const pageCount = useMemo(() => parseInt(response?.headers?.get(X_TOTAL_PAGES) || '0'), [response])
  const pageSize = useMemo(() => parseInt(response?.headers?.get(X_PER_PAGE) || '0'), [response])

  const columns: Column<RepositoryDTO>[] = useMemo(
    () => [
      {
        Header: getString('repos.name'),
        accessor: row => row.name,
        width: '75%',
        Cell: ({ row }: CellProps<RepositoryDTO>) => {
          const record = row.original
          return (
            <Container className={css.nameContainer}>
              <Layout.Horizontal spacing="small" style={{ flexGrow: 1 }}>
                <Layout.Vertical flex className={css.name} ref={ref}>
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
      // {
      //   Header: getString('status'), // TODO: Status is not yet supported by backend
      //   id: 'status',
      //   accessor: row => row.updated,
      //   width: '10%',
      //   Cell: () => <Icon name="success-tick" padding={{ left: 'small' }} />,
      //   disableSortBy: true
      // },
      {
        Header: getString('repos.updated'),
        id: 'menu',
        accessor: row => row.updated,
        width: '15%',
        Cell: ({ row }: CellProps<RepositoryDTO>) => {
          return (
            <Layout.Horizontal style={{ alignItems: 'center' }}>
              <Text color={Color.BLACK} lineClamp={1} rightIconProps={{ size: 10 }} width={120}>
                {formatDate(row.original.updated)}
              </Text>
              {row.original.isPublic === false ? <Icon name="lock" size={10} /> : undefined}
            </Layout.Horizontal>
          )
        },
        disableSortBy: true
      }
    ],
    [history, module, nameTextWidth] // eslint-disable-line react-hooks/exhaustive-deps
  )
  const [pageIndex, setPageIndex] = useState(0)
  const onResize = useCallback((): void => {
    if (ref.current) {
      setNameTextWidth((ref.current.closest('div[role="cell"]') as HTMLDivElement)?.offsetWidth - 100)
    }
  }, [setNameTextWidth])
  const NewRepoButton = (
    <NewRepoModalButton
      space={space}
      modalTitle={getString('newRepo')}
      text={getString('newRepo')}
      variation={ButtonVariation.PRIMARY}
      icon="plus"
      onSubmit={_data => {
        // TODO: Remove this when backend fixes https://harness.slack.com/archives/C03Q1Q4C9J8/p1666521412586789
        const accountId = 'kmpySmUISimoRrJL6NL73w'
        const [_accountId, orgIdentifier, projectIdentifier, repoName] = _data.path.split('/')
        const url = routes.toSCMRepository({
          repoPath: [accountId, orgIdentifier, projectIdentifier, repoName].join('/')
        })
        console.log({ _data, url, accountId, _accountId })
        history.push(
          routes.toSCMRepository({ repoPath: [accountId, orgIdentifier, projectIdentifier, repoName].join('/') })
        )
      }}
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
        loading={loading}
        className={cx({ [css.withError]: !!error })}
        error={error ? getErrorMessage(error) : null}
        retryOnError={() => {
          refetch()
        }}
        noData={{
          when: () => repositories?.length === 0,
          image: emptyStateImage,
          message: getString('repos.noDataMessage'),
          button: NewRepoButton
        }}>
        <Container padding="xlarge">
          <Layout.Horizontal spacing="large">
            {NewRepoButton}
            <FlexExpander />
            {/* TODO: Search is not yet supported by backend */}
            {/* <TextInput placeholder={getString('search')} leftIcon="search" style={{ width: 350 }} autoFocus /> */}
          </Layout.Horizontal>
          <Container margin={{ top: 'medium' }}>
            <Table<RepositoryDTO>
              rowDataTestID={(_, index: number) => `scm-repo-${index}`}
              className={css.table}
              columns={columns}
              data={repositories || []}
              onRowClick={data => {
                // TODO: Remove this when backend fixes https://harness.slack.com/archives/C03Q1Q4C9J8/p1666521412586789
                const accountId = 'kmpySmUISimoRrJL6NL73w'
                const [_accountId, orgIdentifier, projectIdentifier, repoName] = data.path.split('/')
                const url = routes.toSCMRepository({
                  repoPath: [accountId, orgIdentifier, projectIdentifier, repoName].join('/')
                })
                console.info({ data, url, accountId, _accountId })
                history.push(url)
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
                pageSizeOptions={[5, 10, 20, 40]}
              />
            </Container>
          )}
        </Container>
      </PageBody>
    </Container>
  )
}
