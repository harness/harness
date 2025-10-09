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

import React, { useState } from 'react'
import { Text, StringSubstitute } from '@harnessio/uicore'
import { Icon } from '@harnessio/icons'
import { useGet } from 'restful-react'
import cx from 'classnames'
import { ThreadSection } from 'components/ThreadSection/ThreadSection'
import { LIST_FETCHING_LIMIT, RenameDetails } from 'utils/Utils'
import { usePageIndex } from 'hooks/usePageIndex'
import type { TypesCommit, RepoRepositoryOutput } from 'services/code'
import { useStrings } from 'framework/strings'
import { CommitsView } from 'components/CommitsView/CommitsView'
import { ResourceListingPagination } from 'components/ResourceListingPagination/ResourceListingPagination'
import { normalizeGitRef } from 'utils/GitUtils'
import css from './RenameContentHistory.module.scss'

const SingleFileRenameHistory = (props: {
  details: RenameDetails
  fileVisibility: { [key: string]: boolean }
  setFileVisibility: React.Dispatch<React.SetStateAction<{ [key: string]: boolean }>>
  repoMetadata: RepoRepositoryOutput
  page: number
  /* eslint-disable @typescript-eslint/no-explicit-any */
  response: any
  setPage: React.Dispatch<React.SetStateAction<number>>
  setActiveTab: React.Dispatch<React.SetStateAction<string>>
}) => {
  const { details, fileVisibility, setFileVisibility, repoMetadata, page, response, setPage, setActiveTab } = props
  const { getString } = useStrings()
  const { data: commits, refetch: getCommitHistory } = useGet<{
    commits: TypesCommit[]
    rename_details: RenameDetails[]
  }>({
    path: `/api/v1/repos/${repoMetadata?.path}/+/commits`,
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
          git_ref: normalizeGitRef(details.commit_sha_before),
          path: details.old_path
        }
      })
    }
  }

  const isFileShown = fileVisibility[details.old_path]
  const commitsData = commits?.commits
  const showCommitHistory = isFileShown && commitsData && commitsData.length > 0

  const vars = {
    file: details.old_path
  }
  return (
    <ThreadSection
      hideGutter
      hideTitleGutter
      contentClassName={css.contentSection}
      title={
        <Text padding={{ top: 'large' }} className={cx(css.hideText, css.lineDiv)} onClick={toggleCommitHistory}>
          <StringSubstitute
            str={showCommitHistory ? getString('hideCommitHistory') : getString('showCommitHistory')}
            vars={vars}
          />

          {showCommitHistory ? (
            <Icon padding={'xsmall'} name={'main-chevron-up'} size={8}></Icon>
          ) : (
            <Icon padding={'xsmall'} name={'main-chevron-down'} size={8}></Icon>
          )}
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
            setActiveTab={setActiveTab}
          />
          <ResourceListingPagination response={response} page={page} setPage={setPage} />
          <AllFilesRenameHistory
            rename_details={commits.rename_details.filter(file => file.old_path !== details.old_path)}
            repoMetadata={repoMetadata}
            fileVisibility={fileVisibility}
            setFileVisibility={setFileVisibility}
            setActiveTab={setActiveTab}
          />
        </>
      )}
    </ThreadSection>
  )
}

const AllFilesRenameHistory = (props: {
  rename_details: RenameDetails[]
  repoMetadata: RepoRepositoryOutput
  fileVisibility: { [key: string]: boolean }
  setFileVisibility: React.Dispatch<React.SetStateAction<{ [key: string]: boolean }>>
  setActiveTab: React.Dispatch<React.SetStateAction<string>>
}) => {
  const { rename_details, repoMetadata, fileVisibility, setFileVisibility, setActiveTab } = props
  const [page, setPage] = usePageIndex()
  const { response } = useGet<{ commits: TypesCommit[]; rename_details: RenameDetails[] }>({
    path: `/api/v1/repos/${repoMetadata?.path}/+/commits`,
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
          setActiveTab={setActiveTab}
        />
      ))}
    </>
  )
}

const RenameContentHistory = (props: {
  rename_details: RenameDetails[]
  repoMetadata: RepoRepositoryOutput
  setActiveTab: React.Dispatch<React.SetStateAction<string>>
}) => {
  const { rename_details, repoMetadata, setActiveTab } = props
  const [fileVisibility, setFileVisibility] = useState({})

  return (
    <AllFilesRenameHistory
      rename_details={rename_details}
      repoMetadata={repoMetadata}
      fileVisibility={fileVisibility}
      setFileVisibility={setFileVisibility}
      setActiveTab={setActiveTab}
    />
  )
}

export default RenameContentHistory
