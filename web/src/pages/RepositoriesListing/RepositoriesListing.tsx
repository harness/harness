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
  Utils,
  TableV2 as Table,
  Text,
  useToaster
} from '@harnessio/uicore'
import { ProgressBar, Intent } from '@blueprintjs/core'
import { Color, FontVariation } from '@harnessio/design-system'
import type { CellProps, Column } from 'react-table'
import Keywords from 'react-keywords'
import cx from 'classnames'
import { useGet } from 'restful-react'
import { useHistory } from 'react-router-dom'
import { useStrings, String } from 'framework/strings'
import { voidFn, formatDate, getErrorMessage, LIST_FETCHING_LIMIT, PageBrowserProps } from 'utils/Utils'
import { NewRepoModalButton } from 'components/NewRepoModalButton/NewRepoModalButton'
import type { TypesRepository } from 'services/code'
import { useDeleteRepository } from 'services/code'
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
import { OptionsMenuButton } from 'components/OptionsMenuButton/OptionsMenuButton'
import { useConfirmAct } from 'hooks/useConfirmAction'
import { getUsingFetch, getConfig } from 'services/config'
import noRepoImage from './no-repo.svg?url'
import css from './RepositoriesListing.module.scss'

interface TypesRepoExtended extends TypesRepository {
  importing?: boolean
  importProgress?: string
}

enum ImportStatus {
  FAILED = 'failed'
}

interface progessState {
  state: string
}

export default function RepositoriesListing() {
  const { getString } = useStrings()
  const history = useHistory()
  const rowContainerRef = useRef<HTMLDivElement>(null)
  const [nameTextWidth, setNameTextWidth] = useState(600)
  const space = useGetSpaceParam()
  const [searchTerm, setSearchTerm] = useState<string | undefined>()
  const { routes, standalone, hooks, routingId } = useAppContext()
  const { updateQueryParams } = useUpdateQueryParams()
  const pageBrowser = useQueryParams<PageBrowserProps>()
  const pageInit = pageBrowser.page ? parseInt(pageBrowser.page) : 1
  const [page, setPage] = usePageIndex(pageInit)
  const [updatedRepositories, setUpdatedRepositories] = useState<TypesRepository[]>()

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

  const bearerToken = hooks?.useGetToken?.() || ''

  const addImportProgressToData = async (repos: TypesRepository[]) => {
    const updatedData = await Promise.all(
      repos.map(async repo => {
        if (repo.importing) {
          const importProgress = await getUsingFetch(
            getConfig('code/api/v1'),
            `/repos/${repo.path}/+/import-progress`,
            bearerToken,
            {
              queryParams: {
                accountIdentifier: routingId
              }
            }
          )
          return { ...repo, importProgress: (importProgress as progessState).state }
        }
        return repo
      })
    )

    return updatedData
  }

  useEffect(() => {
    const fetchRepo = async () => {
      if (repositories) {
        const updatedRepos = await addImportProgressToData(repositories)
        setUpdatedRepositories(updatedRepos)
      }
    }

    fetchRepo()
  }, [repositories]) // eslint-disable-line react-hooks/exhaustive-deps

  const columns: Column<TypesRepoExtended>[] = useMemo(
    () => [
      {
        Header: getString('repos.name'),
        width: 'calc(100% - 210px)',

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

                  {record?.importProgress === ImportStatus.FAILED ? (
                    <Text className={css.desc} width={nameTextWidth} lineClamp={1}>
                      {getString('importFailed')}
                    </Text>
                  ) : record.importing ? (
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
          return row?.original?.importProgress === ImportStatus.FAILED ? null : row.original.importing ? (
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
      },
      {
        id: 'action',
        width: '30px',
        Cell: ({ row }: CellProps<TypesRepoExtended>) => {
          const { showSuccess, showError } = useToaster()
          const { mutate: deleteRepo } = useDeleteRepository({})
          const confirmCancelImport = useConfirmAct()
          return (
            <Container onClick={Utils.stopEvent}>
              {row?.original?.importProgress === ImportStatus.FAILED ? (
                <OptionsMenuButton
                  isDark
                  width="100px"
                  items={[
                    {
                      text: getString('deleteImport'),
                      onClick: () =>
                        confirmCancelImport({
                          title: getString('deleteImport'),
                          confirmText: getString('delete'),
                          intent: Intent.DANGER,
                          message: (
                            <Text
                              style={{ wordBreak: 'break-word' }}
                              font={{ variation: FontVariation.BODY2_SEMI, size: 'small' }}>
                              <String
                                useRichText
                                stringID="deleteFailedImport"
                                vars={{ name: row.original?.uid }}
                                tagName="div"
                              />
                            </Text>
                          ),
                          action: async () => {
                            deleteRepo(`${row.original?.path as string}/+/`)
                              .then(() => {
                                showSuccess(getString('deletedImport'), 2000)
                                refetch()
                              })
                              .catch(err => {
                                showError(getErrorMessage(err), 0, getString('failedToDeleteImport'))
                              })
                          }
                        })
                    }
                  ]}
                />
              ) : (
                row.original.importing && (
                  <OptionsMenuButton
                    isDark
                    width="100px"
                    items={[
                      {
                        text: getString('cancelImport'),
                        onClick: () =>
                          confirmCancelImport({
                            title: getString('cancelImport'),
                            confirmText: getString('cancelImport'),
                            intent: Intent.DANGER,
                            message: (
                              <Text
                                style={{ wordBreak: 'break-word' }}
                                font={{ variation: FontVariation.BODY2_SEMI, size: 'small' }}>
                                <String
                                  useRichText
                                  stringID="cancelImportConfirm"
                                  vars={{ name: row.original?.uid }}
                                  tagName="div"
                                />
                              </Text>
                            ),
                            action: async () => {
                              deleteRepo(`${row.original?.path as string}/+/`)
                                .then(() => {
                                  showSuccess(getString('cancelledImport'), 2000)
                                  refetch()
                                })
                                .catch(err => {
                                  showError(getErrorMessage(err), 0, getString('failedToCancelImport'))
                                })
                            }
                          })
                      }
                    ]}
                  />
                )
              )}
            </Container>
          )
        }
      }
    ],
    [nameTextWidth, getString, searchTerm] // eslint-disable-line react-hooks/exhaustive-deps
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
                  data={updatedRepositories || []}
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
        </Layout.Horizontal>
      </PageBody>
    </Container>
  )
}
