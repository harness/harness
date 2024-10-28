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

import React, { useEffect, useMemo, useState } from 'react'
import {
  Container,
  Layout,
  FlexExpander,
  Text,
  Avatar,
  Utils,
  Tag,
  useIsMounted,
  StringSubstitute,
  useToaster
} from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'
import { Link } from 'react-router-dom'
import { Render } from 'react-jsx-match'
import cx from 'classnames'
import { defaultTo } from 'lodash-es'
import { GitCommit } from 'iconoir-react'
import { Icon } from '@harnessio/icons'
import { useMutate } from 'restful-react'
import type {
  OpenapiCalculateCommitDivergenceRequest,
  TypesCommitDivergence,
  TypesCommit,
  RepoRepositoryOutput
} from 'services/code'
import { CommitActions } from 'components/CommitActions/CommitActions'
import { useAppContext } from 'AppContext'
import { formatBytes, getErrorMessage } from 'utils/Utils'
import { useStrings } from 'framework/strings'
import { makeDiffRefs, type GitInfoProps, type RepositorySummaryData, isRefATag } from 'utils/GitUtils'
import { PipeSeparator } from 'components/PipeSeparator/PipeSeparator'
import { TimePopoverWithLocal } from 'utils/timePopoverLocal/TimePopoverWithLocal'
import css from './LatestCommit.module.scss'

interface LatestCommitProps extends Pick<GitInfoProps, 'repoMetadata' | 'gitRef'> {
  latestCommit?: TypesCommit
  standaloneStyle?: boolean
  size?: number
  repoSummaryData?: RepositorySummaryData | null
  loadingSummaryData?: boolean
}

interface DivergenceInfoProps {
  commitDivergence: TypesCommitDivergence
  metadata: RepoRepositoryOutput
  currentGitRef: string
}

export function LatestCommitForFolder({
  repoMetadata,
  latestCommit,
  standaloneStyle,
  gitRef,
  repoSummaryData,
  loadingSummaryData
}: LatestCommitProps) {
  const { routes } = useAppContext()
  const { getString } = useStrings()
  const { showError } = useToaster()
  const [divergence, setDivergence] = useState<TypesCommitDivergence>({})

  const commitURL = routes.toCODECommit({
    repoPath: repoMetadata.path as string,
    commitRef: latestCommit?.sha as string
  })

  const commitPage = routes.toCODECommits({
    repoPath: repoMetadata.path as string,
    commitRef: gitRef as string
  })

  const compareCommits = (target: string, source: string) =>
    routes.toCODECompare({
      repoPath: repoMetadata?.path as string,
      diffRefs: makeDiffRefs(target as string, source as string)
    })

  const { mutate: getBranchDivergence, loading: divergenceLoading } = useMutate({
    verb: 'POST',
    path: `/api/v1/repos/${repoMetadata.path}/+/commits/calculate-divergence`
  })

  const branchDivergenceRequestBody: OpenapiCalculateCommitDivergenceRequest = useMemo(() => {
    return {
      maxCount: 0,
      requests: [{ from: gitRef, to: repoMetadata.default_branch }]
    }
  }, [repoMetadata, gitRef])

  const isMounted = useIsMounted()

  useEffect(() => {
    if (isMounted.current && branchDivergenceRequestBody.requests?.length && gitRef !== repoMetadata.default_branch) {
      setDivergence({})
      getBranchDivergence(branchDivergenceRequestBody)
        .then(([response]: TypesCommitDivergence[]) => {
          if (isMounted.current) {
            setDivergence(response)
          }
        })
        .catch(error => {
          showError(getErrorMessage(error), 0, 'unableToGetDivergence')
        })
    }
  }, [getBranchDivergence, branchDivergenceRequestBody, isMounted])

  const currentBranchCommitCount =
    gitRef !== repoMetadata.default_branch &&
    (repoSummaryData?.default_branch_commit_count ?? 0) + (divergence?.ahead ?? 0) - (divergence?.behind ?? 0)

  const DivergenceInfo: React.FC<DivergenceInfoProps> = ({ commitDivergence, metadata, currentGitRef }) => {
    if ((commitDivergence?.ahead as number) > 0 && (commitDivergence?.behind as number) > 0) {
      return (
        <>
          <Link to={compareCommits(metadata.default_branch as string, currentGitRef)}>
            <Text className={css.link} lineClamp={1}>
              <StringSubstitute str={getString('aheadDivergence')} vars={{ aheadCommits: commitDivergence.ahead }} />
            </Text>
          </Link>
          <Text className={css.link} color={Color.GREY_500} lineClamp={1}>
            {getString('and')}
          </Text>
          <Link to={compareCommits(currentGitRef, metadata.default_branch as string)}>
            <Text className={css.link} lineClamp={1}>
              <StringSubstitute str={getString('behindDivergence')} vars={{ behindCommits: commitDivergence.behind }} />
            </Text>
          </Link>
        </>
      )
    }
    if ((commitDivergence?.ahead as number) > 0) {
      return (
        <Link to={compareCommits(metadata.default_branch as string, currentGitRef)}>
          <Text className={css.link} lineClamp={1}>
            <StringSubstitute str={getString('aheadDivergence')} vars={{ aheadCommits: commitDivergence.ahead }} />
          </Text>
        </Link>
      )
    }
    if ((commitDivergence?.behind as number) > 0) {
      return (
        <Link to={compareCommits(currentGitRef, metadata.default_branch as string)}>
          <Text className={css.link} lineClamp={1}>
            <StringSubstitute str={getString('behindDivergence')} vars={{ behindCommits: commitDivergence.behind }} />
          </Text>
        </Link>
      )
    }
    return (
      <Text className={css.link} lineClamp={1}>
        {getString('branchUpToDateWith')}
      </Text>
    )
  }

  return (
    <Render when={latestCommit}>
      <Container>
        <Layout.Vertical className={cx(css.latestCommit, { [css.standalone]: standaloneStyle })}>
          <Render when={gitRef !== repoMetadata.default_branch && !loadingSummaryData}>
            <Layout.Horizontal
              spacing="small"
              padding={{ bottom: 'small' }}
              className={cx(css.border)}
              flex={{ alignItems: 'center' }}>
              <GitCommit
                height={20}
                width={20}
                color={Utils.getRealCSSColor(Color.GREY_500)}
                className={css.commitIcon}
              />
              <Text className={css.noWrap} font={{ variation: FontVariation.SMALL_SEMI }} color={Color.GREY_500}>
                <StringSubstitute str={getString('thisRefHas')} vars={{ isTag: isRefATag(gitRef) }} />
              </Text>
              <Link to={commitPage}>
                <Text className={css.link} lineClamp={1}>
                  <StringSubstitute
                    str={getString('branchCommitCount')}
                    vars={{
                      count: currentBranchCommitCount
                    }}
                  />
                </Text>
              </Link>
              <FlexExpander />
              <Render when={!isRefATag(gitRef) && !divergenceLoading}>
                <>
                  <Layout.Horizontal spacing={'xsmall'}>
                    <DivergenceInfo commitDivergence={divergence} metadata={repoMetadata} currentGitRef={gitRef} />
                  </Layout.Horizontal>
                  <Tag className={css.tag} minimal>
                    <Icon name="code-branch" />
                    {repoMetadata?.default_branch}
                  </Tag>
                </>
              </Render>
            </Layout.Horizontal>
          </Render>
          <Layout.Horizontal spacing="small" flex={{ alignItems: 'center' }}>
            <Avatar hoverCard={false} size="small" name={latestCommit?.author?.identity?.name || ''} />
            <Text className={css.noWrap} font={{ variation: FontVariation.SMALL_BOLD }}>
              {latestCommit?.author?.identity?.name || latestCommit?.author?.identity?.email}
            </Text>
            <Link to={commitURL}>
              <Text className={css.link} lineClamp={1}>
                {latestCommit?.title}
              </Text>
            </Link>
            <FlexExpander />
            <CommitActions sha={latestCommit?.sha as string} href={commitURL} enableCopy />
            <TimePopoverWithLocal
              time={defaultTo(latestCommit?.committer?.when as unknown as number, 0)}
              inline={false}
              className={css.time}
              font={{ variation: FontVariation.SMALL }}
              color={Color.GREY_400}
            />
          </Layout.Horizontal>
        </Layout.Vertical>
      </Container>
    </Render>
  )
}

