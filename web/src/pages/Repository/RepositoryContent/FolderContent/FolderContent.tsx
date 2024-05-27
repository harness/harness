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
  Container,
  TableV2 as Table,
  Text,
  Utils,
  StringSubstitute,
  Layout,
  TextProps,
  useIsMounted
} from '@harnessio/uicore'
import { Color } from '@harnessio/design-system'
import cx from 'classnames'
import type { CellProps, Column } from 'react-table'
import { Page } from 'iconoir-react'

import { Render } from 'react-jsx-match'
import { chunk, sortBy, throttle } from 'lodash-es'
import { useGet, useMutate } from 'restful-react'
import { Link, useHistory } from 'react-router-dom'
import { useAppContext } from 'AppContext'
import type { OpenapiContentInfo, OpenapiDirContent, TypesCommit } from 'services/code'
import { formatDate, isInViewport, LIST_FETCHING_LIMIT, PAGE_CONTAINER_WIDTH } from 'utils/Utils'
import {
  findReadmeInfo,
  GitInfoProps,
  isFile,
  isSymlink,
  isSubmodule,
  normalizeGitRef,
  RepositorySummaryData
} from 'utils/GitUtils'
import { LatestCommitForFolder } from 'components/LatestCommit/LatestCommit'
import { CommitActions } from 'components/CommitActions/CommitActions'
import { useEventListener } from 'hooks/useEventListener'
import { useShowRequestError } from 'hooks/useShowRequestError'
import RepositorySummary from './RepositorySummary/RepositorySummary'
import { Readme } from './Readme'
import CodeFolder from '../../../../icons/CodeFileFill.svg?url'
import Submodule from '../../../../icons/Submodules.svg?url'
import Symlink from '../../../../icons/Symlink.svg?url'
import repositoryCSS from '../../Repository.module.scss'
import css from './FolderContent.module.scss'

type FolderContentProps = Pick<GitInfoProps, 'repoMetadata' | 'resourceContent' | 'resourcePath' | 'gitRef'>

