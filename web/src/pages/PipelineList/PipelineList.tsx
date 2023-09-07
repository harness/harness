import React, { useMemo, useState } from 'react'
import { Classes, Menu, MenuItem, Popover, Position } from '@blueprintjs/core'
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
import { Link, useHistory } from 'react-router-dom'
import { useGet } from 'restful-react'
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
import { ExecutionStatus, ExecutionState } from 'components/ExecutionStatus/ExecutionStatus'
import { getStatus } from 'utils/PipelineUtils'
import { PipeSeparator } from 'components/PipeSeparator/PipeSeparator'
import noPipelineImage from '../RepositoriesListing/no-repo.svg'
import css from './PipelineList.module.scss'
import useNewPipelineModal from 'pages/AddUpdatePipeline/NewPipelineModal'

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
    lazy: !repoMetadata,
    debounce: 500
  })

  const { openModal } = useNewPipelineModal()

  const NewPipelineButton = (
    <Button
      text={getString('pipelines.newPipelineButton')}
      variation={ButtonVariation.PRIMARY}
      icon="plus"
      onClick={() => {
        openModal({ repoMetadata })
      }}
      disabled={loading}
    />
  )

  const columns: Column<TypesPipeline>[] = useMemo(
    () => [
      {
        Header: getString('pipelines.name'),
        width: 'calc(100% - 210px)',
        Cell: ({ row }: CellProps<TypesPipeline>) => {
          const record = row.original
          return (
            <Layout.Horizontal spacing="small" className={css.nameContainer}>
              <ExecutionStatus
                status={getStatus(record?.execution?.status || ExecutionState.PENDING)}
                iconOnly
                noBackground
                iconSize={24}
                className={css.statusIcon}
              />
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
            <Layout.Vertical spacing={'small'}>
              <Layout.Horizontal spacing={'small'} style={{ alignItems: 'center' }}>
                <Text className={css.desc}>{`#${record.number}`}</Text>
                <PipeSeparator height={7} />
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
                <PipeSeparator height={7} />
                <GitFork height={12} width={12} color={Utils.getRealCSSColor(Color.GREY_500)} />
                <Text className={css.author}>{record.source}</Text>
                <PipeSeparator height={7} />
                <Link
                  to={routes.toCODECommit({
                    repoPath: repoMetadata?.path as string,
                    commitRef: record.after as string
                  })}
                  className={css.hash}
                  onClick={e => {
                    e.stopPropagation()
                  }}>
                  {/* {record.after?.slice(0, 6)} */}
                  {'hardcoded'.slice(0, 6)}
                </Link>
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
      },
      {
        Header: ' ',
        width: '30px',
        Cell: ({ row }: CellProps<TypesPipeline>) => {
          const [menuOpen, setMenuOpen] = useState(false)
          const record = row.original
          const { uid } = record
          return (
            <Popover
              isOpen={menuOpen}
              onInteraction={nextOpenState => {
                setMenuOpen(nextOpenState)
              }}
              className={Classes.DARK}
              position={Position.BOTTOM_RIGHT}>
              <Button
                variation={ButtonVariation.ICON}
                icon="Options"
                data-testid={`menu-${record.uid}`}
                onClick={e => {
                  e.stopPropagation()
                  setMenuOpen(true)
                }}
              />
              <Menu>
                <MenuItem
                  icon="edit"
                  text={getString('edit')}
                  onClick={e => {
                    e.stopPropagation()
                    history.push(
                      routes.toCODEPipelineEdit({ repoPath: repoMetadata?.path || '', pipeline: uid as string })
                    )
                  }}
                />
              </Menu>
            </Popover>
          )
        }
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
