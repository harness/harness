import React, { useMemo, useState } from 'react'
import {
  Button,
  Container,
  ButtonVariation,
  PageBody,
  Text,
  Color,
  TableV2,
  Layout,
  StringSubstitute,
  NoDataCard
} from '@harness/uicore'
import { useHistory } from 'react-router-dom'
import { useGet } from 'restful-react'
import type { CellProps, Column } from 'react-table'
import ReactTimeago from 'react-timeago'
import { CodeIcon, makeDiffRefs, PullRequestFilterOption } from 'utils/GitUtils'
import { useAppContext } from 'AppContext'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { useStrings } from 'framework/strings'
import { RepositoryPageHeader } from 'components/RepositoryPageHeader/RepositoryPageHeader'
import { getErrorMessage, LIST_FETCHING_LIMIT } from 'utils/Utils'
import emptyStateImage from 'images/empty-state.svg'
import { usePageIndex } from 'hooks/usePageIndex'
import type { TypesPullReq } from 'services/code'
import { PullRequestsContentHeader } from './PullRequestsContentHeader/PullRequestsContentHeader'
import prImgOpen from './pull-request-open.svg'
import prImgMerged from './pull-request-merged.svg'
import prImgClosed from './pull-request-closed.svg'
import prImgRejected from './pull-request-rejected.svg'
import css from './PullRequests.module.scss'

export default function PullRequests() {
  const { getString } = useStrings()
  const history = useHistory()
  const { routes } = useAppContext()
  const [searchTerm, setSearchTerm] = useState('')
  const [filter, setFilter] = useState(PullRequestFilterOption.ALL)
  const [pageIndex, setPageIndex] = usePageIndex()
  const { repoMetadata, error, loading, refetch } = useGetRepositoryMetadata()
  const queryParams = useMemo(() => {
    const searchParams = new URLSearchParams({
      limit: String(LIST_FETCHING_LIMIT),
      page: String(pageIndex + 1),
      sort: 'date',
      order: 'desc',
      include_commit: String(true),
      query: searchTerm
    })

    switch (filter) {
      case PullRequestFilterOption.OPEN:
      case PullRequestFilterOption.MERGED:
      case PullRequestFilterOption.CLOSED:
      case PullRequestFilterOption.REJECTED:
        searchParams.append('state', filter)
        break
      default:
        searchParams.append('state', PullRequestFilterOption.OPEN)
        searchParams.append('state', PullRequestFilterOption.MERGED)
        searchParams.append('state', PullRequestFilterOption.CLOSED)
        searchParams.append('state', PullRequestFilterOption.REJECTED)
        break
    }

    return searchParams.toString()
  }, [searchTerm, pageIndex, filter])
  const {
    data,
    error: prError,
    loading: prLoading
  } = useGet<TypesPullReq[]>({
    path: `/api/v1/repos/${repoMetadata?.path}/+/pullreq?${queryParams}`,
    lazy: !repoMetadata
  })
  const columns: Column<TypesPullReq>[] = useMemo(
    () => [
      {
        id: 'title',
        width: '100%',
        Cell: ({ row }: CellProps<TypesPullReq>) => {
          return (
            <Layout.Horizontal spacing="medium" padding={{ left: 'medium' }}>
              <img src={stateToImage(row.original)} title={row.original.state} />
              <Container padding={{ left: 'small' }}>
                <Layout.Vertical spacing="small">
                  <Text icon="success-tick" color={Color.GREY_800} className={css.title}>
                    {row.original.title}
                  </Text>
                  <Text color={Color.GREY_500}>
                    <StringSubstitute
                      str={getString('pr.statusLine')}
                      vars={{
                        state: row.original.state,
                        number: <Text inline>{row.original.number}</Text>,
                        time: <ReactTimeago date={row.original.created as number} />,
                        user: row.original.author?.display_name
                      }}
                    />
                  </Text>
                </Layout.Vertical>
              </Container>
            </Layout.Horizontal>
          )
        }
      }
    ],
    [getString]
  )

  return (
    <Container className={css.main}>
      <RepositoryPageHeader
        repoMetadata={repoMetadata}
        title={getString('pullRequests')}
        dataTooltipId="repositoryPullRequests"
      />
      <PageBody loading={loading} error={getErrorMessage(error || prError)} retryOnError={() => refetch()}>
        {repoMetadata && (
          <Layout.Vertical>
            <PullRequestsContentHeader
              loading={prLoading}
              repoMetadata={repoMetadata}
              onPullRequestFilterChanged={_filter => {
                setFilter(_filter)
                setPageIndex(0)
              }}
              onSearchTermChanged={value => {
                setSearchTerm(value)
                setPageIndex(0)
              }}
            />
            <Container padding="xlarge">
              {!!data?.length && (
                <TableV2<TypesPullReq>
                  className={css.table}
                  hideHeaders
                  columns={columns}
                  data={data}
                  getRowClassName={() => css.row}
                  onRowClick={row => {
                    history.push(
                      routes.toCODEPullRequest({
                        repoPath: repoMetadata.path as string,
                        pullRequestId: String(row.number)
                      })
                    )
                  }}
                />
              )}
              {data?.length === 0 && (
                <Container className={css.noData}>
                  <NoDataCard
                    image={emptyStateImage}
                    message={getString('pullRequestEmpty')}
                    button={
                      <Button
                        variation={ButtonVariation.PRIMARY}
                        text={getString('newPullRequest')}
                        icon={CodeIcon.Add}
                        onClick={() => {
                          history.push(
                            routes.toCODECompare({
                              repoPath: repoMetadata?.path as string,
                              diffRefs: makeDiffRefs(repoMetadata?.default_branch as string, '')
                            })
                          )
                        }}
                      />
                    }
                  />
                </Container>
              )}
            </Container>
          </Layout.Vertical>
        )}
      </PageBody>
    </Container>
  )
}

const stateToImage = (pr: TypesPullReq) => {
  switch (pr.state) {
    case PullRequestFilterOption.OPEN:
      return prImgOpen
    case PullRequestFilterOption.MERGED:
      return prImgMerged
    case PullRequestFilterOption.CLOSED:
      return prImgClosed
    case PullRequestFilterOption.REJECTED:
      return prImgRejected
  }

  return prImgClosed
}