const checkIcon = (row: OpenapiContentInfo): React.ReactElement => {
  if (isFile(row)) {
    return <Page className={css.fileIcon} width={14} height={14} />
  } else if (isSymlink(row)) {
    return <img width={14} height={14} src={Symlink} />
  } else if (isSubmodule(row)) {
    return <img width={14} height={14} src={Submodule} />
  } else {
    return <img width={14} height={14} src={CodeFolder} />
  }
}
export function FolderContent({ repoMetadata, resourceContent, gitRef, resourcePath }: FolderContentProps) {
  const history = useHistory()
  const { routes, standalone } = useAppContext()
  const columns: Column<OpenapiContentInfo>[] = useMemo(
    () => [
      {
        Header: 'Files',
        id: 'name',
        width: '30%',
        Cell: ({ row }: CellProps<OpenapiContentInfo>) => {
          return (
            <Container>
              <Layout.Horizontal spacing="small">
                {checkIcon(row.original)}
                <ListingItemLink
                  url={routes.toCODERepository({
                    repoPath: repoMetadata.path as string,
                    gitRef,
                    resourcePath: row.original.path
                  })}
                  text={row.original.name as string}
                  data-resource-path={row.original.path}
                  lineClamp={1}
                />
              </Layout.Horizontal>
            </Container>
          )
        }
      },
      {
        Header: 'Date',
        id: 'when',
        width: '150px',
        Cell: ({ row }: CellProps<OpenapiContentInfo>) => {
          return (
            <Text lineClamp={1} color={Color.GREY_500} className={css.rowText}>
              {formatDate(row.original.latest_commit?.author?.when as string)}
            </Text>
          )
        }
      },
      {
        Header: 'Commits',
        id: 'message',
        width: 'calc(70% - 150px)',
        Cell: ({ row }: CellProps<OpenapiContentInfo>) => (
          <Container className={css.commitContainer}>
            <CommitMessageLinks repoMetadata={repoMetadata} rowData={row.original} />
          </Container>
        )
      }
    ],
    [gitRef] // eslint-disable-line react-hooks/exhaustive-deps
  )
  const readmeInfo = useMemo(() => findReadmeInfo(resourceContent), [resourceContent])
  const scrollDOMElement = useMemo(
    () => (standalone ? document.querySelector(`.${repositoryCSS.main}`)?.parentElement : window) as HTMLElement,
    [standalone]
  )
  const resourceEntries = useMemo(
    () => sortBy((resourceContent.content as OpenapiDirContent)?.entries || [], ['type', 'name']),
    [resourceContent.content]
  )
  const [pathsChunks, setPathsChunks] = useState<PathsChunks>([])
  const { mutate: fetchLastCommitsForPaths } = useMutate<PathDetails>({
    verb: 'POST',
    path: `/api/v1/repos/${repoMetadata.path as string}/+/path-details`,
    queryParams: {
      git_ref: normalizeGitRef(gitRef)
    }
  })
  const lastCommitMapping = useRef<Record<string, TypesCommit>>({})
  const mergedContentEntries = useMemo(
    () =>
      resourceEntries.map(entry => ({
        ...entry,
        latest_commit: lastCommitMapping.current[entry.path as string] || entry.latest_commit
      })),
    [resourceEntries, pathsChunks] // eslint-disable-line react-hooks/exhaustive-deps
  )
  const isMounted = useIsMounted()
  const {
    data: repoSummaryData,
    error: repoSummaryError,
    loading: loadingSummaryData
  } = useGet<RepositorySummaryData>({
    path: `/api/v1/repos/${repoMetadata?.path}/+/summary`
  })

  useShowRequestError(repoSummaryError)

  // The idea is to fetch last commit details for chunks that has atleast one path which is
  // rendered in the viewport
  // eslint-disable-next-line react-hooks/exhaustive-deps
  const scrollCallback = useCallback(
    throttle(() => {
      if (isMounted.current) {
        for (const pathsChunk of pathsChunks) {
          const { paths, loaded, loading, failed } = pathsChunk

          if (!loaded && !loading && !failed) {
            for (let i = 0; i < paths.length; i++) {
              try {
                const element = document.querySelector(`[data-resource-path="${paths[i]}"]`)

                if (element && isInViewport(element, IN_VIEW_DETECTION_MARGIN)) {
                  pathsChunk.loading = true

                  if (isMounted.current) {
                    setPathsChunks(pathsChunks.map(_chunk => (pathsChunk === _chunk ? pathsChunk : _chunk)))

                    fetchLastCommitsForPaths({ paths })
                      .then(response => {
                        pathsChunk.loaded = true

                        if (isMounted.current) {
                          response?.details?.forEach(({ path, last_commit }) => {
                            lastCommitMapping.current[path] = last_commit
                          })

                          setPathsChunks(pathsChunks.map(_chunk => (pathsChunk === _chunk ? pathsChunk : _chunk)))
                        }
                      })
                      .catch(error => {
                        pathsChunk.loaded = false
                        pathsChunk.loading = false
                        pathsChunk.failed = true

                        if (isMounted.current) {
                          setPathsChunks(pathsChunks.map(_chunk => (pathsChunk === _chunk ? pathsChunk : _chunk)))
                        }

                        console.error('Failed to fetch path commit details', error) // eslint-disable-line no-console
                      })
                  }
                  break
                }
              } catch (exception) {
                // eslint-disable-next-line no-console
                console.error('Failed to handle scroll', exception)
              }
            }
          }
        }
      }
    }, 50),
    [pathsChunks, setPathsChunks]
  )

  // Group all resourceEntries paths into chunks, each has LIST_FETCHING_LIMIT paths
  useEffect(() => {
    setPathsChunks(
      chunk(resourceEntries.map(entry => entry.path as string) || [], LIST_FETCHING_LIMIT).map(paths => ({
        paths,
        loaded: false,
        loading: false,
        failed: false
      }))
    )
    lastCommitMapping.current = {}
  }, [resourceEntries])

  useEventListener('scroll', scrollCallback, scrollDOMElement)

  // Trigger scroll event callback on mount and cancel it on unmount
  useEffect(() => {
    scrollCallback()
    return () => scrollCallback.cancel()
  }, [scrollCallback])

  return (
    <Layout.Horizontal>
      <Container className={css.folderContent}>
        <LatestCommitForFolder
          repoMetadata={repoMetadata}
          latestCommit={resourceContent?.latest_commit}
          gitRef={gitRef}
          repoSummaryData={repoSummaryData}
          loadingSummaryData={loadingSummaryData}
        />
        <Table<OpenapiContentInfo>
          className={css.table}
          columns={columns}
          data={mergedContentEntries}
          onRowClick={entry => {
            history.push(
              routes.toCODERepository({
                repoPath: repoMetadata.path as string,
                gitRef,
                resourcePath: entry.path
              })
            )
          }}
          getRowClassName={() => css.row}
        />

        <Render when={readmeInfo}>
          <Readme
            metadata={repoMetadata}
            readmeInfo={readmeInfo as OpenapiContentInfo}
            gitRef={gitRef}
            maxWidth={`calc(var(${PAGE_CONTAINER_WIDTH}) - 48px)`}
          />
        </Render>
      </Container>
      <Render when={!resourcePath}>
        <RepositorySummary
          metadata={repoMetadata}
          gitRef={gitRef}
          repoSummaryData={repoSummaryData}
          loadingSummaryData={loadingSummaryData}
        />
      </Render>
    </Layout.Horizontal>
  )
}

