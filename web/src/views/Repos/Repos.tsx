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
import { useStrings } from 'framework/strings'
import { NewRepoModalButton } from './NewRepoModalButton'
import { PinnedRibbon } from './PinnedRibbon'
import chartImg from './chart.svg'
import css from './Repos.module.scss'

// TODO: USE FROM SERVICE (DOES NOT EXIST YET)
interface Repository {
  accountId: string
  name: string
  description?: string
  pinned?: boolean
  public?: boolean
  metadata?: {
    language: string
    branch: number
    commit: number
    fork: number
  }
  activities: string
  updated: number
}

const repodata: Repository[] = [...Array(50).keys()].map(number => ({
  accountId: `Harness`,
  name: `Repo ${number}`,
  description:
    number < 4
      ? 'Repo very long name very long name long long long longRepo very long name very long name long long long long'
      : '',
  data: `Data ${number}`,
  pinned: number < 3,
  public: number < 2,
  metadata:
    number % 2
      ? {
          language: 'Java',
          branch: 1000,
          commit: 12323,
          fork: 2
        }
      : undefined,
  activities: `Activity ${number}`,
  updated: Date.now() - 40000
}))

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
              {'July 13, 2022' || row.original.updated}
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

  useEffect(() => {
    onResize()
    window.addEventListener('resize', onResize)

    return () => {
      window.removeEventListener('resize', onResize)
    }
  }, [])

  const itemCount = repodata.length
  const pageSize = 25
  const pageCount = itemCount

  return (
    <Container className={css.main}>
      <PageHeader title={getString('repositories')} />
      <PageBody>
        <Container padding="xlarge">
          <Layout.Horizontal spacing="large">
            {/* <Button text="New Repository" variation={ButtonVariation.PRIMARY} icon="book" /> */}
            <NewRepoModalButton
              accountIdentifier=""
              orgIdentifier=""
              projectIdentifier=""
              modalTitle="Create Repository"
              text="New Repository"
              variation={ButtonVariation.PRIMARY}
              icon="plus"
              onSubmit={_data => {
                console.log(_data)
              }}
            />
            <FlexExpander />
            <TextInput placeholder="Search" leftIcon="search" style={{ width: 350 }} autoFocus />
          </Layout.Horizontal>
          <Container margin={{ top: 'medium' }}>
            <Table<Repository>
              rowDataTestID={(_, index: number) => `scm-repo-${index}`}
              className={css.table}
              columns={columns}
              data={repodata || []}
              onRowClick={_data => {
                // onPolicyClicked(data)
              }}
              getRowClassName={() => css.row}
            />
          </Container>
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
        </Container>
      </PageBody>
    </Container>
  )
}
