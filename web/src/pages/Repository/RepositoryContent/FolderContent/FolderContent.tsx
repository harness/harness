import React, { useMemo } from 'react'
import {
  Container,
  Color,
  Layout,
  Button,
  ButtonSize,
  FlexExpander,
  ButtonVariation,
  TableV2 as Table,
  Text,
  FontVariation
} from '@harness/uicore'
import type { CellProps, Column } from 'react-table'
import { sortBy } from 'lodash-es'
import ReactTimeago from 'react-timeago'
import { Link, useHistory } from 'react-router-dom'
import { useStrings } from 'framework/strings'
import { useAppContext } from 'AppContext'
import type { OpenapiContentInfo, OpenapiDirContent, OpenapiGetContentOutput, TypesRepository } from 'services/scm'
import { formatDate } from 'utils/Utils'
import { findReadmeInfo, GitIcon, isFile } from 'utils/GitUtils'
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
        accessor: (row: OpenapiContentInfo) => row.name,
        width: '30%',
        Cell: ({ row }: CellProps<OpenapiContentInfo>) => {
          return (
            <Text
              color={Color.BLACK}
              icon={isFile(row.original) ? GitIcon.FILE : GitIcon.FOLDER}
              lineClamp={1}
              iconProps={{ margin: { right: 'xsmall' } }}>
              {row.original.name}
            </Text>
          )
        }
      },
      {
        Header: getString('commits'),
        accessor: row => row.latestCommit,
        width: '55%',
        Cell: ({ row }: CellProps<OpenapiContentInfo>) => {
          return (
            <Text color={Color.BLACK} lineClamp={1}>
              {row.original.latestCommit?.title}
            </Text>
          )
        }
      },
      {
        Header: getString('repos.lastChange'),
        accessor: row => row.latestCommit?.author?.when,
        width: '15%',
        Cell: ({ row }: CellProps<OpenapiContentInfo>) => {
          return <Text lineClamp={1}>{formatDate(row.original.latestCommit?.author?.when as string)}</Text>
        },
        disableSortBy: true
      }
    ],
    [getString]
  )
  const readmeInfo = useMemo(() => findReadmeInfo(contentInfo), [contentInfo])

  return (
    <Container className={css.folderContent}>
      <Container>
        <Layout.Horizontal spacing="medium" padding={{ bottom: 'medium' }} className={css.lastCommit}>
          <Text font={{ variation: FontVariation.SMALL_SEMI }}>
            {contentInfo.latestCommit?.author?.identity?.name || contentInfo.latestCommit?.author?.identity?.email}
          </Text>
          <Link to="">{contentInfo.latestCommit?.title}</Link>
          <FlexExpander />
          <Button
            className={css.shaBtn}
            text={contentInfo.latestCommit?.sha?.substring(0, 6)}
            variation={ButtonVariation.SECONDARY}
            size={ButtonSize.SMALL}
          />
          <Text font={{ variation: FontVariation.SMALL }} color={Color.GREY_400}>
            <ReactTimeago date={contentInfo.latestCommit?.author?.when as string} />
          </Text>
        </Layout.Horizontal>
      </Container>

      <Table<OpenapiContentInfo>
        className={css.table}
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
