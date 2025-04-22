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
import { PopoverPosition } from '@blueprintjs/core'
import { Button, ButtonProps, ButtonVariation, useToaster } from '@harnessio/uicore'
import { useMutate } from 'restful-react'
import { useHistory } from 'react-router-dom'
import { useStrings } from 'framework/strings'
import { getErrorMessage } from 'utils/Utils'
import { useAppContext } from 'AppContext'
import { makeDiffRefs } from 'utils/GitUtils'
import type { RepoRepositoryOutput, TypesPullReq } from 'services/code'

export interface RevertPRButtonProps extends ButtonProps {
  pullRequestMetadata: TypesPullReq
  repoMetadata: RepoRepositoryOutput
}

export const RevertPRButton: React.FC<RevertPRButtonProps> = ({ pullRequestMetadata, repoMetadata, ...props }) => {
  const { getString } = useStrings()
  const { showError, showSuccess } = useToaster()
  const { routes } = useAppContext()
  const history = useHistory()
  const { mutate: revertPR } = useMutate({
    verb: 'POST',
    path: `/api/v1/repos/${repoMetadata.path}/+/pullreq/${pullRequestMetadata.number}/revert`
  })
  return (
    <Button
      style={{ whiteSpace: 'nowrap' }}
      text={'Revert'}
      variation={ButtonVariation.SECONDARY}
      onClick={() => {
        revertPR({})
          .then(data => {
            history.push(
              routes.toCODECompare({
                repoPath: repoMetadata?.path as string,
                diffRefs: makeDiffRefs(pullRequestMetadata?.target_branch as string, data.branch)
              })
            )
            showSuccess(getString('pr.revertBranchSuccess', { branch: data.branch }), 3000)
          })
          .catch(err => {
            if (err.status === 400) {
              const match = err.data.message.match(revertBranchExistsRegex)
              if (match) {
                const branchName = match[1]
                history.push(
                  routes.toCODECompare({
                    repoPath: repoMetadata?.path as string,
                    diffRefs: makeDiffRefs(pullRequestMetadata?.target_branch as string, branchName)
                  })
                )
              }
            } else {
              showError(getErrorMessage(err), 0, getString('pr.revertOpFailed'))
            }
          })
      }}
      tooltip={getString('pr.createRevertPR')}
      tooltipProps={{
        interactionKind: 'hover',
        usePortal: true,
        position: PopoverPosition.TOP_RIGHT
      }}
      {...props}
    />
  )
}

const revertBranchExistsRegex = /Branch\s+"([^"]+)"\s+already exists\./
