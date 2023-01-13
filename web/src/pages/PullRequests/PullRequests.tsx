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
import cx from 'classnames'
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
import prImgDraft from './pull-request-draft.svg'
import css from './PullRequests.module.scss'

export default function PullRequests() {
  const { getString } = useStrings()
  const history = useHistory()
  const { routes } = useAppContext()
  const [searchTerm, setSearchTerm] = useState('')
  const [filter, setFilter] = useState(PullRequestFilterOption.OPEN)
  const [pageIndex, setPageIndex] = usePageIndex()
  const { repoMetadata, error, loading, refetch } = useGetRepositoryMetadata()
  const {
    data,
    error: prError,
    loading: prLoading
  } = useGet<TypesPullReq[]>({
    path: `/api/v1/repos/${repoMetadata?.path}/+/pullreq`,
    queryParams: {
      limit: String(LIST_FETCHING_LIMIT),
      page: String(pageIndex + 1),
      sort: filter == PullRequestFilterOption.MERGED ? 'merged' : 'number',
      order: 'desc',
      query: searchTerm,
      state: filter == PullRequestFilterOption.ALL ? '' : filter
    },
    lazy: !repoMetadata
  })
  const columns: Column<TypesPullReq>[] = useMemo(
    () => [
      {
        id: 'title',
        width: '100%',
        Cell: ({ row }: CellProps<TypesPullReq>) => {
          return (
            <Layout.Horizontal className={css.titleRow} spacing="medium">
              <img {...stateToImageProps(row.original)} />
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
                        time: <ReactTimeago date={(row.original.state == 'merged' ? row.original.merged : row.original.created) as number} />,
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

const stateToImageProps = (pr: TypesPullReq) => {
  let src = prImgClosed
  let clazz = css.open

  switch (pr.state) {
    case PullRequestFilterOption.OPEN:
      src = prImgOpen
      clazz = css.open
      break
    case PullRequestFilterOption.MERGED:
      src = prImgMerged
      clazz = css.merged
      break
    case PullRequestFilterOption.CLOSED:
      src = prImgClosed
      clazz = css.closed
      break
    case PullRequestFilterOption.REJECTED:
      src = prImgRejected
      clazz = css.rejected
      break
    case PullRequestFilterOption.DRAFT:
      src = prImgDraft
      clazz = css.draft
      break
  }

  return { src, title: pr.state, className: cx(css.rowImg, clazz) }
}
