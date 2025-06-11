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
  Checkbox,
  CheckboxVariant,
  Container,
  FlexExpander,
  Layout,
  PageBody,
  PageHeader,
  TableV2 as Table,
  Text,
  useToaster,
  Utils
} from '@harnessio/uicore'
import { ProgressBar, Intent } from '@blueprintjs/core'
import { Color, FontVariation } from '@harnessio/design-system'
import type { CellProps, Column } from 'react-table'
import { debounce, isEmpty } from 'lodash-es'
import Keywords from 'react-keywords'
import cx from 'classnames'
import { useGet } from 'restful-react'
import { useHistory } from 'react-router-dom'
import { Icon } from '@harnessio/icons'
import { useStrings, String } from 'framework/strings'
import { useAppContext } from 'AppContext'
import { voidFn, formatDate, getErrorMessage, LIST_FETCHING_LIMIT, PageBrowserProps } from 'utils/Utils'
import { NewRepoModalButton } from 'components/NewRepoModalButton/NewRepoModalButton'
import type { RepoRepositoryOutput } from 'services/code'
import { useDeleteRepository } from 'services/code'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { usePageIndex } from 'hooks/usePageIndex'
import { useQueryParams } from 'hooks/useQueryParams'
import { useUpdateQueryParams } from 'hooks/useUpdateQueryParams'
import useSpaceSSE from 'hooks/useSpaceSSE'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { SearchInputWithSpinner } from 'components/SearchInputWithSpinner/SearchInputWithSpinner'
import FavoriteStar from 'components/FavoriteStar/FavoriteStar'
import { LoadingSpinner } from 'components/LoadingSpinner/LoadingSpinner'
import { NoResultCard } from 'components/NoResultCard/NoResultCard'
import { ResourceListingPagination } from 'components/ResourceListingPagination/ResourceListingPagination'
import { RepoTypeLabel } from 'components/RepoTypeLabel/RepoTypeLabel'
import KeywordSearch from 'components/CodeSearch/KeywordSearch'
import { OptionsMenuButton } from 'components/OptionsMenuButton/OptionsMenuButton'
import { useConfirmAct } from 'hooks/useConfirmAction'
import { getUsingFetch, getConfig } from 'services/config'
import noRepoImage from './no-repo.svg?url'
import css from './RepositoriesListing.module.scss'

interface TypesRepoExtended extends RepoRepositoryOutput {
  importing?: boolean
  importProgress?: string
  importProgressErrorMessage?: string
}

enum ImportStatus {
  FAILED = 'failed',
  FETCH_FAILED = 'fetch failed'
}

interface ProgressState {
  state: string
}

