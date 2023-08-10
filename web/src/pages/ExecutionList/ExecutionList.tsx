import React, { useMemo, useState } from 'react'
import {
  Button,
  ButtonVariation,
  Color,
  Container,
  FlexExpander,
  Icon,
  Layout,
  PageBody,
  PageHeader,
  TableV2 as Table,
  Text
} from '@harness/uicore'
import cx from 'classnames'
import type { CellProps, Column } from 'react-table'
import Keywords from 'react-keywords'
import { useHistory } from 'react-router-dom'
import { useStrings } from 'framework/strings'
import { LoadingSpinner } from 'components/LoadingSpinner/LoadingSpinner'
import { useAppContext } from 'AppContext'
import { SearchInputWithSpinner } from 'components/SearchInputWithSpinner/SearchInputWithSpinner'
import { NoResultCard } from 'components/NoResultCard/NoResultCard'
import { formatDate } from 'utils/Utils'
import noExecutionImage from '../RepositoriesListing/no-repo.svg'
import css from './ExecutionList.module.scss'

interface Execution {
  id: number
  uid: string
  name: string
  updated: number
  description?: string
  isPublic?: boolean
  spaceUid: string
  pipelineUid: string
}

const executions: Execution[] = [
  {
    id: 1,
    uid: '1',
    name: 'Exec 1',
    updated: 1687234800,
    description: 'This is a description',
    isPublic: true,
    spaceUid: 'root',
    pipelineUid: 'pipeline-1'
  },
  {
    id: 2,
    uid: '2',
    name: 'Exec 2',
    updated: 1730275200,
    description: 'This is a description',
    isPublic: true,
    spaceUid: 'root',
    pipelineUid: 'pipeline-2'
  },
  {
    id: 3,
    uid: '3',
    name: 'Exec 3',
    updated: 1773315600,
    description: 'This is a description',
    isPublic: false,
    spaceUid: 'root',
    pipelineUid: 'pipeline-3'
  }
]

const loading = false

const ExecutionList = () => {
  const { routes } = useAppContext()
  const history = useHistory()
  const { getString } = useStrings()
  const [searchTerm, setSearchTerm] = useState<string | undefined>()

  const NewExecutionButton = (
    <Button
      text={getString('executions.newExecutionButton')}
      variation={ButtonVariation.PRIMARY}
      disabled={true}
      icon="plus"></Button>
  )

  const columns: Column<Execution>[] = useMemo(
    () => [
      {
        Header: getString('repos.name'),
        width: 'calc(100% - 180px)',
        Cell: ({ row }: CellProps<Execution>) => {
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
        Cell: ({ row }: CellProps<Execution>) => {
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
    [getString, searchTerm]
  )

  return (
    <Container className={css.main}>
      <PageHeader title={getString('pageTitle.executions')} />
      <PageBody
        noData={{
          when: () => executions.length === 0,
          image: noExecutionImage,
          message: getString('executions.noData'),
          button: NewExecutionButton
        }}>
        <LoadingSpinner visible={loading && !searchTerm} />

        <Container padding="xlarge">
          <Layout.Horizontal spacing="large" className={css.layout}>
            {NewExecutionButton}
            <FlexExpander />
            <SearchInputWithSpinner loading={loading} query={searchTerm} setQuery={setSearchTerm} />
          </Layout.Horizontal>

          <Container margin={{ top: 'medium' }}>
            {!!executions?.length && (
              <Table<Execution>
                className={css.table}
                columns={columns}
                data={executions || []}
                onRowClick={executionInfo =>
                  history.push(
                    routes.toCODEExecution({
                      space: executionInfo.spaceUid,
                      pipeline: executionInfo.pipelineUid,
                      execution: executionInfo.uid
                    })
                  )
                }
                getRowClassName={row => cx(css.row, !row.original.description && css.noDesc)}
              />
            )}

            <NoResultCard showWhen={() => !!executions.length && !!searchTerm?.length} forSearch={true} />
          </Container>
          {/* <ResourceListingPagination response={response} page={page} setPage={setPage} /> */}
        </Container>
      </PageBody>
    </Container>
  )
}

export default ExecutionList
