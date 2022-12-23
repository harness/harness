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
  StringSubstitute
} from '@harness/uicore'
import { useHistory } from 'react-router-dom'
import { useGet } from 'restful-react'
import type { CellProps, Column } from 'react-table'
import ReactTimeago from 'react-timeago'
import { CodeIcon, makeDiffRefs } from 'utils/GitUtils'
import { useAppContext } from 'AppContext'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { useStrings } from 'framework/strings'
import { RepositoryPageHeader } from 'components/RepositoryPageHeader/RepositoryPageHeader'
import { getErrorMessage, LIST_FETCHING_PER_PAGE } from 'utils/Utils'
import emptyStateImage from 'images/empty-state.svg'
import { usePageIndex } from 'hooks/usePageIndex'
import { PullRequestsContentHeader } from './PullRequestsContentHeader/PullRequestsContentHeader'
import prOpenImg from './pull-request-open.svg'
import css from './PullRequests.module.scss'
import type { TypesPullReq } from 'services/code'

export default function PullRequests() {
  const { getString } = useStrings()
  const history = useHistory()
  const { routes } = useAppContext()
  const [searchTerm, setSearchTerm] = useState('')
  const [pageIndex, setPageIndex] = usePageIndex()
  const { repoMetadata, error, loading, refetch } = useGetRepositoryMetadata()
  const {
    data,
    error: prError,
    loading: prLoading
  } = useGet<TypesPullReq[]>({
    path: `/api/v1/repos/${repoMetadata?.path}/+/pullreq`,
    queryParams: {
      per_page: LIST_FETCHING_PER_PAGE,
      page: pageIndex + 1,
      sort: 'date',
      direction: 'desc',
      include_commit: true,
      query: searchTerm
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
            <Layout.Horizontal spacing="medium" padding={{ left: 'medium' }}>
              <img src={prOpenImg} />
              <Container padding={{ left: 'small' }}>
                <Layout.Vertical spacing="small">
                  <Text icon="success-tick" color={Color.GREY_800} className={css.title}>
                    {row.original.title}
                  </Text>
                  <Text color={Color.GREY_500}>
                    <StringSubstitute
                      str={getString('pr.openBy')}
                      vars={{
                        number: <Text inline>{row.original.number}</Text>,
                        time: <ReactTimeago date={row.original.created!} />,
                        user: row.original.author?.name
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
      <PageBody
        loading={loading || prLoading}
        error={getErrorMessage(error || prError)}
        retryOnError={() => refetch()}
        noData={{
          // TODO: Use NoDataCard, this won't render toolbar
          // when search returns empty response
          when: () => data?.length === 0,
          message: getString('pullRequestEmpty'),
          image: emptyStateImage,
          button: (
            <Button
              variation={ButtonVariation.PRIMARY}
              text={getString('newPullRequest')}
              icon={CodeIcon.Add}
              onClick={() => {
                history.push(
                  routes.toCODECompare({
                    repoPath: repoMetadata?.path as string,
                    diffRefs: makeDiffRefs(repoMetadata?.defaultBranch as string, '')
                  })
                )
              }}
            />
          )
        }}>
        {repoMetadata && (
          <Layout.Vertical>
            <PullRequestsContentHeader
              repoMetadata={repoMetadata}
              onPullRequestFilterChanged={_filter => {
                setPageIndex(0)
              }}
              onSearchTermChanged={value => {
                setSearchTerm(value)
                setPageIndex(0)
              }}
            />
            {!!data?.length && (
              <Container padding="xlarge">
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
              </Container>
            )}
          </Layout.Vertical>
        )}
      </PageBody>
    </Container>
  )
}
