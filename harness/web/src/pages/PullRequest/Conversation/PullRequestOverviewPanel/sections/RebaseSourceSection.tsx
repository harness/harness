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
import { Color, FontVariation } from '@harnessio/design-system'
import cx from 'classnames'
import {
  Button,
  ButtonSize,
  ButtonVariation,
  Container,
  Layout,
  StringSubstitute,
  Text,
  useToaster
} from '@harnessio/uicore'
import { useMutate } from 'restful-react'
import type { RebaseBranchRequestBody, RepoRepositoryOutput, TypesPullReq } from 'services/code'
import { useStrings } from 'framework/strings'
import { GitRefLink } from 'components/GitRefLink/GitRefLink'
import { getErrorMessage, permissionProps } from 'utils/Utils'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { useAppContext } from 'AppContext'
import FailRed from '../../../../../icons/code-fail.svg?url'
import css from '../PullRequestOverviewPanel.module.scss'

interface RebaseSourceSectionProps {
  pullReqMetadata: TypesPullReq
  repoMetadata: RepoRepositoryOutput
  refetchActivities: () => void
}

const RebaseSourceSection = (props: RebaseSourceSectionProps) => {
  const { pullReqMetadata, repoMetadata, refetchActivities } = props
  const { getString } = useStrings()
  const { showSuccess, showError } = useToaster()
  const { mutate: rebase } = useMutate<RebaseBranchRequestBody>({
    verb: 'POST',
    path: `/api/v1/repos/${repoMetadata.path}/+/rebase`
  })
  const {
    hooks: { usePermissionTranslate },
    standalone,
    routes
  } = useAppContext()
  const space = useGetSpaceParam()
  const permPushResult = usePermissionTranslate(
    {
      resource: {
        resourceType: 'CODE_REPOSITORY',
        resourceIdentifier: repoMetadata?.identifier as string
      },
      permissions: ['code_repo_push']
    },
    [space]
  )

  const rebaseRequestPayload = {
    base_branch: pullReqMetadata.target_branch,
    bypass_rules: true,
    dry_run_rules: false,
    head_branch: pullReqMetadata.source_branch,
    head_commit_sha: pullReqMetadata.source_sha
  }

  return (
    <>
      <Container className={cx(css.sectionContainer, css.borderRadius)}>
        <Layout.Horizontal flex={{ justifyContent: 'space-between' }}>
          <Layout.Horizontal flex={{ alignItems: 'center' }}>
            <img alt={getString('failed')} width={26} height={26} src={FailRed} />
            <Layout.Vertical padding={{ left: 'medium' }}>
              <Text padding={{ bottom: 'xsmall' }} className={css.sectionTitle} color={Color.RED_500}>
                {getString('rebaseSource.title')}
              </Text>
              <Text className={css.sectionSubheader} color={Color.GREY_450} font={{ variation: FontVariation.BODY }}>
                <StringSubstitute
                  str={getString('rebaseSource.message')}
                  vars={{
                    target: (
                      <GitRefLink
                        text={pullReqMetadata.target_branch as string}
                        url={routes.toCODERepository({
                          repoPath: repoMetadata.path as string,
                          gitRef: pullReqMetadata.target_branch
                        })}
                        showCopy
                      />
                    ),
                    source: (
                      <GitRefLink
                        text={pullReqMetadata.source_branch as string}
                        url={routes.toCODERepository({
                          repoPath: repoMetadata.path as string,
                          gitRef: pullReqMetadata.source_branch
                        })}
                        showCopy
                      />
                    )
                  }}
                />
              </Text>
            </Layout.Vertical>
          </Layout.Horizontal>

          <Button
            className={cx(css.blueTextColor)}
            variation={ButtonVariation.TERTIARY}
            size={ButtonSize.MEDIUM}
            text={getString('updateWithRebase')}
            onClick={() =>
              rebase(rebaseRequestPayload)
                .then(() => {
                  showSuccess(getString('updatedBranchMessageRebase'))
                  setTimeout(() => {
                    refetchActivities()
                  }, 1000)
                })
                .catch(err => showError(getErrorMessage(err)))
            }
            {...permissionProps(permPushResult, standalone)}
          />
        </Layout.Horizontal>
      </Container>
    </>
  )
}

export default RebaseSourceSection