type PathDetails = {
  details: Array<{
    path: string
    last_commit: TypesCommit
  }>
}

type PathsChunks = Array<{
  paths: string[]
  loaded: boolean
  loading: boolean
  failed: boolean
}>

interface CommitMessageLinksProps extends Pick<GitInfoProps, 'repoMetadata'> {
  rowData: OpenapiContentInfo
}

const CommitMessageLinks: React.FC<CommitMessageLinksProps> = ({ repoMetadata, rowData }) => {
  const { routes } = useAppContext()
  let title: string | JSX.Element = (rowData.latest_commit?.title || '') as string
  const match = title.match(/\(#\d+\)$/)

  if (match?.length) {
    const titleWithoutPullRequestId = title.replace(match[0], '')
    const pullRequestId = match[0].replace('(#', '').replace(')', '')

    title = (
      <StringSubstitute
        str="{COMMIT_URL}&nbsp;({PR_URL})"
        vars={{
          COMMIT_URL: (
            <ListingItemLink
              url={routes.toCODECommit({
                repoPath: repoMetadata.path as string,
                commitRef: rowData.latest_commit?.sha as string
              })}
              text={titleWithoutPullRequestId}
            />
          ),
          PR_URL: (
            <ListingItemLink
              url={routes.toCODEPullRequest({
                repoPath: repoMetadata.path as string,
                pullRequestId
              })}
              text={`#${pullRequestId}`}
              className={css.hightlight}
              wrapperClassName={css.noShrink}
            />
          )
        }}
      />
    )
  } else {
    title = (
      <ListingItemLink
        url={routes.toCODECommit({
          repoPath: repoMetadata.path as string,
          commitRef: rowData.latest_commit?.sha as string
        })}
        text={title}
      />
    )
  }

  return (
    <Container>
      <Layout.Horizontal>
        {rowData.latest_commit?.sha && (
          <Container onClick={Utils.stopEvent}>
            <CommitActions
              href={routes.toCODECommit({
                repoPath: repoMetadata.path as string,
                commitRef: rowData.latest_commit?.sha as string
              })}
              sha={rowData.latest_commit?.sha as string}
              enableCopy
            />
          </Container>
        )}
        <Layout.Horizontal padding={{ left: 'large' }} className={css.commitMsgLayout}>
          {title}
        </Layout.Horizontal>
      </Layout.Horizontal>
    </Container>
  )
}

interface ListingItemLinkProps extends TextProps {
  url: string
  text: string
  wrapperClassName?: string
}

const ListingItemLink: React.FC<ListingItemLinkProps> = ({ url, text, className, wrapperClassName, ...props }) => (
  <Container onClick={Utils.stopEvent} className={cx(css.linkContainer, wrapperClassName)}>
    <Link className={css.link} to={url}>
      <Text tag="span" color={Color.BLACK} lineClamp={1} className={cx(css.text, className)} {...props}>
        {text.trim()}
      </Text>
    </Link>
  </Container>
)

const IN_VIEW_DETECTION_MARGIN = 500
