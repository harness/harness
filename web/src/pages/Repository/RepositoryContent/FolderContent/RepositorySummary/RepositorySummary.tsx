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

import React from 'react'
import { Button, ButtonVariation, Container, Layout, Text, Utils } from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'
import { GitCommit, GitFork, Label, GitPullRequest } from 'iconoir-react'
import { useHistory } from 'react-router-dom'
import { useAppContext } from 'AppContext'
import { useStrings } from 'framework/strings'
import { CodeIcon, RepositorySummaryData } from 'utils/GitUtils'
import type { RepoRepositoryOutput } from 'services/code'
import { permissionProps, formatDate } from 'utils/Utils'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import css from './RepositorySummary.module.scss'

interface RepositorySummaryProps {
  metadata: RepoRepositoryOutput
  gitRef?: string
  repoSummaryData: RepositorySummaryData | null
  loadingSummaryData: boolean
}

enum MetaDataType {
  BRANCH = 'branch',
  TAG = 'tag',
  COMMIT = 'commit',
  PULL_REQUEST = 'pull_request'
}

interface MetaDataProps {
  type: MetaDataType
  text: string
  data: number | undefined
}

const RepositorySummary = (props: RepositorySummaryProps) => {
  const { metadata, repoSummaryData, loadingSummaryData } = props
  const { getString } = useStrings()
  const { standalone, hooks, routes } = useAppContext()
  const { space } = useGetRepositoryMetadata()

  const MetaData: React.FC<MetaDataProps> = ({ type, text, data }) => {
    let DataIcon
    let routeTo: string
    const history = useHistory()
    switch (type) {
      case MetaDataType.BRANCH:
        DataIcon = GitFork
        routeTo = routes.toCODEBranches({
          repoPath: metadata?.path as string
        })
        break
      case MetaDataType.TAG:
        DataIcon = Label
        routeTo = routes.toCODETags({
          repoPath: metadata?.path as string
        })
        break
      case MetaDataType.COMMIT:
        DataIcon = GitCommit
        routeTo = routes.toCODECommits({
          repoPath: metadata?.path as string,
          commitRef: metadata?.default_branch as string
        })
        break
      case MetaDataType.PULL_REQUEST:
        DataIcon = GitPullRequest
        routeTo = routes.toCODEPullRequests({
          repoPath: metadata?.path as string
        })
        break
      default:
        DataIcon = GitCommit
    }

    return (
      <Layout.Horizontal className={css.metaData}>
        <Text color={Color.BLACK} font={{ variation: FontVariation.BODY2_SEMI }} className={css.align}>
          <DataIcon height={20} width={20} color={Utils.getRealCSSColor(Color.GREY_500)} />
          {text}
        </Text>
        <Button
          className={css.link}
          icon={loadingSummaryData ? 'steps-spinner' : undefined}
          variation={ButtonVariation.LINK}
          text={data?.toLocaleString()}
          onClick={() => history.push(routeTo)}
        />
      </Layout.Horizontal>
    )
  }

  const permPushResult = hooks?.usePermissionTranslate?.(
    {
      resource: {
        resourceType: 'CODE_REPOSITORY',
        resourceIdentifier: metadata?.identifier as string
      },
      permissions: ['code_repo_edit']
    },
    [space]
  )

  const history = useHistory()
  return (
    <Container padding={'medium'} background={Color.WHITE} className={css.summaryContainer}>
      <Container className={css.heading} padding={'small'}>
        <Text font={{ variation: FontVariation.H5 }} margin={{ bottom: 'xsmall' }}>
          {getString('summary')}
        </Text>
        <Text font={{ variation: FontVariation.SMALL_SEMI }} color={Color.GREY_450}>
          {getString('created')} {formatDate(metadata?.created || 0, 'long')}
        </Text>
      </Container>
      <Layout.Vertical padding={'small'} className={css.content}>
        <Layout.Vertical spacing="medium">
          <Text font={{ variation: FontVariation.BODY2_SEMI }} color={Color.BLACK_100} className={css.summaryDesc}>
            {metadata.description ? (
              metadata.description
            ) : (
              <Button
                variation={ButtonVariation.LINK}
                text={getString('addDescription')}
                icon={CodeIcon.Add}
                onClick={() =>
                  history.push(
                    routes.toCODESettings({
                      repoPath: metadata?.path as string
                    })
                  )
                }
                {...permissionProps(permPushResult, standalone)}
              />
            )}
          </Text>
        </Layout.Vertical>
        <Layout.Vertical spacing="large">
          <MetaData
            type={MetaDataType.COMMIT}
            text={getString('commits')}
            data={repoSummaryData?.default_branch_commit_count}
          />
          <MetaData type={MetaDataType.BRANCH} text={getString('branches')} data={repoSummaryData?.branch_count} />
          <MetaData type={MetaDataType.TAG} text={getString('tags')} data={repoSummaryData?.tag_count} />
          <MetaData
            type={MetaDataType.PULL_REQUEST}
            text={'Open Pull Requests'}
            data={repoSummaryData?.pull_req_summary.open_count}
          />
        </Layout.Vertical>
      </Layout.Vertical>
    </Container>
  )
}

export default RepositorySummary
