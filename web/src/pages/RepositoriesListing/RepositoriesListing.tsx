/*
 * Copyright 2023 Harness, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import {
  ButtonVariation,
  Container,
  FlexExpander,
  Layout,
  PageBody,
  PageHeader,
  TableV2 as Table,
  Text
} from '@harnessio/uicore'
import { ProgressBar, Intent } from '@blueprintjs/core'
import { Color } from '@harnessio/design-system'
import type { CellProps, Column } from 'react-table'
import Keywords from 'react-keywords'
import cx from 'classnames'
import { useGet } from 'restful-react'
import { useHistory } from 'react-router-dom'
import { useStrings } from 'framework/strings'
import { voidFn, formatDate, getErrorMessage, LIST_FETCHING_LIMIT, PageBrowserProps } from 'utils/Utils'
import { NewRepoModalButton } from 'components/NewRepoModalButton/NewRepoModalButton'
import type { TypesRepository } from 'services/code'
import { usePageIndex } from 'hooks/usePageIndex'
import { useQueryParams } from 'hooks/useQueryParams'
import { useUpdateQueryParams } from 'hooks/useUpdateQueryParams'
import useSpaceSSE from 'hooks/useSpaceSSE'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { SearchInputWithSpinner } from 'components/SearchInputWithSpinner/SearchInputWithSpinner'
import { useAppContext } from 'AppContext'
import { LoadingSpinner } from 'components/LoadingSpinner/LoadingSpinner'
import { NoResultCard } from 'components/NoResultCard/NoResultCard'
import { ResourceListingPagination } from 'components/ResourceListingPagination/ResourceListingPagination'
import { RepoPublicLabel } from 'components/RepoPublicLabel/RepoPublicLabel'
import KeywordSearch from 'components/CodeSearch/KeywordSearch'
import noRepoImage from './no-repo.svg'
import FeatureMap from './FeatureMap/FeatureMap'
import css from './RepositoriesListing.module.scss'

interface TypesRepoExtended extends TypesRepository {
  importing?: boolean
}

export default function RepositoriesListing() {
  const { getString } = useStrings()
  const history = useHistory()
  const rowContainerRef = useRef<HTMLDivElement>(null)
  const [nameTextWidth, setNameTextWidth] = useState(600)
  const space = useGetSpaceParam()
  const [searchTerm, setSearchTerm] = useState<string | undefined>()
  const { routes, standalone } = useAppContext()
  const { updateQueryParams } = useUpdateQueryParams()
  const pageBrowser = useQueryParams<PageBrowserProps>()
  const pageInit = pageBrowser.page ? parseInt(pageBrowser.page) : 1
  const [page, setPage] = usePageIndex(pageInit)

  const {
    data: repositories,
    error,
    loading,
    refetch,
    response
  } = useGet<TypesRepository[]>({
    path: `/api/v1/spaces/${space}/+/repos`,
    queryParams: { page, limit: LIST_FETCHING_LIMIT, query: searchTerm },
    debounce: 500
  })

  const onEvent = useCallback(
    data => {
      // should I include repo id here? what if a new repo is created? coould check for ids that are higher than the lowest id on the page?
      if (repositories?.some(repository => repository.id === data?.id && repository.parent_id === data?.parent_id)) {
        //TODO - revisit full refresh - can I use the message to update the execution?
        refetch()
      }
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [repositories]
  )

  const events = useMemo(() => ['repository_import_completed'], [])

  useSpaceSSE({
    space,
    events,
    onEvent
  })

  useEffect(() => {
    setSearchTerm(undefined)
    if (page > 1) {
      updateQueryParams({ page: page.toString() })
    }
  }, [space, setPage]) // eslint-disable-line react-hooks/exhaustive-deps

  const columns: Column<TypesRepoExtended>[] = useMemo(
    () => [
      {
        Header: getString('repos.name'),
        width: 'calc(100% - 180px)',

        Cell: ({ row }: CellProps<TypesRepoExtended>) => {
          const record = row.original
          return (
            <Container className={css.nameContainer}>
              <Layout.Horizontal spacing="small" style={{ flexGrow: 1 }}>
                <Layout.Vertical flex className={css.name} ref={rowContainerRef}>
                  <Text className={css.repoName} width={nameTextWidth} lineClamp={2}>
                    <Keywords value={searchTerm}>{record.uid}</Keywords>
                    <RepoPublicLabel isPublic={row.original.is_public} margin={{ left: 'small' }} />
                  </Text>

                  {record.importing ? (
                    <Text className={css.desc} width={nameTextWidth} lineClamp={1}>
                      {getString('importProgress')}
                    </Text>
                  ) : (
                    record.description && (
                      <Text className={css.desc} width={nameTextWidth} lineClamp={1}>
                        {record.description}
                      </Text>
                    )
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
        Cell: ({ row }: CellProps<TypesRepoExtended>) => {
          return row.original.importing ? (
            <Layout.Horizontal style={{ alignItems: 'center' }} padding={{ right: 'large' }}>
              <ProgressBar intent={Intent.PRIMARY} className={css.progressBar} />
            </Layout.Horizontal>
          ) : (
            <Layout.Horizontal style={{ alignItems: 'center' }}>
              <Text color={Color.BLACK} lineClamp={1} rightIconProps={{ size: 10 }} width={120}>
                {formatDate(row.original.updated as number)}
              </Text>
            </Layout.Horizontal>
          )
        },
        disableSortBy: true
      }
    ],
    [nameTextWidth, getString, searchTerm]
  )

  const onResize = useCallback(() => {
    if (rowContainerRef.current) {
      setNameTextWidth((rowContainerRef.current.closest('div[role="cell"]') as HTMLDivElement)?.offsetWidth - 100)
    }
  }, [setNameTextWidth])
  const NewRepoButton = (
    <NewRepoModalButton
      space={space}
      modalTitle={getString('createARepo')}
      text={getString('newRepo')}
      variation={ButtonVariation.PRIMARY}
      onSubmit={repoInfo => {
        if (repoInfo.importing) {
          refetch()
        } else if (repoInfo.importing_repos) {
          history.push(routes.toCODERepositories({ space: space as string }))
          refetch()
        } else {
          history.push(routes.toCODERepository({ repoPath: (repoInfo as TypesRepository).path as string }))
        }
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
      <PageHeader title={getString('repositories')} toolbar={standalone ? null : <KeywordSearch />} />
      <PageBody
        className={cx({ [css.withError]: !!error })}
        error={error ? getErrorMessage(error) : null}
        retryOnError={voidFn(refetch)}
        noData={{
          when: () => repositories?.length === 0 && searchTerm === undefined,
          image: noRepoImage,
          message: getString('repos.noDataMessage'),
          button: NewRepoButton
        }}>
        <LoadingSpinner visible={loading && searchTerm === undefined} className={css.spinner} />
        <Layout.Horizontal>
          <Container className={css.repoListingContainer} margin={{ top: 'medium' }}>
            <Container padding="xlarge">
              <Layout.Horizontal spacing="large" className={css.layout}>
                {NewRepoButton}
                <FlexExpander />
                <SearchInputWithSpinner
                  loading={loading && searchTerm !== undefined}
                  query={searchTerm}
                  setQuery={setSearchTerm}
                />
              </Layout.Horizontal>

              {!!repositories?.length && (
                <Table<TypesRepoExtended>
                  className={css.table}
                  columns={columns}
                  data={repositories || []}
                  onRowClick={repoInfo => {
                    return repoInfo.importing
                      ? undefined
                      : history.push(routes.toCODERepository({ repoPath: repoInfo.path as string }))
                  }}
                  getRowClassName={row =>
                    cx(css.row, !row.original.description && css.noDesc, row.original.importing && css.rowDisable)
                  }
                />
              )}

              <NoResultCard
                showWhen={() => !!repositories && repositories.length === 0 && !!searchTerm?.length}
                forSearch={true}
              />
            </Container>
            <ResourceListingPagination response={response} page={page} setPage={setPage} />
          </Container>
          <FeatureMap />
        </Layout.Horizontal>
      </PageBody>
    </Container>
  )
}
