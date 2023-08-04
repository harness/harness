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
import { SearchInputWithSpinner } from 'components/SearchInputWithSpinner/SearchInputWithSpinner'
import { NoResultCard } from 'components/NoResultCard/NoResultCard'
import { formatDate } from 'utils/Utils'
import { routes } from 'RouteDefinitions'
import noPipelineImage from '../RepositoriesListing/no-repo.svg'
import css from './PipelineList.module.scss'

interface Pipeline {
  id: number
  uid: string
  name: string
  updated: number
  description?: string
  isPublic?: boolean
  spaceUid: string
}

const pipelines: Pipeline[] = [
  {
    id: 1,
    uid: 'pipeline-1',
    name: 'Pipeline 1',
    updated: 1687234800,
    description: 'This is a description',
    isPublic: true,
    spaceUid: 'root'
  },
  {
    id: 2,
    uid: 'pipeline-2',
    name: 'Pipeline 2',
    updated: 1730275200,
    description: 'This is a description',
    isPublic: true,
    spaceUid: 'root'
  },
  {
    id: 3,
    uid: 'pipeline-3',
    name: 'Pipeline 3',
    updated: 1773315600,
    description: 'This is a description',
    isPublic: false,
    spaceUid: 'root'
  }
]

const loading = false

const PipelineList = () => {
  const history = useHistory()
  const { getString } = useStrings()
  const [searchTerm, setSearchTerm] = useState<string | undefined>()

  const NewPipelineButton = (
    <Button
      text={getString('pipelines.newPipelineButton')}
      variation={ButtonVariation.PRIMARY}
      disabled={true}
      icon="plus"></Button>
  )

  const columns: Column<Pipeline>[] = useMemo(
    () => [
      {
        Header: getString('pipelines.name'),
        width: 'calc(100% - 180px)',
        Cell: ({ row }: CellProps<Pipeline>) => {
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
        Cell: ({ row }: CellProps<Pipeline>) => {
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
      <PageHeader title={getString('pageTitle.pipelines')} />
      <PageBody
        noData={{
          when: () => pipelines.length === 0,
          image: noPipelineImage,
          message: getString('pipelines.noData'),
          button: NewPipelineButton
        }}>
        <LoadingSpinner visible={loading && !searchTerm} />

        <Container padding="xlarge">
          <Layout.Horizontal spacing="large" className={css.layout}>
            {NewPipelineButton}
            <FlexExpander />
            <SearchInputWithSpinner loading={loading} query={searchTerm} setQuery={setSearchTerm} />
          </Layout.Horizontal>

          <Container margin={{ top: 'medium' }}>
            {!!pipelines?.length && (
              <Table<Pipeline>
                className={css.table}
                columns={columns}
                data={pipelines || []}
                onRowClick={pipelineInfo =>
                  history.push(routes.toCODEExecutions({ space: pipelineInfo.spaceUid, pipeline: pipelineInfo.uid }))
                }
                getRowClassName={row => cx(css.row, !row.original.description && css.noDesc)}
              />
            )}

            <NoResultCard showWhen={() => !!pipelines.length && !!searchTerm?.length} forSearch={true} />
          </Container>
          {/* <ResourceListingPagination response={response} page={page} setPage={setPage} /> */}
        </Container>
      </PageBody>
    </Container>
  )
}

export default PipelineList
