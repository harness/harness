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
import cx from 'classnames'
import { Button, ButtonVariation, Container, Layout, StringSubstitute, Text, useToaster } from '@harnessio/uicore'
import type { MutateMethod } from 'restful-react'
import { Icon } from '@harnessio/icons'
import { useStrings } from 'framework/strings'
import { CodeIcon } from 'utils/GitUtils'
import { getErrorMessage } from 'utils/Utils'
import type {
  CreateBranchPathParams,
  DeletePullReqSourceBranchQueryParams,
  OpenapiCreateBranchRequest
} from 'services/code'
import css from '../PullRequestOverviewPanel.module.scss'

interface BranchActionsSectionProps {
  sourceSha: string
  sourceBranch: string
  createBranch: MutateMethod<any, any, OpenapiCreateBranchRequest, CreateBranchPathParams>
  refetchActivities: () => void
  refetchBranch: () => Promise<void>
  deleteBranch: MutateMethod<any, any, DeletePullReqSourceBranchQueryParams, unknown>
  showDeleteBranchButton: boolean
  setShowDeleteBranchButton: React.Dispatch<React.SetStateAction<boolean>>
  setShowRestoreBranchButton: React.Dispatch<React.SetStateAction<boolean>>
  setIsSourceBranchDeleted?: React.Dispatch<React.SetStateAction<boolean>>
}

const BranchActionsSection = (props: BranchActionsSectionProps) => {
  const { getString } = useStrings()

  return (
    <Container className={cx(css.branchActionsSectionContainer, css.borderRadius)} padding={{ right: 'xlarge' }}>
      <Layout.Horizontal flex={{ justifyContent: 'space-between' }}>
        <Text flex={{ alignItems: 'center' }}>
          <StringSubstitute
            str={getString('pr.closedPrBranchDelete')}
            vars={{
              source: (
                <Container padding={{ left: 'small', right: 'small' }}>
                  <strong className={cx(css.boldText, css.branchContainer)}>
                    <Icon name={CodeIcon.Branch} size={16} />
                    <Text className={cx(css.boldText, css.widthContainer)} lineClamp={1}>
                      {props.sourceBranch}
                    </Text>
                  </strong>
                </Container>
              )
            }}
          />
        </Text>
        <BranchActionsButton {...props} />
      </Layout.Horizontal>
    </Container>
  )
}

export const BranchActionsButton = ({
  sourceSha,
  sourceBranch,
  createBranch,
  refetchActivities,
  refetchBranch,
  deleteBranch,
  showDeleteBranchButton,
  setShowRestoreBranchButton,
  setShowDeleteBranchButton,
  setIsSourceBranchDeleted
}: BranchActionsSectionProps) => {
  const { getString } = useStrings()
  const { showSuccess, showError } = useToaster()

  return (
    <Button
      style={{ whiteSpace: 'nowrap' }}
      text={showDeleteBranchButton ? getString('deleteBranch') : getString('restoreBranch')}
      variation={ButtonVariation.SECONDARY}
      onClick={() => {
        showDeleteBranchButton
          ? deleteBranch({}, { queryParams: { bypass_rules: true, dry_run_rules: false } })
              .then(() => {
                refetchBranch()
                setIsSourceBranchDeleted?.(true)
                setShowDeleteBranchButton(false)
                refetchActivities()
                showSuccess(
                  <StringSubstitute
                    str={getString('branchDeleted')}
                    vars={{
                      branch: sourceBranch
                    }}
                  />,
                  5000
                )
              })
              .catch(err => showError(getErrorMessage(err)))
          : createBranch({ name: sourceBranch, target: sourceSha, bypass_rules: true })
              .then(() => {
                refetchBranch()
                setIsSourceBranchDeleted?.(false)
                setShowRestoreBranchButton(false)
                refetchActivities()
                showSuccess(
                  <StringSubstitute
                    str={getString('branchRestored')}
                    vars={{
                      branch: sourceBranch
                    }}
                  />,
                  5000
                )
              })
              .catch(err => showError(getErrorMessage(err)))
      }}
    />
  )
}

export default BranchActionsSection
