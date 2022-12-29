import React, { useMemo } from 'react'
import { Container, Color, TableV2 as Table, Text } from '@harness/uicore'
import type { CellProps, Column } from 'react-table'
import { sortBy } from 'lodash-es'
import { useHistory } from 'react-router-dom'
import { useAppContext } from 'AppContext'
import type { OpenapiContentInfo, OpenapiDirContent } from 'services/code'
import { formatDate } from 'utils/Utils'
import { findReadmeInfo, CodeIcon, GitInfoProps, isFile } from 'utils/GitUtils'
import { LatestCommitForFolder } from 'components/LatestCommit/LatestCommit'
import { Readme } from './Readme'
import css from './FolderContent.module.scss'

export function FolderContent({
  repoMetadata,
  resourceContent,
  gitRef
}: Pick<GitInfoProps, 'repoMetadata' | 'resourceContent' | 'gitRef'>) {
  const history = useHistory()
  const { routes } = useAppContext()
  const columns: Column<OpenapiContentInfo>[] = useMemo(
    () => [
      {
        id: 'name',
        width: '40%',
        Cell: ({ row }: CellProps<OpenapiContentInfo>) => {
          return (
            <Text
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
            <Text color={Color.BLACK} lineClamp={1} className={css.rowText}>
              {row.original.latest_commit?.title}
            </Text>
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
    []
  )
  const readmeInfo = useMemo(() => findReadmeInfo(resourceContent), [resourceContent])

  return (
    <Container className={css.folderContent}>
      <LatestCommitForFolder repoMetadata={repoMetadata} latestCommit={resourceContent?.latest_commit} />

      <Table<OpenapiContentInfo>
        className={css.table}
        hideHeaders
        columns={columns}
        data={sortBy((resourceContent.content as OpenapiDirContent)?.entries || [], ['type', 'name'])}
        onRowClick={data => {
          history.push(
            routes.toCODERepository({
              repoPath: repoMetadata.path as string,
              gitRef,
              resourcePath: data.path
            })
          )
        }}
        getRowClassName={() => css.row}
      />

      {!!readmeInfo && <Readme metadata={repoMetadata} readmeInfo={readmeInfo} />}
    </Container>
  )
}
