import React, { useEffect, useMemo, useRef, useState } from 'react'
import {
  ButtonVariation,
  Container,
  FlexExpander,
  Layout,
  PageBody,
  PageHeader,
  TextInput,
  TableV2 as Table,
  Text,
  Color,
  Pagination
} from '@harness/uicore'
import type { CellProps, Column } from 'react-table'
import cx from 'classnames'
import { useGet } from 'restful-react'
import { useStrings } from 'framework/strings'
import { formatDate, getErrorMessage, X_PER_PAGE, X_TOTAL, X_TOTAL_PAGES } from 'utils/Utils'
import type { Repository } from 'types/Repository'
import { NewRepoModalButton } from './NewRepoModalButton'
import { PinnedRibbon } from './PinnedRibbon'
import chartImg from './chart.svg'
import emptyStateImage from './empty_state.png'
import css from './Repos.module.scss'

function RepoMetadata(): JSX.Element {
  return (
    <Container width="70%">
      <Layout.Horizontal spacing="large">
        <Text icon="dot" iconProps={{ size: 20, color: Color.BLUE_500 }}>
          Java
        </Text>
        <Text color={Color.GREY_200}>{' | '}</Text>
        <Text icon="git-new-branch">165</Text>
        <Text icon="git-branch-existing">123</Text>
        <Text icon="git-merge">432</Text>
      </Layout.Horizontal>
    </Container>
  )
}

export default function Repos(): JSX.Element {
  const { getString } = useStrings()
  const ref = useRef<HTMLDivElement>(null)
  const [nameTextWidth, setNameTextWidth] = useState(300)
  const space = 3
  const {
    data: repositories,
    error,
    loading,
    refetch,
    response
  } = useGet<Repository[]>({
    path: `/api/v1/spaces/${space}/repos`
  })
  const itemCount = useMemo(() => parseInt(response?.headers?.get(X_TOTAL) || '0'), [response])
  const pageCount = useMemo(() => parseInt(response?.headers?.get(X_TOTAL_PAGES) || '0'), [response])
  const pageSize = useMemo(() => parseInt(response?.headers?.get(X_PER_PAGE) || '0'), [response])

  const columns: Column<Repository>[] = useMemo(
    () => [
      {
        Header: getString('repos.name'),
        accessor: row => row.name,
        width: '40%',
        Cell: ({ row }: CellProps<Repository>) => {
          const record = row.original
          return (
            <Container className={css.nameContainer}>
              {record.pinned && <PinnedRibbon />}
              <Layout.Horizontal spacing="small" style={{ flexGrow: 1, paddingLeft: 'var(--spacing-xlarge)' }}>
                <Layout.Vertical flex className={css.name} ref={ref}>
                  <Text className={css.repoName} width={nameTextWidth} lineClamp={2}>
                    {record.name}
                    <Text inline className={css.repoScope}>
                      {record.public ? getString('public') : getString('private')}
                    </Text>
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
        Header: getString('repos.data'),
        accessor: row => row.metadata,
        width: '30%',
        Cell: ({ row }: CellProps<Repository>) => {
          // const record = row.original
          return <RepoMetadata />
        }
      },
      {
        Header: getString('repos.activities'),
        accessor: row => row.activities,
        width: '23%',
        Cell: ({ row }: CellProps<Repository>) => {
          const record = row.original
          return (
            <Text color={Color.BLACK} lineClamp={1}>
              {/*record.activities || 'Activities...'*/}
              <img width="183" height="47" src={chartImg} />
            </Text>
          )
        }
      },
      {
        Header: getString('repos.updated'),
        id: 'menu',
        accessor: row => row.updated,
        width: '12%',
        Cell: ({ row }: CellProps<Repository>) => {
          return (
            <Text color={Color.BLACK} lineClamp={1}>
              {formatDate(row.original.updated)}
            </Text>
          )
        },
        disableSortBy: true
      }
    ],
    [history, module, nameTextWidth] // eslint-disable-line react-hooks/exhaustive-deps
  )
  const [pageIndex, setPageIndex] = useState(0)
  const onResize = (): void => {
    if (ref.current) {
      setNameTextWidth((ref.current.closest('div[role="cell"]') as HTMLDivElement)?.offsetWidth - 100)
    }
  }
  const NewRepoButton = (
    <NewRepoModalButton
      space={space}
      modalTitle={getString('newRepo')}
      text={getString('newRepo')}
      variation={ButtonVariation.PRIMARY}
      icon="plus"
      onSubmit={_data => {
        // TODO: navigate to Repo instead of refetch
        refetch()
      }}
    />
  )

  useEffect(() => {
    onResize()
    window.addEventListener('resize', onResize)
    return () => {
      window.removeEventListener('resize', onResize)
    }
  }, [])

  console.log({ response, repositories, pageCount, pageIndex, pageSize, itemCount })

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
          messageTitle: getString('repos.noDataTitle'),
          message: getString('repos.noDataMessage'),
          button: NewRepoButton
        }}>
        <Container padding="xlarge">
          <Layout.Horizontal spacing="large">
            {NewRepoButton}
            <FlexExpander />
            <TextInput placeholder={getString('search')} leftIcon="search" style={{ width: 350 }} autoFocus />
          </Layout.Horizontal>
          <Container margin={{ top: 'medium' }}>
            <Table<Repository>
              rowDataTestID={(_, index: number) => `scm-repo-${index}`}
              className={css.table}
              columns={columns}
              data={repositories || []}
              onRowClick={_data => {
                // onPolicyClicked(data)
              }}
              getRowClassName={() => css.row}
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
