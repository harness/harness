import React, { useState } from 'react'
import { Container, Icon, Text } from '@harness/uicore'
import { useGet } from 'restful-react'
import cx from 'classnames'
import { ThreadSection } from 'components/ThreadSection/ThreadSection'
import { LIST_FETCHING_LIMIT, RenameDetails } from 'utils/Utils'
import { usePageIndex } from 'hooks/usePageIndex'
import type { TypesCommit, TypesRepository } from 'services/code'
import { useStrings } from 'framework/strings'
import { CommitsView } from 'components/CommitsView/CommitsView'
import { ResourceListingPagination } from 'components/ResourceListingPagination/ResourceListingPagination'

import css from './RenameContentHistory.module.scss'

const SingleFileRenameHistory = (props: {
  details: RenameDetails
  fileVisibility: { [key: string]: boolean }
  setFileVisibility: React.Dispatch<React.SetStateAction<{ [key: string]: boolean }>>
  repoMetadata: TypesRepository
  page: number
  response: any
  setPage: React.Dispatch<React.SetStateAction<number>>
}) => {
  const { details, fileVisibility, setFileVisibility, repoMetadata, page, response, setPage } = props
  const { getString } = useStrings()
  const { data: commits, refetch: getCommitHistory } = useGet<{
    commits: TypesCommit[]
    rename_details: RenameDetails[]
  }>({
    path: `/api/v1/repos/${repoMetadata?.path}/+/commitsV2`,
    lazy: true
  })

  const toggleCommitHistory = async () => {
    setFileVisibility(prevVisibility => ({
      ...prevVisibility,
      [details.old_path]: !prevVisibility[details.old_path]
    }))

    if (!fileVisibility[details.old_path]) {
      await getCommitHistory({
        queryParams: {
          limit: LIST_FETCHING_LIMIT,
          page,
          git_ref: details.commit_sha_before,
          path: details.old_path
        }
      })
    }
  }

  const isFileShown = fileVisibility[details.old_path]
  const commitsData = commits?.commits
  const showCommitHistory = isFileShown && commitsData && commitsData.length > 0

  return (
    <ThreadSection
      hideGutter
      hideTitleGutter
      contentClassName={css.contentSection}
      title={
        <Text padding={{top:"large"}} hidden={showCommitHistory} className={cx(css.hideText, css.lineDiv)} onClick={toggleCommitHistory}>
          {showCommitHistory ? getString('hideCommitHistory', { file: details.old_path }) : getString('showCommitHistory', { file: details.old_path })} 
          {showCommitHistory ? <Icon padding={'xsmall'} name={'main-chevron-up'} size={8}></Icon> : <Icon padding={'xsmall'} name={'main-chevron-down'} size={8}></Icon>} 
        </Text>
      }
      onlyTitle={showCommitHistory}>
      {showCommitHistory && (
        <>
          <CommitsView
            commits={commits.commits}
            repoMetadata={repoMetadata}
            emptyTitle={getString('noCommits')}
            emptyMessage={getString('noCommitsMessage')}
            showFileHistoryIcons={true}
            resourcePath={details.old_path}
          />
          <Container className={css.lineDiv}>

          <Text
            className={cx(css.hideText,css.lineDiv)}
            padding={{ left: 'xxxlarge', right: 'xxxlarge', top: 'large' }}
            onClick={toggleCommitHistory}>
            {getString('hideCommitHistory', { file: details.old_path })}
            <Icon padding={'xsmall'} name={'main-chevron-up'} size={8}></Icon>
          </Text>
              </Container>
          <ResourceListingPagination response={response} page={page} setPage={setPage} />
          <AllFilesRenameHistory
            rename_details={commits.rename_details.filter(file => file.old_path !== details.old_path)}
            repoMetadata={repoMetadata}
            fileVisibility={fileVisibility}
            setFileVisibility={setFileVisibility}
          />
        </>
      )}
    </ThreadSection>
  )
}

const AllFilesRenameHistory = (props: {
  rename_details: RenameDetails[]
  repoMetadata: TypesRepository
  fileVisibility: { [key: string]: boolean }
  setFileVisibility: React.Dispatch<React.SetStateAction<{ [key: string]: boolean }>>
}) => {
  const { rename_details, repoMetadata, fileVisibility, setFileVisibility } = props
  const [page, setPage] = usePageIndex()
  const { data: commits, response } = useGet<{ commits: TypesCommit[]; rename_details: RenameDetails[] }>({
    path: `/api/v1/repos/${repoMetadata?.path}/+/commitsV2`,
    lazy: true
  })

  return (
    <>
      {rename_details.map((details, index) => (
        <SingleFileRenameHistory
          key={index}
          details={details}
          fileVisibility={fileVisibility}
          setFileVisibility={setFileVisibility}
          repoMetadata={repoMetadata}
          page={page}
          response={response}
          setPage={setPage}
        />
      ))}
    </>
  )
}

const RenameContentHistory = (props: { rename_details: RenameDetails[]; repoMetadata: TypesRepository }) => {
  const { rename_details, repoMetadata } = props
  const [fileVisibility, setFileVisibility] = useState({})

  return (
    <AllFilesRenameHistory
      rename_details={rename_details}
      repoMetadata={repoMetadata}
      fileVisibility={fileVisibility}
      setFileVisibility={setFileVisibility}
    />
  )
}

export default RenameContentHistory