export function LatestCommitForFile({ repoMetadata, latestCommit, standaloneStyle, size }: LatestCommitProps) {
  const { routes } = useAppContext()
  const commitURL = routes.toCODECommit({
    repoPath: repoMetadata.path as string,
    commitRef: latestCommit?.sha as string
  })

  return (
    <Render when={latestCommit}>
      <Container>
        <Layout.Horizontal
          spacing="medium"
          className={cx(css.latestCommit, css.forFile, { [css.standalone]: standaloneStyle })}>
          <Avatar hoverCard={false} size="small" name={latestCommit?.author?.identity?.name || ''} />
          <Text font={{ variation: FontVariation.SMALL_BOLD }} className={css.noWrap}>
            {latestCommit?.author?.identity?.name || latestCommit?.author?.identity?.email}
          </Text>
          <PipeSeparator height={9} />

          <Text lineClamp={1} tooltipProps={{ portalClassName: css.popover }}>
            <Link to={commitURL} className={css.link}>
              {latestCommit?.title}
            </Link>
          </Text>
          <PipeSeparator height={9} />
          <TimePopoverWithLocal
            time={defaultTo(latestCommit?.committer?.when as unknown as number, 0)}
            inline={false}
            className={css.time}
            font={{ variation: FontVariation.SMALL }}
            color={Color.GREY_400}
          />
          {(size && size > 0 && (
            <>
              <PipeSeparator height={9} />
              <Text font={{ variation: FontVariation.SMALL }} color={Color.GREY_400} className={css.noWrap}>
                {formatBytes(size)}
              </Text>
            </>
          )) ||
            ''}

          <FlexExpander />
          <CommitActions sha={latestCommit?.sha as string} href={commitURL} enableCopy />
        </Layout.Horizontal>
      </Container>
    </Render>
  )
}