export default function RepositoriesListing() {
  const { getString } = useStrings()
  const history = useHistory()
  const rowContainerRef = useRef<HTMLDivElement>(null)
  const [nameTextWidth, setNameTextWidth] = useState(600)
  const space = useGetSpaceParam()
  const { routes, standalone, hooks, routingId } = useAppContext()
  const { updateQueryParams, replaceQueryParams } = useUpdateQueryParams()
  const { updateRepoMetadata } = useGetRepositoryMetadata()
  const pageBrowser = useQueryParams<PageBrowserProps & { only_favorites?: string }>()
  const pageInit = pageBrowser.page ? parseInt(pageBrowser.page) : 1
  const [page, setPage] = usePageIndex(pageInit)
  const [searchTerm, setSearchTerm] = useState<string | undefined>()
  const [debouncedSearchTerm, setDebouncedSearchTerm] = useState(searchTerm)
  const { showError, showSuccess } = useToaster()
  const [updatedRepositories, setUpdatedRepositories] = useState<RepoRepositoryOutput[]>()

  const {
    data: repositories,
    error,
    loading,
    refetch,
    response
  } = useGet<RepoRepositoryOutput[]>({
    path: `/api/v1/spaces/${space}/+/repos`,
    queryParams: {
      page: pageBrowser.page,
      limit: LIST_FETCHING_LIMIT,
      query: debouncedSearchTerm,
      only_favorites: pageBrowser.only_favorites
    }
  })

  const debouncedRefetch = useCallback(
    debounce((value: string) => {
      setDebouncedSearchTerm(value)
      setPage(1)
    }, 500),
    []
  )

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
    if (page > 1) {
      updateQueryParams({ page: page.toString() })
    } else {
      const params = { ...pageBrowser }
      delete params.page
      replaceQueryParams(params, undefined, true)
    }
  }, [space, page]) // eslint-disable-line react-hooks/exhaustive-deps

  const bearerToken = hooks?.useGetToken?.() || ''

  const addImportProgressToData = async (repos: RepoRepositoryOutput[]) => {
    return await Promise.all(
      repos.map(async repo => {
        if (repo.importing) {
          try {
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
            return { ...repo, importProgress: (importProgress as ProgressState).state }
          } catch (err) {
            return {
              ...repo,
              importProgress: ImportStatus.FETCH_FAILED,
              importProgressErrorMessage: getErrorMessage(err) as string
            }
          }
        }
        return repo
      })
    )
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
          const renderImportProgressText = () => {
            switch (record?.importProgress) {
              case ImportStatus.FAILED:
                return getString('importFailed')
              case ImportStatus.FETCH_FAILED:
                return record?.importProgressErrorMessage
              default:
                if (record?.importing) {
                  return getString('importProgress')
                }
                return record?.description ?? null
            }
          }
          return (
            <Container className={css.nameContainer}>
              <Layout.Horizontal spacing="small" style={{ flexGrow: 1 }}>
                <Layout.Vertical flex className={css.name} ref={rowContainerRef}>
                  <Text className={css.repoName} width={nameTextWidth} lineClamp={2}>
                    <Keywords value={searchTerm}>{record.identifier}</Keywords>
                    <RepoTypeLabel
                      isPublic={row.original.is_public}
                      isArchived={row.original.archived}
                      margin={{ left: 'small' }}
                    />
                  </Text>

                  <Text className={css.desc} width={nameTextWidth} lineClamp={1}>
                    {renderImportProgressText()}
                  </Text>
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
          if (
            [ImportStatus.FAILED, ImportStatus.FETCH_FAILED].includes(row?.original?.importProgress as ImportStatus)
          ) {
            return null
          }

          return row.original.importing ? (
            <Layout.Horizontal style={{ alignItems: 'center' }} padding={{ right: 'large' }}>
              <ProgressBar intent={Intent.PRIMARY} className={css.progressBar} />
            </Layout.Horizontal>
          ) : (
            <Layout.Horizontal style={{ alignItems: 'center', justifyContent: 'space-between' }}>
              <Text color={Color.BLACK} lineClamp={1} rightIconProps={{ size: 10 }} width={120}>
                {formatDate(row.original.updated as number)}
              </Text>
              <FavoriteStar
                isFavorite={row.original.is_favorite}
                resourceId={row.original.id || 0}
                resourceType={'REPOSITORY'}
                className={css.favorite}
                activeClassName={css.favoriteActive}
                key={row.original.id}
                onChange={favorite => {
                  row.original.is_favorite = favorite
                  updateRepoMetadata(row.original.path || '', 'is_favorite', favorite)
                }}
              />
            </Layout.Horizontal>
          )
        },
        disableSortBy: true
      },
      {
        id: 'action',
        width: '30px',
        Cell: ({ row }: CellProps<TypesRepoExtended>) => {
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
                                vars={{ name: row.original?.identifier }}
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
                                  vars={{ name: row.original?.identifier }}
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
          history.push(routes.toCODERepository({ repoPath: (repoInfo as RepoRepositoryOutput).path as string }))
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
          when: () =>
            !loading &&
            isEmpty(repositories) &&
            debouncedSearchTerm === undefined &&
            !JSON.parse(pageBrowser.only_favorites || 'false'),
          image: noRepoImage,
          message: getString('repos.noDataMessage'),
          button: NewRepoButton
        }}>
        <LoadingSpinner visible={loading} className={css.spinner} />
        <Layout.Horizontal>
          <Container className={css.repoListingContainer} margin={{ top: 'medium' }}>
            <Container padding="xlarge">
              <Layout.Horizontal spacing="large" className={css.layout}>
                {NewRepoButton}
                <Checkbox
                  variant={CheckboxVariant.BOXED}
                  checked={JSON.parse(pageBrowser.only_favorites || 'false')}
                  labelElement={<Icon name="star" color={Color.YELLOW_900} size={14} />}
                  onChange={e => {
                    updateQueryParams({ only_favorites: e.currentTarget.checked.toString() })
                    setPage(1)
                  }}
                />
                <FlexExpander />
                <SearchInputWithSpinner
                  loading={loading && debouncedSearchTerm !== undefined}
                  query={searchTerm}
                  setQuery={value => {
                    setSearchTerm(value)
                    debouncedRefetch(value)
                  }}
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
                showWhen={() =>
                  isEmpty(repositories) && (JSON.parse(pageBrowser.only_favorites || 'false') || !!searchTerm?.length)
                }
                forSearch={!isEmpty(debouncedSearchTerm)}
                forFilter={JSON.parse(pageBrowser.only_favorites || 'false')}
              />
            </Container>
            <ResourceListingPagination response={response} page={page} setPage={setPage} />
          </Container>
        </Layout.Horizontal>
      </PageBody>
    </Container>
  )
}
