import React, { useCallback, useEffect, useMemo, useState } from 'react'
import { Container, Color, TableV2 as Table, Text, Utils } from '@harness/uicore'
import type { CellProps, Column } from 'react-table'
import { Render } from 'react-jsx-match'
import { chunk, clone, sortBy, throttle } from 'lodash-es'
import { useMutate } from 'restful-react'
import { useHistory } from 'react-router-dom'
import { useAppContext } from 'AppContext'
import type { OpenapiContentInfo, OpenapiDirContent, TypesCommit } from 'services/code'
import { formatDate, LIST_FETCHING_LIMIT } from 'utils/Utils'
import { findReadmeInfo, CodeIcon, GitInfoProps, isFile } from 'utils/GitUtils'
import { LatestCommitForFolder } from 'components/LatestCommit/LatestCommit'
import { useEventListener } from 'hooks/useEventListener'
import { Readme } from './Readme'
import repositoryCSS from '../../Repository.module.scss'
import css from './FolderContent.module.scss'

export function FolderContent({
  repoMetadata,
  resourceContent,
  gitRef
}: Pick<GitInfoProps, 'repoMetadata' | 'resourceContent' | 'gitRef'>) {
  const history = useHistory()
  const { routes, standalone } = useAppContext()
  const columns: Column<OpenapiContentInfo>[] = useMemo(
    () => [
      {
        id: 'name',
        width: '40%',
        Cell: ({ row }: CellProps<OpenapiContentInfo>) => {
          return (
            <Text
              data-resource-path={row.original.path}
              lineClamp={1}
              className={css.rowText}
              color={Color.BLACK}
              icon={isFile(row.original) ? CodeIcon.File : CodeIcon.Folder}
              iconProps={{ margin: { right: 'xsmall' } }}>
              {row.original.name}
            </Text>
          )
        }
      },
      {
        id: 'message',
        width: 'calc(60% - 100px)',
        Cell: ({ row }: CellProps<OpenapiContentInfo>) => {
          return (
            <Container onClick={Utils.stopEvent}>
              <Text
                tag="a"
                role="button"
                color={Color.BLACK}
                lineClamp={1}
                padding={{ right: 'small' }}
                className={css.rowText}
                onClick={() => {
                  history.push(
                    routes.toCODECommit({
                      repoPath: repoMetadata.path as string,
                      commitRef: row.original.latest_commit?.sha as string
                    })
                  )
                }}>
                {row.original.latest_commit?.title}
              </Text>
            </Container>
          )
        }
      },
      {
        id: 'when',
        width: '100px',
        Cell: ({ row }: CellProps<OpenapiContentInfo>) => {
          return (
            <Text lineClamp={1} color={Color.GREY_500} className={css.rowText}>
              {formatDate(row.original.latest_commit?.author?.when as string)}
            </Text>
          )
        }
      }
    ],
    [] // eslint-disable-line react-hooks/exhaustive-deps
  )
  const readmeInfo = useMemo(() => findReadmeInfo(resourceContent), [resourceContent])
  const scrollElement = useMemo(
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
    path: `/api/v1/repos/${encodeURIComponent(repoMetadata.path as string)}/path-details`
  })
  const [lastCommitMapping, setLastCommitMapping] = useState<Record<string, TypesCommit>>({})
  const mergedContentEntries = useMemo(
    () => resourceEntries.map(entry => ({ ...entry, latest_commit: lastCommitMapping[entry.path as string] })),
    [resourceEntries, lastCommitMapping]
  )

  // The idea is to fetch last commit details for chunks that has atleast one path which is
  // rendered in the viewport
  // eslint-disable-next-line react-hooks/exhaustive-deps
  const scrollCallback = useCallback(
    throttle(() => {
      pathsChunks.forEach(pathsChunk => {
        const { paths, loaded, loading } = pathsChunk

        if (!loaded && !loading) {
          for (let i = 0; i < paths.length; i++) {
            const element = document.querySelector(`[data-resource-path="${paths[i]}"]`)

            if (element && isInViewport(element)) {
              pathsChunk.loading = true

              setPathsChunks(pathsChunks.map(_chunk => (pathsChunk === _chunk ? pathsChunk : _chunk)))

              fetchLastCommitsForPaths({ paths })
                .then(response => {
                  const pathMapping: Record<string, TypesCommit> = clone(lastCommitMapping)

                  pathsChunk.loaded = true
                  setPathsChunks(pathsChunks.map(_chunk => (pathsChunk === _chunk ? pathsChunk : _chunk)))

                  response?.details?.forEach(({ path, last_commit }) => {
                    pathMapping[path] = last_commit
                  })
                  setLastCommitMapping(pathMapping)
                })
                .catch(error => {
                  pathsChunk.loaded = false
                  pathsChunk.loading = false
                  setPathsChunks(pathsChunks.map(_chunk => (pathsChunk === _chunk ? pathsChunk : _chunk)))
                  console.log('Failed to fetch path commit details', error) // eslint-disable-line no-console
                })

              break
            }
          }
        }
      })
    }, 100),
    [pathsChunks, lastCommitMapping]
  )

  // Group all resourceEntries paths into chunks, each has LIST_FETCHING_LIMIT paths
  useEffect(() => {
    setPathsChunks(
      chunk(resourceEntries.map(entry => entry.path as string) || [], LIST_FETCHING_LIMIT).map(paths => ({
        paths,
        loaded: false,
        loading: false
      }))
    )
  }, [resourceEntries])

  useEventListener('scroll', scrollCallback, scrollElement)

  // Trigger scroll event callback on mount and cancel it on unmount
  useEffect(() => {
    scrollCallback()

    return () => {
      scrollCallback.cancel()
    }
  }, [scrollCallback])

  return (
    <Container className={css.folderContent}>
      <LatestCommitForFolder repoMetadata={repoMetadata} latestCommit={resourceContent?.latest_commit} />

      <Table<OpenapiContentInfo>
        className={css.table}
        hideHeaders
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
        <Readme metadata={repoMetadata} readmeInfo={readmeInfo as OpenapiContentInfo} gitRef={gitRef} />
      </Render>
    </Container>
  )
}

function isInViewport(element: Element) {
  const rect = element.getBoundingClientRect()
  return (
    rect.top >= 0 &&
    rect.left >= 0 &&
    rect.bottom <= (window.innerHeight || document.documentElement.clientHeight) &&
    rect.right <= (window.innerWidth || document.documentElement.clientWidth)
  )
}

interface PathDetails {
  details: Array<{ path: string; last_commit: TypesCommit }>
}

type PathsChunks = Array<{ paths: string[]; loaded: boolean; loading: boolean }>
