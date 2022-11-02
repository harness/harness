import React, { useMemo } from 'react'
import { Container, Color, TableV2 as Table, Text } from '@harness/uicore'
import type { CellProps, Column } from 'react-table'
import { sortBy } from 'lodash-es'
import { useHistory } from 'react-router-dom'
import { useStrings } from 'framework/strings'
import { useAppContext } from 'AppContext'
import type { OpenapiContentInfo, OpenapiDirContent, OpenapiGetContentOutput, TypesRepository } from 'services/scm'
import { formatDate } from 'utils/Utils'
import { findReadmeInfo, GitIcon, isFile } from 'utils/GitUtils'
import { LatestCommit } from 'components/LatestCommit/LatestCommit'
import { Readme } from './Readme'
import css from './FolderContent.module.scss'

interface FolderContentProps {
  repoMetadata: TypesRepository
  gitRef?: string
  contentInfo: OpenapiGetContentOutput
}

export function FolderContent({ repoMetadata, contentInfo, gitRef }: FolderContentProps): JSX.Element {
  const { getString } = useStrings()
  const history = useHistory()
  const { routes } = useAppContext()
  const columns: Column<OpenapiContentInfo>[] = useMemo(
    () => [
      {
        Header: getString('name'),
        width: '40%',
        Cell: ({ row }: CellProps<OpenapiContentInfo>) => {
          return (
            <Text
              className={css.rowText}
              color={Color.BLACK}
              icon={isFile(row.original) ? GitIcon.FILE : GitIcon.FOLDER}
              iconProps={{ margin: { right: 'xsmall' } }}>
              {row.original.name}
            </Text>
          )
        }
      },
      {
        Header: getString('commits'),
        width: 'calc(60% - 100px)',
        Cell: ({ row }: CellProps<OpenapiContentInfo>) => {
          return (
            <Text color={Color.BLACK} lineClamp={1} className={css.rowText}>
              {row.original.latestCommit?.title}
            </Text>
          )
        }
      },
      {
        Header: getString('repos.lastChange'),
        width: '100px',
        Cell: ({ row }: CellProps<OpenapiContentInfo>) => {
          return (
            <Text lineClamp={1} color={Color.GREY_500} className={css.rowText}>
              {formatDate(row.original.latestCommit?.author?.when as string)}
            </Text>
          )
        }
      }
    ],
    [getString]
  )
  const readmeInfo = useMemo(() => findReadmeInfo(contentInfo), [contentInfo])

  return (
    <Container className={css.folderContent}>
      <LatestCommit repoMetadata={repoMetadata} latestCommit={contentInfo?.latestCommit} />

      <Table<OpenapiContentInfo>
        className={css.table}
        hideHeaders
        columns={columns}
        data={sortBy((contentInfo.content as OpenapiDirContent)?.entries || [], ['type', 'name'])}
        onRowClick={data => {
          history.push(
            routes.toSCMRepository({
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
