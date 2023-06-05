import React, { useState } from 'react'
import { Text } from '@harness/uicore'
import { useGet } from 'restful-react'
import { ThreadSection } from 'components/ThreadSection/ThreadSection'
import { LIST_FETCHING_LIMIT, RenameDetails } from 'utils/Utils'
import { usePageIndex } from 'hooks/usePageIndex'
import type { TypesCommit, TypesRepository } from 'services/code'
import { useStrings } from 'framework/strings'
import { CommitsView } from 'components/CommitsView/CommitsView'
import { ResourceListingPagination } from 'components/ResourceListingPagination/ResourceListingPagination'

import css from './RenameContentHistory.module.scss'

const RenameContentHistory = (props: { rename_details: RenameDetails[], repoMetadata: TypesRepository, fileVisibility?: { [key: string]: boolean } }) => {
  const { rename_details, repoMetadata, fileVisibility: initialFileVisibility } = props;
  const { getString } = useStrings();
  const [fileVisibility, setFileVisibility] = useState(initialFileVisibility || {});
  const [page, setPage] = usePageIndex();
  const { data: commits, response, refetch: getCommitHistory } = useGet<{ commits: TypesCommit[]; rename_details: RenameDetails[] }>({
    path: `/api/v1/repos/${repoMetadata?.path}/+/commits`,
    lazy: true
  });

  const toggleCommitHistory = async (details: RenameDetails) => {
    setFileVisibility(prevVisibility => ({
      ...prevVisibility,
      [details.old_path]: !prevVisibility[details.old_path]
    }));

    if (!fileVisibility[details.old_path]) {
      await getCommitHistory({
        queryParams: {
          limit: LIST_FETCHING_LIMIT,
          page,
          git_ref: details.commit_sha_before,
          path: details.old_path
        }
      });
    }
  };

  return (
    <>
      {rename_details.map((details, index) => {
        const isFileShown = fileVisibility[details.old_path];
        const commitsData = commits?.commits;
        const showCommitHistory = isFileShown && commitsData && commitsData.length > 0;

        return (
          <ThreadSection
            key={index}
            hideGutter
            hideTitleGutter
            contentClassName={css.contentSection}
            title={
              <Text
                hidden={showCommitHistory}
                className={css.hideText}
                padding={{top:"large"}}
                onClick={() => toggleCommitHistory(details)}
              >
                {showCommitHistory ?getString('hideCommitHistory',{file:details.old_path})  :getString('showCommitHistory',{file:details.old_path})} 
              </Text>
            }
            onlyTitle={showCommitHistory}
          >
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
                <Text
                  className={css.hideText}
                  padding={{ left: 'xxxlarge', right: 'xxxlarge', top: 'large' }}
                  onClick={() => toggleCommitHistory(details)}
                >
                  {getString('hideCommitHistory',{file:details.old_path})}
                </Text>
                <ResourceListingPagination response={response} page={page} setPage={setPage} />
                <RenameContentHistory
                  rename_details={commits.rename_details.filter(file => file.old_path !== details.old_path)}
                  repoMetadata={repoMetadata}
                  fileVisibility={fileVisibility}
                />
              </>
            )}
          </ThreadSection>
        );
      })}
    </>
  );
};

export default RenameContentHistory;
